package repository

import (
	"context"
	"errors"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ChatMemberRepo interface {
	Save(ctx context.Context, member models.ChatMember) error
	GetByChatID(ctx context.Context, chatID int64) ([]models.ChatMember, error)
	GetByUserID(ctx context.Context, userID int64) ([]models.ChatMember, error)
	Delete(ctx context.Context, id int64) error
}

type chatMemberStorage struct {
	db *pgxpool.Pool
}

func NewChatMemberStorage(db *pgxpool.Pool) ChatMemberRepo {
	return &chatMemberStorage{db: db}
}

func (s *chatMemberStorage) Save(ctx context.Context, member models.ChatMember) error {
	query := `INSERT INTO chat_member (uid, chat_id, profile_id, joined_at, chat_role) 
              VALUES ($1, $2, $3, $4, $5)`
	_, err := s.db.Exec(ctx, query, member.Uid, member.ChatID, member.MemberID, member.JoinedAt, member.Role)
	return err
}

func (s *chatMemberStorage) GetByChatID(ctx context.Context, chatID int64) ([]models.ChatMember, error) {
	query := `SELECT id, uid, chat_id, profile_id, joined_at, is_active, leave_at, created_at, updated_at, chat_role FROM chat_member WHERE chat_id=$1 AND leave_at IS NULL AND is_active=true;`
	var members []models.ChatMember
	err := pgxscan.Select(ctx, s.db, &members, query, chatID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	return members, nil
}

func (s *chatMemberStorage) GetByUserID(ctx context.Context, userID int64) ([]models.ChatMember, error) {
	query := `SELECT id, uid, chat_id, profile_id, joined_at, is_active, leave_at, created_at, updated_at, chat_role FROM chat_member WHERE profile_id=$1 AND leave_at IS NULL AND is_active=true;`
	var members []models.ChatMember
	err := pgxscan.Select(ctx, s.db, &members, query, userID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	return members, nil
}

func (s *chatMemberStorage) Delete(ctx context.Context, id int64) error {
	query := `UPDATE chat_member SET leave_at=NOW(), is_active=false WHERE id=$1`
	_, err := s.db.Exec(ctx, query, id)
	return err
}
