package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/community/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

var (
	ErrCommunityNotFound       = errors.New("community not found")
	ErrCommunityMemberNotFound = errors.New("community member not found")
	ErrDuplicateEntry          = errors.New("duplicate entry")
)

type DB interface {
	Begin(context.Context) (pgx.Tx, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type CommunityRepo interface {
	Create(ctx context.Context, community model.Community, ownerProfileID int64, avatarID *int64) (*model.Community, error)
	Get(ctx context.Context, communityID int64) (*model.Community, error)
	GetByProfileID(ctx context.Context, profileID int64) (*model.Community, error)
	List(ctx context.Context, limit, offset int) ([]model.Community, error)
	Update(ctx context.Context, community model.Community) (*model.Community, error)
	UpdateAvatar(ctx context.Context, communityProfileID int64, avatarID *int64) error
	GetAvatarID(ctx context.Context, communityProfileID int64) (*int64, error)
	Delete(ctx context.Context, communityID int64) error
	GetMember(ctx context.Context, communityID, profileID int64) (*model.CommunityMember, error)
	ListMembers(ctx context.Context, communityID int64, includeBlocked bool) ([]model.CommunityMember, error)
	UpsertMemberRole(ctx context.Context, communityID, profileID int64, role model.CommunityMemberRole) (*model.CommunityMember, error)
	DeactivateMember(ctx context.Context, communityID, profileID int64) error
	Search(ctx context.Context, query string, limit int) ([]SearchCommunityResult, error)
}

type SearchCommunityResult struct {
	ID           int64               `db:"id"`
	ProfileID    int64               `db:"profile_id"`
	Username     string              `db:"username"`
	Title        string              `db:"title"`
	Bio          *string             `db:"bio"`
	Type         model.CommunityType `db:"community_type"`
	AvatarID     *int64              `db:"avatar_id"`
	CoverMediaID *int64              `db:"cover_media_id"`
}

type communityStorage struct {
	db DB
}

func NewCommunityStorage(db DB) CommunityRepo {
	return &communityStorage{db: db}
}

func (s *communityStorage) Create(ctx context.Context, community model.Community, ownerProfileID int64, avatarID *int64) (*model.Community, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var profileID int64
	if err := tx.QueryRow(ctx, `INSERT INTO profile (uid, avatar_id) VALUES ($1, $2) RETURNING id`, uuid.New(), avatarID).Scan(&profileID); err != nil {
		return nil, err
	}

	var created model.Community
	err = pgxscan.Get(ctx, tx, &created, `
		INSERT INTO community (uid, title, bio, community_type, profile_id, username, cover_media_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, uid, title, bio, community_type, profile_id, username, cover_media_id, is_active, created_at, updated_at
	`, uuid.New(), community.Title, community.Bio, community.Type, profileID, community.Username, community.CoverMediaID)
	if err != nil {
		return nil, mapPgError(err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO community_member (uid, profile_id, community_id, community_role)
		VALUES ($1, $2, $3, $4)
	`, uuid.New(), ownerProfileID, created.ID, model.Owner); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &created, nil
}

func (s *communityStorage) Get(ctx context.Context, communityID int64) (*model.Community, error) {
	var community model.Community
	start := time.Now()
	err := pgxscan.Get(ctx, s.db, &community, communitySelect()+` WHERE c.id=$1 AND c.is_active=TRUE`, communityID)
	logQuery(ctx, "communityStorage.Get", start)
	if err != nil {
		return nil, normalizeCommunityError(err)
	}
	return &community, nil
}

func (s *communityStorage) GetByProfileID(ctx context.Context, profileID int64) (*model.Community, error) {
	var community model.Community
	start := time.Now()
	err := pgxscan.Get(ctx, s.db, &community, communitySelect()+` WHERE c.profile_id=$1 AND c.is_active=TRUE`, profileID)
	logQuery(ctx, "communityStorage.GetByProfileID", start)
	if err != nil {
		return nil, normalizeCommunityError(err)
	}
	return &community, nil
}

func (s *communityStorage) List(ctx context.Context, limit, offset int) ([]model.Community, error) {
	start := time.Now()
	rows, err := s.db.Query(ctx, communitySelect()+` WHERE c.is_active=TRUE ORDER BY c.created_at DESC, c.id DESC LIMIT $1 OFFSET $2`, limit, offset)
	logQuery(ctx, "communityStorage.List", start)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return pgx.CollectRows(rows, pgx.RowToStructByName[model.Community])
}

func (s *communityStorage) Update(ctx context.Context, community model.Community) (*model.Community, error) {
	var updated model.Community
	start := time.Now()
	err := pgxscan.Get(ctx, s.db, &updated, `
		UPDATE community
		SET title=$1, bio=$2, community_type=$3, username=$4, cover_media_id=$5, updated_at=NOW()
		WHERE id=$6 AND is_active=TRUE
		RETURNING id, uid, title, bio, community_type, profile_id, username, cover_media_id, is_active, created_at, updated_at
	`, community.Title, community.Bio, community.Type, community.Username, community.CoverMediaID, community.ID)
	logQuery(ctx, "communityStorage.Update", start)
	if err != nil {
		return nil, normalizeCommunityError(mapPgError(err))
	}
	return &updated, nil
}

func (s *communityStorage) UpdateAvatar(ctx context.Context, communityProfileID int64, avatarID *int64) error {
	start := time.Now()
	tag, err := s.db.Exec(ctx, `UPDATE profile SET avatar_id=$1, updated_at=NOW() WHERE id=$2`, avatarID, communityProfileID)
	logQuery(ctx, "communityStorage.UpdateAvatar", start)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrCommunityNotFound
	}
	return nil
}

func (s *communityStorage) GetAvatarID(ctx context.Context, communityProfileID int64) (*int64, error) {
	start := time.Now()
	row := s.db.QueryRow(ctx, `SELECT avatar_id FROM profile WHERE id=$1 AND is_active=TRUE`, communityProfileID)
	logQuery(ctx, "communityStorage.GetAvatarID", start)

	var avatarID *int64
	if err := row.Scan(&avatarID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCommunityNotFound
		}
		return nil, err
	}
	return avatarID, nil
}

func (s *communityStorage) Delete(ctx context.Context, communityID int64) error {
	start := time.Now()
	tag, err := s.db.Exec(ctx, `UPDATE community SET is_active=FALSE, updated_at=NOW() WHERE id=$1 AND is_active=TRUE`, communityID)
	logQuery(ctx, "communityStorage.Delete", start)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrCommunityNotFound
	}
	return nil
}

func (s *communityStorage) GetMember(ctx context.Context, communityID, profileID int64) (*model.CommunityMember, error) {
	var member model.CommunityMember
	start := time.Now()
	err := pgxscan.Get(ctx, s.db, &member, `SELECT * FROM community_member WHERE community_id=$1 AND profile_id=$2 AND is_active=TRUE`, communityID, profileID)
	logQuery(ctx, "communityStorage.GetMember", start)
	if err != nil {
		if pgxscan.NotFound(err) {
			return nil, ErrCommunityMemberNotFound
		}
		return nil, err
	}
	return &member, nil
}

func (s *communityStorage) ListMembers(ctx context.Context, communityID int64, includeBlocked bool) ([]model.CommunityMember, error) {
	query := `SELECT * FROM community_member WHERE community_id=$1 AND is_active=TRUE`
	args := []any{communityID}
	if !includeBlocked {
		query += ` AND community_role <> $2`
		args = append(args, model.Blocked)
	}
	query += ` ORDER BY joined_at ASC, id ASC`

	start := time.Now()
	rows, err := s.db.Query(ctx, query, args...)
	logQuery(ctx, "communityStorage.ListMembers", start)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return pgx.CollectRows(rows, pgx.RowToStructByName[model.CommunityMember])
}

func (s *communityStorage) UpsertMemberRole(ctx context.Context, communityID, profileID int64, role model.CommunityMemberRole) (*model.CommunityMember, error) {
	var member model.CommunityMember
	start := time.Now()
	err := pgxscan.Get(ctx, s.db, &member, `
		INSERT INTO community_member (uid, profile_id, community_id, community_role, is_active, leave_at)
		VALUES ($1, $2, $3, $4, TRUE, NULL)
		ON CONFLICT (profile_id, community_id)
		DO UPDATE SET community_role=EXCLUDED.community_role, is_active=TRUE, leave_at=NULL, updated_at=NOW()
		RETURNING *
	`, uuid.New(), profileID, communityID, role)
	logQuery(ctx, "communityStorage.UpsertMemberRole", start)
	if err != nil {
		return nil, err
	}
	return &member, nil
}

func (s *communityStorage) DeactivateMember(ctx context.Context, communityID, profileID int64) error {
	start := time.Now()
	tag, err := s.db.Exec(ctx, `
		UPDATE community_member
		SET is_active=FALSE, leave_at=NOW(), updated_at=NOW()
		WHERE community_id=$1 AND profile_id=$2 AND is_active=TRUE
	`, communityID, profileID)
	logQuery(ctx, "communityStorage.DeactivateMember", start)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrCommunityMemberNotFound
	}
	return nil
}

func (s *communityStorage) Search(ctx context.Context, query string, limit int) ([]SearchCommunityResult, error) {
	pattern := likePattern(query)
	prefixPattern := likePrefixPattern(query)
	sql := `
		SELECT c.id, c.profile_id, c.username, c.title, c.bio,
		       c.community_type, p.avatar_id, c.cover_media_id
		FROM community c
		JOIN profile p ON p.id = c.profile_id
		WHERE c.is_active = TRUE
		  AND p.is_active = TRUE
		  AND c.community_type = $1
		  AND (
			c.username ILIKE $2 ESCAPE E'\\'
			OR c.title ILIKE $2 ESCAPE E'\\'
			OR COALESCE(c.bio, '') ILIKE $2 ESCAPE E'\\'
		  )
		ORDER BY
		  CASE
			WHEN c.username ILIKE $3 ESCAPE E'\\' THEN 0
			WHEN c.title ILIKE $3 ESCAPE E'\\' THEN 1
			ELSE 2
		  END,
		  c.title ASC,
		  c.id ASC
		LIMIT $4`

	start := time.Now()
	rows, err := s.db.Query(ctx, sql, model.PublicGroup, pattern, prefixPattern, limit)
	logQuery(ctx, "communityStorage.Search", start)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return pgx.CollectRows(rows, pgx.RowToStructByName[SearchCommunityResult])
}

func communitySelect() string {
	return `
		SELECT c.id, c.uid, c.title, c.bio, c.community_type, c.profile_id, c.username, c.cover_media_id,
		       c.is_active, c.created_at, c.updated_at
		FROM community c`
}

func likePattern(query string) string {
	return "%" + escapeLike(strings.TrimSpace(query)) + "%"
}

func likePrefixPattern(query string) string {
	return escapeLike(strings.TrimSpace(query)) + "%"
}

func escapeLike(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(value)
}

func normalizeCommunityError(err error) error {
	if pgxscan.NotFound(err) {
		return ErrCommunityNotFound
	}
	return err
}

func mapPgError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrDuplicateEntry
	}
	return err
}

func logQuery(ctx context.Context, query string, start time.Time) {
	logg := logger.FromContext(ctx)
	if logg == nil {
		return
	}
	logg.Debug("db query", zap.String("query", query), zap.Duration("duration_ms", time.Since(start)))
}
