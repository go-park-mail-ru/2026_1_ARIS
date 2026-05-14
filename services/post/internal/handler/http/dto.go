package http

import "time"

type mediaRequestData struct {
	MediaID  int64  `json:"mediaID"`
	MediaURL string `json:"mediaURL"`
}

type postCreationRequest struct {
	Media           *[]mediaRequestData `json:"media"`
	Text            *string             `json:"text"`
	AuthorProfileID *int64              `json:"authorProfileId,omitempty"`
	CommunityID     *int64              `json:"communityId,omitempty"`
}

type postCreationResponse struct {
	ID          int64              `json:"id"`
	ProfileID   int64              `json:"profileID"`
	CommunityID *int64             `json:"communityId,omitempty"`
	Media       []mediaRequestData `json:"media"`
	Text        *string            `json:"text"`
	Author      postAuthorDTO      `json:"author"`
	Likes       int                `json:"likes"`
	IsLiked     bool               `json:"isLiked"`
}

type postListItemResponse struct {
	ID          int64              `json:"id"`
	ProfileID   int64              `json:"profileID"`
	CommunityID *int64             `json:"communityId,omitempty"`
	Text        string             `json:"text"`
	Author      postAuthorDTO      `json:"author"`
	Media       []mediaRequestData `json:"media"`
	CreatedAt   time.Time          `json:"createdAt"`
	UpdatedAt   *time.Time         `json:"updatedAt,omitempty"`
	Likes       int                `json:"likes"`
	IsLiked     bool               `json:"isLiked"`
}

type postAuthorDTO struct {
	ProfileID     int64   `json:"profileID"`
	FirstName     string  `json:"firstName"`
	LastName      string  `json:"lastName"`
	Username      string  `json:"username"`
	UserAccountID int64   `json:"userAccountID"`
	AvatarURL     *string `json:"avatarURL,omitempty"`
}

type feedResponse struct {
	Items      []postFeedDTO `json:"posts"`
	NextCursor string        `json:"nextCursor"`
	HasMore    bool          `json:"hasMore"`
}

type postFeedDTO struct {
	ID        int64          `json:"id"`
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
	ID         string `json:"id"`
	FirstName  string `json:"firstName"`
	LastName   string `json:"lastName"`
	Username   string `json:"username"`
	AvatarLink string `json:"avatarLink"`
}

type mediaFeedDTO struct {
	ID       string `json:"id"`
	MimeType string `json:"mimeType"`
	Link     string `json:"mediaLink"`
}

type errorResponse struct {
	Error string `json:"error"`
}
