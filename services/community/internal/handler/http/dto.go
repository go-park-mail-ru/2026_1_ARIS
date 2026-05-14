package http

import (
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/services/community/internal/model"
)

type createCommunityRequest struct {
	Title        string              `json:"title"`
	Bio          *string             `json:"bio,omitempty"`
	Type         model.CommunityType `json:"type"`
	Username     string              `json:"username"`
	AvatarID     *int64              `json:"avatarId,omitempty"`
	CoverMediaID *int64              `json:"coverId,omitempty"`
}

type updateCommunityRequest struct {
	Title        *string              `json:"title,omitempty"`
	Bio          *string              `json:"bio,omitempty"`
	Type         *model.CommunityType `json:"type,omitempty"`
	Username     *string              `json:"username,omitempty"`
	AvatarID     *int64               `json:"avatarId,omitempty"`
	CoverMediaID *int64               `json:"coverId,omitempty"`
	RemoveAvatar *bool                `json:"removeAvatar,omitempty"`
	RemoveCover  *bool                `json:"removeCover,omitempty"`
}

type updateMemberRoleRequest struct {
	Role model.CommunityMemberRole `json:"role"`
}

type communityResponse struct {
	ID           int64               `json:"id"`
	UID          string              `json:"uid"`
	ProfileID    int64               `json:"profileId"`
	Username     string              `json:"username"`
	Title        string              `json:"title"`
	Bio          *string             `json:"bio,omitempty"`
	Type         model.CommunityType `json:"type"`
	AvatarID     *int64              `json:"avatarId,omitempty"`
	AvatarURL    *string             `json:"avatarUrl,omitempty"`
	CoverMediaID *int64              `json:"coverId,omitempty"`
	CoverURL     *string             `json:"coverUrl,omitempty"`
	IsActive     bool                `json:"isActive"`
	CreatedAt    time.Time           `json:"createdAt"`
	UpdatedAt    time.Time           `json:"updatedAt"`
}

type membershipResponse struct {
	IsMember bool                       `json:"isMember"`
	Role     *model.CommunityMemberRole `json:"role,omitempty"`
	Blocked  bool                       `json:"blocked"`
}

type permissionsResponse struct {
	CanEditCommunity   bool `json:"canEditCommunity"`
	CanDeleteCommunity bool `json:"canDeleteCommunity"`
	CanPost            bool `json:"canPost"`
	CanPostAsCommunity bool `json:"canPostAsCommunity"`
	CanPostAsMember    bool `json:"canPostAsMember"`
	CanManageMembers   bool `json:"canManageMembers"`
	CanChangeRoles     bool `json:"canChangeRoles"`
}

type communityDetailsResponse struct {
	Community   communityResponse   `json:"community"`
	Membership  membershipResponse  `json:"membership"`
	Permissions permissionsResponse `json:"permissions"`
}

type communityListResponse struct {
	Items []communityDetailsResponse `json:"items"`
}

type communityMemberResponse struct {
	ProfileID     int64                     `json:"profileId"`
	UserAccountID int64                     `json:"userAccountId"`
	FirstName     string                    `json:"firstName"`
	LastName      string                    `json:"lastName"`
	Username      string                    `json:"username"`
	AvatarID      *int64                    `json:"avatarId,omitempty"`
	AvatarURL     *string                   `json:"avatarUrl,omitempty"`
	Role          model.CommunityMemberRole `json:"role"`
	Blocked       bool                      `json:"blocked"`
	IsSelf        bool                      `json:"isSelf"`
	JoinedAt      string                    `json:"joinedAt"`
}

type communityMembersResponse struct {
	Items []communityMemberResponse `json:"items"`
}

type errorResponse struct {
	Error string `json:"error"`
}
