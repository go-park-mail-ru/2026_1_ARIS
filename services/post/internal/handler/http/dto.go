package http

//go:generate go run github.com/mailru/easyjson/easyjson -all $GOFILE

import "time"

type mediaRequestData struct {
	MediaID  int64  `json:"mediaID"`
	Name     string `json:"name,omitempty"`
	MediaURL string `json:"mediaURL"`
}

type postCreationRequest struct {
	Media           *[]mediaRequestData `json:"media"`
	Files           *[]mediaRequestData `json:"files"`
	Text            *string             `json:"text"`
	AuthorProfileID *int64              `json:"authorProfileId,omitempty"`
	CommunityID     *int64              `json:"communityId,omitempty"`
}

type postCreationResponse struct {
	ID          int64              `json:"id"`
	ProfileID   int64              `json:"profileID"`
	CommunityID *int64             `json:"communityId,omitempty"`
	Media       []mediaRequestData `json:"media"`
	Files       []mediaRequestData `json:"files"`
	Text        *string            `json:"text"`
	Author      postAuthorDTO      `json:"author"`
	Likes       int                `json:"likes"`
	Comments    int                `json:"comments"`
	IsLiked     bool               `json:"isLiked"`
}

type postListItemResponse struct {
	ID          int64              `json:"id"`
	ProfileID   int64              `json:"profileID"`
	CommunityID *int64             `json:"communityId,omitempty"`
	Text        string             `json:"text"`
	Author      postAuthorDTO      `json:"author"`
	Media       []mediaRequestData `json:"media"`
	Files       []mediaRequestData `json:"files"`
	CreatedAt   time.Time          `json:"createdAt"`
	UpdatedAt   *time.Time         `json:"updatedAt,omitempty"`
	Likes       int                `json:"likes"`
	Comments    int                `json:"comments"`
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
	IsLiked   bool           `json:"isLiked"`
	Comments  int            `json:"comments"`
	Reposts   int            `json:"reposts"`
	Medias    []mediaFeedDTO `json:"medias"`
	Files     []mediaFeedDTO `json:"files"`
}

type commentRequest struct {
	Text            string `json:"text"`
	ParentCommentID *int64 `json:"parentCommentId,omitempty"`
}

type commentResponse struct {
	ID              string        `json:"id"`
	Uid             string        `json:"uid"`
	Text            *string       `json:"text,omitempty"`
	PostID          string        `json:"postId"`
	ParentCommentID *string       `json:"parentCommentId,omitempty"`
	Author          postAuthorDTO `json:"author"`
	CreatedAt       time.Time     `json:"createdAt"`
	UpdatedAt       time.Time     `json:"updatedAt"`
	RepliesCount    int           `json:"repliesCount"`
	Likes           int           `json:"likes"`
	IsLiked         bool          `json:"isLiked"`
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
	Name     string `json:"name"`
	MimeType string `json:"mimeType"`
	Link     string `json:"mediaLink"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type feedEventItem struct {
	PostID   int64  `json:"postId"`
	Type     string `json:"type"`
	DwellMs  uint32 `json:"dwellMs"`
	Position uint16 `json:"position"`
	Source   string `json:"source"`
}

type feedEventsRequest struct {
	Events []feedEventItem `json:"events"`
}
