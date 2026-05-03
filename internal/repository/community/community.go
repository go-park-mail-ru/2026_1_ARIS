package community

//go:generate mockgen -destination=./../mocks/community_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/community CommunityRepo

import (
	"context"
	"errors"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	pgerrors "github.com/go-park-mail-ru/2026_1_ARIS/internal/utils/pg_errors"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

var ErrCommunityNotFound = errors.New("community not found")
var ErrCommunityMemberNotFound = errors.New("community member not found")

type CommunityRepo interface {
	Create(ctx context.Context, community models.Community, ownerProfileID int64, avatarID *int64) (*models.Community, error)
	Get(ctx context.Context, communityID int64) (*models.Community, error)
	GetByProfileID(ctx context.Context, profileID int64) (*models.Community, error)
	List(ctx context.Context, limit, offset int) ([]models.Community, error)
	Update(ctx context.Context, community models.Community) (*models.Community, error)
	UpdateAvatar(ctx context.Context, communityProfileID int64, avatarID *int64) error
	GetAvatarID(ctx context.Context, communityProfileID int64) (*int64, error)
	Delete(ctx context.Context, communityID int64) error
	GetMember(ctx context.Context, communityID, profileID int64) (*models.CommunityMember, error)
	ListMembers(ctx context.Context, communityID int64, includeBlocked bool) ([]models.CommunityMember, error)
	UpsertMemberRole(ctx context.Context, communityID, profileID int64, role models.CommunityMemberRole) (*models.CommunityMember, error)
	DeactivateMember(ctx context.Context, communityID, profileID int64) error
}

type communityStorage struct {
	db communityDB
}

type communityDB interface {
	Begin(context.Context) (pgx.Tx, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func NewCommunityStorage(db communityDB) CommunityRepo {
	return &communityStorage{db: db}
}

func (storage *communityStorage) Create(ctx context.Context, community models.Community, ownerProfileID int64, avatarID *int64) (*models.Community, error) {
	logQuery(ctx, "communityRepo.Create")
	tx, err := storage.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var profileID int64
	if err := tx.QueryRow(ctx, `INSERT INTO profile (uid, avatar_id) VALUES ($1, $2) RETURNING id`, uuid.New(), avatarID).Scan(&profileID); err != nil {
		return nil, err
	}

	var created models.Community
	query := `
		INSERT INTO community (uid, title, bio, community_type, profile_id, username, cover_media_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, uid, title, bio, community_type, profile_id, username, cover_media_id, is_active, created_at, updated_at`
	if err := pgxscan.Get(ctx, tx, &created, query, uuid.New(), community.Title, community.Bio, community.Type, profileID, community.Username, community.CoverMediaID); err != nil {
		return nil, pgerrors.MapPgError(err)
	}

	memberQuery := `
		INSERT INTO community_member (uid, profile_id, community_id, community_role)
		VALUES ($1, $2, $3, $4)`
	if _, err := tx.Exec(ctx, memberQuery, uuid.New(), ownerProfileID, created.ID, models.Owner); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &created, nil
}

func (storage *communityStorage) Get(ctx context.Context, communityID int64) (*models.Community, error) {
	query := communitySelect() + ` WHERE c.id=$1 AND c.is_active=TRUE`
	var community models.Community
	start := time.Now()
	err := pgxscan.Get(ctx, storage.db, &community, query, communityID)
	logDuration(ctx, "communityRepo.Get", start)
	if err != nil {
		return nil, normalizeCommunityError(err)
	}
	return &community, nil
}

func (storage *communityStorage) GetByProfileID(ctx context.Context, profileID int64) (*models.Community, error) {
	query := communitySelect() + ` WHERE c.profile_id=$1 AND c.is_active=TRUE`
	var community models.Community
	start := time.Now()
	err := pgxscan.Get(ctx, storage.db, &community, query, profileID)
	logDuration(ctx, "communityRepo.GetByProfileID", start)
	if err != nil {
		return nil, normalizeCommunityError(err)
	}
	return &community, nil
}

func (storage *communityStorage) List(ctx context.Context, limit, offset int) ([]models.Community, error) {
	query := communitySelect() + ` WHERE c.is_active=TRUE ORDER BY c.created_at DESC, c.id DESC LIMIT $1 OFFSET $2`
	start := time.Now()
	rows, err := storage.db.Query(ctx, query, limit, offset)
	logDuration(ctx, "communityRepo.List", start)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.Community])
}

func (storage *communityStorage) Update(ctx context.Context, community models.Community) (*models.Community, error) {
	query := `
		UPDATE community
		SET title=$1, bio=$2, community_type=$3, username=$4, cover_media_id=$5, updated_at=NOW()
		WHERE id=$6 AND is_active=TRUE
		RETURNING id, uid, title, bio, community_type, profile_id, username, cover_media_id, is_active, created_at, updated_at`
	var updated models.Community
	if err := pgxscan.Get(ctx, storage.db, &updated, query, community.Title, community.Bio, community.Type, community.Username, community.CoverMediaID, community.ID); err != nil {
		return nil, normalizeCommunityError(pgerrors.MapPgError(err))
	}
	return &updated, nil
}

func (storage *communityStorage) UpdateAvatar(ctx context.Context, communityProfileID int64, avatarID *int64) error {
	query := `UPDATE profile SET avatar_id=$1, updated_at=NOW() WHERE id=$2`
	start := time.Now()
	tag, err := storage.db.Exec(ctx, query, avatarID, communityProfileID)
	logDuration(ctx, "communityRepo.UpdateAvatar", start)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrCommunityNotFound
	}
	return nil
}

