package chatmember

//go:generate mockgen -destination=./../mocks/chat_member_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/chat_member ChatMemberRepo

import (
	"context"
	"errors"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

type ChatMemberRepo interface {
	Save(ctx context.Context, member models.ChatMember) error
	GetByChatID(ctx context.Context, chatID int64) ([]models.ChatMember, error)
	GetByUserID(ctx context.Context, userID int64) ([]models.ChatMember, error)
	Delete(ctx context.Context, id int64) error
}

type chatMemberStorage struct {
	db chatMemberDB
}

type chatMemberDB interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func NewChatMemberStorage(db chatMemberDB) ChatMemberRepo {
	return &chatMemberStorage{db: db}
}

func (s *chatMemberStorage) Save(ctx context.Context, member models.ChatMember) error {
	logger := logger.FromContext(ctx)
	query := `INSERT INTO chat_member (uid, chat_id, profile_id, joined_at, chat_role) 
              VALUES ($1, $2, $3, $4, $5)`
	start := time.Now()
	_, err := s.db.Exec(ctx, query, member.Uid, member.ChatID, member.MemberID, member.JoinedAt, member.Role)
	if logger != nil {
		logger.Debug("db query",
			zap.String("query", "GetUserByID"),
			zap.Duration("duration_ms", time.Since(start)),
		)
	}
	return err
}

func (s *chatMemberStorage) GetByChatID(ctx context.Context, chatID int64) ([]models.ChatMember, error) {
	logger := logger.FromContext(ctx)
	query := `SELECT * FROM chat_member WHERE chat_id=$1 AND leave_at IS NULL`
	var members []models.ChatMember
	start := time.Now()
	err := pgxscan.Select(ctx, s.db, &members, query, chatID)
	if logger != nil {
		logger.Debug("db query",
			zap.String("query", "GetUserByID"),
			zap.Duration("duration_ms", time.Since(start)),
		)
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	return members, nil
}

func (s *chatMemberStorage) GetByUserID(ctx context.Context, userID int64) ([]models.ChatMember, error) {
	logger := logger.FromContext(ctx)
	query := `SELECT * FROM chat_member WHERE profile_id=$1 AND leave_at IS NULL`
	var members []models.ChatMember
	start := time.Now()
	err := pgxscan.Select(ctx, s.db, &members, query, userID)
	if logger != nil {
		logger.Debug("db query",
			zap.String("query", "GetUserByID"),
			zap.Duration("duration_ms", time.Since(start)),
		)
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	return members, nil
}

func (s *chatMemberStorage) Delete(ctx context.Context, id int64) error {
	logger := logger.FromContext(ctx)
	query := `UPDATE chat_member SET leave_at=NOW(), updated_at=NOW() WHERE id=$1`
	start := time.Now()
	_, err := s.db.Exec(ctx, query, id)
	if logger != nil {
		logger.Debug("db query",
			zap.String("query", "GetUserByID"),
			zap.Duration("duration_ms", time.Since(start)),
		)
	}
	return err
}
