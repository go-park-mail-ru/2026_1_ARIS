package http

//go:generate go run github.com/mailru/easyjson/easyjson -all $GOFILE

type ChatResponse struct {
	ID                        string  `json:"id"`
	Uid                       string  `json:"uid"`
	Title                     string  `json:"title"`
	AvatarID                  *int64  `json:"avatarId,omitempty"`
	AvatarLink                string  `json:"avatarLink,omitempty"`
	Type                      string  `json:"type"`
	IsActive                  bool    `json:"isActive"`
	InterlocutorProfileID     *string `json:"interlocutorProfileId,omitempty"`
	InterlocutorUserAccountID *string `json:"interlocutorUserAccountId,omitempty"`
	IsOnline                  bool    `json:"isOnline"`
	LastSeenAt                *string `json:"lastSeenAt,omitempty"`
	CreatedAt                 string  `json:"createdAt"`
	UpdatedAt                 string  `json:"updatedAt"`
}

type MessageResponse struct {
	ID              string               `json:"id"`
	Uid             string               `json:"uid"`
	Text            *string              `json:"text,omitempty"`
	AuthorName      string               `json:"authorName"`
	ParentMessageID *string              `json:"parentMessage,omitempty"`
	ChatID          string               `json:"chat"`
	AuthorID        string               `json:"authorId"`
	StickerID       *string              `json:"sticker,omitempty"`
	Sticker         *StickerResponse     `json:"stickerData,omitempty"`
	Media           []AttachmentResponse `json:"media"`
	Files           []AttachmentResponse `json:"files"`
	Reactions       []ReactionResponse   `json:"reactions"`
	MyReaction      *string              `json:"myReaction,omitempty"`
	Type            string               `json:"type"`
	IsActive        bool                 `json:"isActive"`
	CreatedAt       string               `json:"createdAt"`
	UpdatedAt       string               `json:"updatedAt"`
}

type messageRequest struct {
	Text            string              `json:"text"`
	ParentMessageID *int64              `json:"parentMessageId,omitempty"`
	StickerID       *int64              `json:"stickerId,omitempty"`
	Media           []attachmentRequest `json:"media"`
	Files           []attachmentRequest `json:"files"`
	Type            string              `json:"type"`
}

type attachmentRequest struct {
	MediaID int64 `json:"mediaID"`
}

type AttachmentResponse struct {
	ID       string `json:"id"`
	Uid      string `json:"uid"`
	Name     string `json:"name"`
	MimeType string `json:"mimeType"`
	URL      string `json:"url"`
}

type StickerPackResponse struct {
	ID        string  `json:"id"`
	Uid       string  `json:"uid"`
	Title     string  `json:"title"`
	AuthorID  *string `json:"authorId"`
	CreatedAt string  `json:"createdAt"`
	UpdatedAt string  `json:"updatedAt"`
}

type stickerPackRequest struct {
	Title string `json:"title"`
}

type stickerRequest struct {
	MediaID   int64 `json:"mediaID"`
	SortOrder *int  `json:"sortOrder,omitempty"`
}

type StickerResponse struct {
	ID       string  `json:"id"`
	Uid      string  `json:"uid"`
	PackID   *string `json:"packId,omitempty"`
	MediaID  *string `json:"mediaId,omitempty"`
	MimeType *string `json:"mimeType,omitempty"`
	URL      *string `json:"url,omitempty"`
}

type reactionRequest struct {
	Type string `json:"type"`
}

type ReactionResponse struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
}

type errorResponse struct {
	Error string `json:"error"`
}