func (storage *communityStorage) GetAvatarID(ctx context.Context, communityProfileID int64) (*int64, error) {
	query := `SELECT avatar_id FROM profile WHERE id=$1 AND is_active=TRUE`
	var avatarID *int64
	start := time.Now()
	err := storage.db.QueryRow(ctx, query, communityProfileID).Scan(&avatarID)
	logDuration(ctx, "communityRepo.GetAvatarID", start)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCommunityNotFound
		}
		return nil, err
	}
	return avatarID, nil
}

func (storage *communityStorage) Delete(ctx context.Context, communityID int64) error {
	query := `UPDATE community SET is_active=FALSE, updated_at=NOW() WHERE id=$1 AND is_active=TRUE`
	start := time.Now()
	tag, err := storage.db.Exec(ctx, query, communityID)
	logDuration(ctx, "communityRepo.Delete", start)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrCommunityNotFound
	}
	return nil
}

func (storage *communityStorage) GetMember(ctx context.Context, communityID, profileID int64) (*models.CommunityMember, error) {
	query := `SELECT * FROM community_member WHERE community_id=$1 AND profile_id=$2 AND is_active=TRUE`
	var member models.CommunityMember
	start := time.Now()
	err := pgxscan.Get(ctx, storage.db, &member, query, communityID, profileID)
	logDuration(ctx, "communityRepo.GetMember", start)
	if err != nil {
		if pgxscan.NotFound(err) {
			return nil, ErrCommunityMemberNotFound
		}
		return nil, err
	}
	return &member, nil
}

func (storage *communityStorage) ListMembers(ctx context.Context, communityID int64, includeBlocked bool) ([]models.CommunityMember, error) {
	query := `SELECT * FROM community_member WHERE community_id=$1 AND is_active=TRUE`
	args := []any{communityID}
	if !includeBlocked {
		query += ` AND community_role <> $2`
		args = append(args, models.Blocked)
	}
	query += ` ORDER BY joined_at ASC, id ASC`

	start := time.Now()
	rows, err := storage.db.Query(ctx, query, args...)
	logDuration(ctx, "communityRepo.ListMembers", start)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return pgx.CollectRows(rows, pgx.RowToStructByName[models.CommunityMember])
}

func (storage *communityStorage) UpsertMemberRole(ctx context.Context, communityID, profileID int64, role models.CommunityMemberRole) (*models.CommunityMember, error) {
	query := `
		INSERT INTO community_member (uid, profile_id, community_id, community_role, is_active, leave_at)
		VALUES ($1, $2, $3, $4, TRUE, NULL)
		ON CONFLICT (profile_id, community_id)
		DO UPDATE SET community_role=EXCLUDED.community_role, is_active=TRUE, leave_at=NULL, updated_at=NOW()
		RETURNING *`
	var member models.CommunityMember
	start := time.Now()
	err := pgxscan.Get(ctx, storage.db, &member, query, uuid.New(), profileID, communityID, role)
	logDuration(ctx, "communityRepo.UpsertMemberRole", start)
	if err != nil {
		return nil, err
	}
	return &member, nil
}

func (storage *communityStorage) DeactivateMember(ctx context.Context, communityID, profileID int64) error {
	query := `
		UPDATE community_member
		SET is_active=FALSE, leave_at=NOW(), updated_at=NOW()
		WHERE community_id=$1 AND profile_id=$2 AND is_active=TRUE`
	start := time.Now()
	tag, err := storage.db.Exec(ctx, query, communityID, profileID)
	logDuration(ctx, "communityRepo.DeactivateMember", start)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrCommunityMemberNotFound
	}
	return nil
}

func communitySelect() string {
	return `
		SELECT c.id, c.uid, c.title, c.bio, c.community_type, c.profile_id, c.username, c.cover_media_id,
		       c.is_active, c.created_at, c.updated_at
		FROM community c`
}

func normalizeCommunityError(err error) error {
	if err == nil {
		return nil
	}
	if pgxscan.NotFound(err) {
		return ErrCommunityNotFound
	}
	return err
}

func logQuery(ctx context.Context, queryName string) {
	if log := logger.FromContext(ctx); log != nil {
		log.Debug("db query", zap.String("query", queryName))
	}
}

func logDuration(ctx context.Context, queryName string, start time.Time) {
	if log := logger.FromContext(ctx); log != nil {
		log.Debug("db query", zap.String("query", queryName), zap.Duration("duration_ms", time.Since(start)))
	}
}
