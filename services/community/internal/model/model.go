package model

import (
	"time"

	"github.com/google/uuid"
)

type CommunityType string

const (
	PublicGroup  CommunityType = "public"
	PrivateGroup CommunityType = "private"
)

type CommunityMemberRole string

const (
	Owner     CommunityMemberRole = "owner"
	Admin     CommunityMemberRole = "admin"
	Moderator CommunityMemberRole = "moderator"
	Member    CommunityMemberRole = "member"
	Blocked   CommunityMemberRole = "blocked"
)

type Community struct {
	ID           int64         `db:"id"`
	Uid          uuid.UUID     `db:"uid"`
	Title        string        `db:"title"`
	Bio          *string       `db:"bio"`
	Type         CommunityType `db:"community_type"`
	ProfileID    int64         `db:"profile_id"`
	Username     string        `db:"username"`
	CoverMediaID *int64        `db:"cover_media_id"`
	IsActive     bool          `db:"is_active"`
	CreatedAt    time.Time     `db:"created_at"`
	UpdatedAt    time.Time     `db:"updated_at"`
}

type CommunityMember struct {
	ID          int64               `db:"id"`
	Uid         uuid.UUID           `db:"uid"`
	CommunityID int64               `db:"community_id"`
	MemberID    int64               `db:"profile_id"`
	Role        CommunityMemberRole `db:"community_role"`
	JoinedAt    time.Time           `db:"joined_at"`
	LeaveAt     *time.Time          `db:"leave_at"`
	IsActive    bool                `db:"is_active"`
	CreatedAt   time.Time           `db:"created_at"`
	UpdatedAt   time.Time           `db:"updated_at"`
}
