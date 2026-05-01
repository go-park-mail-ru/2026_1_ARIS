package http

import (
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/dto"
)

type PostCreationRequest struct {
	Media           *[]dto.MediaRequestData `json:"media"`
	Text            *string                 `json:"text"`
	AuthorProfileID *int64                  `json:"authorProfileId,omitempty"`
}

type PostCreationResponse struct {
	ID            int64                  `json:"id"`
	ProfileID     int64                  `json:"profileID"`
	Media         []dto.MediaRequestData `json:"media"`
	MediaURL      []string               `json:"mediaURL"`
	Text          *string                `json:"text"`
	FirstName     string                 `json:"firstName"`
	LastName      string                 `json:"lastName"`
	UserAccountID int64                  `json:"userAccountID"`
	AvatarURL     *string                `json:"avatarURL"`
	Likes         int                    `json:"likes"`
	IsLiked       bool                   `json:"isLiked"`
}

type PostListItemResponse struct {
	ID        int64                  `json:"id"`
	ProfileID int64                  `json:"profileID"`
	Text      string                 `json:"text"`
	Media     []dto.MediaRequestData `json:"media"`
	MediaURL  []string               `json:"mediaURL"`
	CreatedAt time.Time              `json:"createdAt"`
	UpdatedAt *time.Time             `json:"updatedAt,omitempty"`
	Likes     int                    `json:"likes"`
	IsLiked   bool                   `json:"isLiked"`
}

type FeedResponse struct {
	Items      []postFeedDTO `json:"posts"`
	NextCursor string        `json:"nextCursor"`
	HasMore    bool          `json:"hasMore"`
}

type postFeedDTO struct {
	Id        string         `json:"id"`
	Text      string         `json:"text"`
	Author    authorFeedDTO  `json:"author"`
	CreatedAt time.Time      `json:"createdAt"`
	Likes     int            `json:"likes"`
	Comments  int            `json:"comments"`
	Reposts   int            `json:"reposts"`
	Medias    []mediaFeedDTO `json:"medias"`
}

type popularPostDTO struct {
	Title string `json:"title"`
}

type popularPostsResponse struct {
	Items []popularPostDTO `json:"items"`
}

type authorFeedDTO struct {
	Id         string `json:"id"`
	FirstName  string `json:"firstName"`
	LastName   string `json:"lastName"`
	Username   string `json:"username"`
	AvatarLink string `json:"avatarLink"`
}

type mediaFeedDTO struct {
	Id       string `json:"id"`
	MimeType string `json:"mimeType"`
	Link     string `json:"mediaLink"`
}

type errorResponse struct {
	Error string `json:"error"`
}
