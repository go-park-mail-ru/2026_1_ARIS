package http

type ChatResponse struct {
	ID        string `json:"id"`
	Uid       string `json:"uid"`
	Title     string `json:"title"`
	AvatarID  *int64 `json:"avatarId,omitempty"`
	Type      string `json:"type"`
	IsActive  bool   `json:"isActive"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type MessageResponse struct {
	ID              string  `json:"id"`
	Uid             string  `json:"uid"`
	Text            *string `json:"text,omitempty"`
	AuthorName      string  `json:"authorName"`
	ParentMessageID *string `json:"parentMessage,omitempty"`
	ChatID          string  `json:"chat"`
	AuthorID        string  `json:"authorId"`
	StickerID       *string `json:"sticker,omitempty"`
	IsActive        bool    `json:"isActive"`
	CreatedAt       string  `json:"createdAt"`
	UpdatedAt       string  `json:"updatedAt"`
}

type textRequest struct {
	Text string `json:"text"`
}

type errorResponse struct {
	Error string `json:"error"`
}
