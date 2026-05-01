package http

import (
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
)

type createCommunityRequest struct {
	Title    string               `json:"title"`
	Bio      *string              `json:"bio,omitempty"`
	Type     models.CommunityType `json:"type"`
	Username string               `json:"username"`
	AvatarID *int64               `json:"avatarId,omitempty"`
}

type updateCommunityRequest struct {
	Title    *string               `json:"title,omitempty"`
	Bio      *string               `json:"bio,omitempty"`
	Type     *models.CommunityType `json:"type,omitempty"`
	Username *string               `json:"username,omitempty"`
	AvatarID *int64                `json:"avatarId,omitempty"`
}

type communityResponse struct {
	ID        int64                `json:"id"`
	UID       string               `json:"uid"`
	ProfileID int64                `json:"profileId"`
	Username  string               `json:"username"`
	Title     string               `json:"title"`
	Bio       *string              `json:"bio,omitempty"`
	Type      models.CommunityType `json:"type"`
	AvatarID  *int64               `json:"avatarId,omitempty"`
	AvatarURL *string              `json:"avatarUrl,omitempty"`
	IsActive  bool                 `json:"isActive"`
	CreatedAt time.Time            `json:"createdAt"`
	UpdatedAt time.Time            `json:"updatedAt"`
}

type membershipResponse struct {
	IsMember bool                        `json:"isMember"`
	Role     *models.CommunityMemberRole `json:"role,omitempty"`
}

type permissionsResponse struct {
	CanEdit   bool `json:"canEdit"`
	CanDelete bool `json:"canDelete"`
	CanPost   bool `json:"canPost"`
}

type communityDetailsResponse struct {
	Community   communityResponse   `json:"community"`
	Membership  membershipResponse  `json:"membership"`
	Permissions permissionsResponse `json:"permissions"`
}

type communityListResponse struct {
	Items []communityDetailsResponse `json:"items"`
}

type errorResponse struct {
	Error string `json:"error"`
}
