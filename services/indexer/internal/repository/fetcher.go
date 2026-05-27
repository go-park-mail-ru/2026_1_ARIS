package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

var ErrNotFound = errors.New("entity not found")

type UserDoc struct {
	UserAccountID int64
	ProfileID     int64
	Username      string
	FirstName     string
	LastName      string
	AvatarID      *int64
	IsActive      bool
}

type CommunityDoc struct {
	CommunityID   int64
	ProfileID     int64
	Username      string
	Title         string
	Bio           *string
	CommunityType string
	AvatarID      *int64
	CoverMediaID  *int64
	IsActive      bool
}

type PostDoc struct {
	PostID          int64
	PostText        *string
	AuthorID        int64
	AuthorProfileID int64
	AuthorUsername  string
	AuthorFirstName string
	AuthorLastName  string
	AuthorAvatarID  *int64
	CommunityID     *int64
	IsActive        bool
	IsPublic        bool
	CreatedAt       time.Time
}

type FetcherRepo struct {
	db DB
}

func NewFetcherRepo(db DB) *FetcherRepo {
	return &FetcherRepo{db: db}
}

func (r *FetcherRepo) FetchUser(ctx context.Context, userAccountID int64) (*UserDoc, error) {
	row := r.db.QueryRow(ctx, `
		SELECT ua.id, p.id,
		       ua.username, up.first_name, up.last_name,
		       p.avatar_id,
		       ua.is_active AND up.is_active AND p.is_active
		FROM user_account ua
		JOIN user_profile up ON up.user_account_id = ua.id
		JOIN profile p ON p.id = up.profile_id
		WHERE ua.id = $1
	`, userAccountID)

	var doc UserDoc
	if err := row.Scan(&doc.UserAccountID, &doc.ProfileID,
		&doc.Username, &doc.FirstName, &doc.LastName,
		&doc.AvatarID, &doc.IsActive); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &doc, nil
}

func (r *FetcherRepo) FetchCommunity(ctx context.Context, communityID int64) (*CommunityDoc, error) {
	row := r.db.QueryRow(ctx, `
		SELECT c.id, c.profile_id, c.username, c.title, c.bio,
		       c.community_type, p.avatar_id, c.cover_media_id, c.is_active
		FROM community c
		JOIN profile p ON p.id = c.profile_id
		WHERE c.id = $1
	`, communityID)

	var doc CommunityDoc
	if err := row.Scan(&doc.CommunityID, &doc.ProfileID, &doc.Username, &doc.Title, &doc.Bio,
		&doc.CommunityType, &doc.AvatarID, &doc.CoverMediaID, &doc.IsActive); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &doc, nil
}

func (r *FetcherRepo) FetchPost(ctx context.Context, postID int64) (*PostDoc, error) {
	row := r.db.QueryRow(ctx, `
		SELECT p.id, p.post_text, p.author_id, p.community_id,
		       p.is_active, p.is_public_demo, p.created_at,
		       pr.id, ua.username, up.first_name, up.last_name, pr.avatar_id
		FROM post p
		JOIN profile pr ON pr.id = p.author_id
		JOIN user_profile up ON up.profile_id = pr.id
		JOIN user_account ua ON ua.id = up.user_account_id
		WHERE p.id = $1
	`, postID)

	var doc PostDoc
	if err := row.Scan(&doc.PostID, &doc.PostText, &doc.AuthorID, &doc.CommunityID,
		&doc.IsActive, &doc.IsPublic, &doc.CreatedAt,
		&doc.AuthorProfileID, &doc.AuthorUsername, &doc.AuthorFirstName, &doc.AuthorLastName, &doc.AuthorAvatarID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &doc, nil
}
