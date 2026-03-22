package models

import (
	"math/rand/v2"
	"time"

	"github.com/google/uuid"
)

type ID int64
type UserID ID
type SessionID string

// models Types

type ChatType string

const (
	PrivateChat ChatType = "private"
	GroupChat   ChatType = "community"
)

type CommunityType string

const (
	PublicGroup  CommunityType = "public"
	PrivateGroup CommunityType = "private"
)

type CommunityMemberRole string

const (
	Owner   CommunityMemberRole = "owner"
	Admin   CommunityMemberRole = "admin"
	Manager CommunityMemberRole = "manager"
	Member  CommunityMemberRole = "member"
)

type FriendshipStatus string

const (
	FriendshipPending  FriendshipStatus = "pending"
	FriendshipAccepted FriendshipStatus = "accepted"
	FriendshipDeclined FriendshipStatus = "declined"
	FriendshipBlocked  FriendshipStatus = "blocked"
)

type ReactionType string

const (
	ReactionLike  ReactionType = "👍"
	ReactionLove  ReactionType = "❤️"
	ReactionLaugh ReactionType = "😂"
	ReactionSad   ReactionType = "😢"
	ReactionAngry ReactionType = "😡"
)

type Gender int

const (
	Male Gender = iota
	Female
)

type MessageStatus int

const (
	NotSend MessageStatus = iota
	Senging
	Send
	Read
)

// models structs
// credentials данные
type UserAccount struct {
	ID           int64     `json:"id"`
	Uid          uuid.UUID `json:"uid"`
	Username     string    `json:"username"`
	Email        *string   `json:"email"`
	Phone        *string   `json:"phone"`
	PasswordHash string    `json:"-"`
	IsActive     bool      `json:"isActive"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func NewUserAccount(username string, email, phone *string, passwordHash string) *UserAccount {
	now := time.Now()
	return &UserAccount{
		ID:           rand.Int64(),
		Uid:          uuid.New(),
		Email:        email,
		Phone:        phone,
		PasswordHash: passwordHash,
		Username:     username,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// UserProfile - user-specific profile information
// профиль пользователя
type UserProfile struct {
	ID            int64     `json:"id"`
	Uid           uuid.UUID `json:"uid"`
	UserAccountID int64     `json:"userAccountId"`
	ProfileID     int64     `json:"profileId"`
	FirstName     string    `json:"firstName"`
	LastName      string    `json:"lastName"`
	Bio           *string   `json:"bio,omitempty"`
	BirthdayDate  time.Time `json:"birthdayDate,omitempty"`
	Gender        Gender    `json:"gender"`
	IsActive      bool      `json:"isActive"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

func NewUserProfile(userAccountId, profileID int64, firstName, lastName string, bio *string, birthday time.Time, gender Gender) *UserProfile {
	now := time.Now()
	return &UserProfile{
		ID:            rand.Int64(),
		Uid:           uuid.New(),
		UserAccountID: userAccountId,
		ProfileID:     profileID,
		FirstName:     firstName,
		LastName:      lastName,
		Bio:           bio,
		BirthdayDate:  birthday,
		Gender:        gender,
		IsActive:      true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

type Media struct {
	ID          int64     `json:"id"`
	Uid         uuid.UUID `json:"uid"`
	Name        string    `json:"name"`
	Extension   string    `json:"extension"`
	Description *string   `json:"description,omitempty"`
	MimeType    string    `json:"mimeType"`
	Link        string    `json:"link"`
	Size        int       `json:"size"`
	IsActive    bool      `json:"isActive"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func NewMedia(name, extension string, description *string, mimeType, link string) *Media {
	now := time.Now()
	return &Media{
		ID:          rand.Int64(),
		Uid:         uuid.New(),
		Name:        name,
		Extension:   extension,
		Description: description,
		MimeType:    mimeType,
		Link:        link,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// Abstract profile for both users and groups
type Profile struct {
	ID        int64     `json:"id"`
	Uid       uuid.UUID `json:"uid"`
	AvatarID  *int64    `json:"avatar,omitempty"`
	IsActive  bool      `json:"isActive"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func NewProfile(avatarID *int64) *Profile {
	now := time.Now()

	return &Profile{
		ID:        rand.Int64(),
		Uid:       uuid.New(),
		AvatarID:  avatarID,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

type Post struct {
	ID           int64     `json:"id"`
	Uid          uuid.UUID `json:"uid"`
	Text         *string   `json:"text,omitempty"`
	AuthorID     int64     `json:"authorId"` // to Profile
	IsPublicDemo bool      `json:"isPublicDemo"`
	IsActive     bool      `json:"isActive"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func NewPost(text *string, authorID int64) *Post {
	now := time.Now()

	return &Post{
		ID:        rand.Int64(),
		Uid:       uuid.New(),
		Text:      text,
		AuthorID:  authorID,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// PostWithMedia - junction table for posts and media
type PostWithMedia struct {
	PostID  int64 `json:"postId"`
	MediaID int64 `json:"mediaId"`
	Order   int   `json:"order"`
}

func NewPostWithMedia(postID, mediaID int64, order int) *PostWithMedia {
	return &PostWithMedia{
		PostID:  postID,
		MediaID: mediaID,
		Order:   order,
	}
}

type Chat struct {
	ID        int64     `json:"id"`
	Uid       uuid.UUID `json:"uid"`
	Type      ChatType  `json:"type"`
	Title     string    `json:"title"`
	AvatarID  *int64    `json:"avatar,omitempty"`
	IsActive  bool      `json:"isActive"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ChatMember - represents a member in a chat
type ChatMember struct {
	ID        int64      `json:"id"`
	Uid       uuid.UUID  `json:"uid"`
	ChatID    int64      `json:"chat"`
	MemberID  int64      `json:"member"`
	JoinedAt  time.Time  `json:"joinedAt"`
	IsActive  bool       `json:"isActive"`
	LeaveAt   *time.Time `json:"leaveAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updateAt"`
	Role      string     `json:"role"`
}

type Message struct {
	ID              int64         `json:"id"`
	Uid             uuid.UUID     `json:"uid"`
	Text            *string       `json:"text,omitempty"`
	ParentMessageID *int64        `json:"parentMessage,omitempty"`
	ChatID          int64         `json:"chat"`
	Status          MessageStatus `json:"status"`
	AuthorID        int64         `json:"authorId,omitempty"`
	StickerID       *int64        `json:"sticker,omitempty"`
	IsActive        bool          `json:"isActive"`
	CreatedAt       time.Time     `json:"createdAt"`
	UpdatedAt       time.Time     `json:"updatedAt"`
}

// MessageWithMedia - junction table for messages and media
type MessageWithMedia struct {
	MessageID int64 `json:"messageId"`
	MediaID   int64 `json:"mediaId"`
	Order     int   `json:"order"`
}

type Community struct {
	ID        int64         `json:"id"`
	Uid       uuid.UUID     `json:"uid"`
	Title     string        `json:"title"`
	Bio       *string       `json:"bio,omitempty"`
	Type      CommunityType `json:"type"`
	ProfileID int64         `json:"profileId"` // Abstract-Profile
	IsActive  bool          `json:"isActive"`
	CreatedAt time.Time     `json:"createdAt"`
	UpdatedAt time.Time     `json:"updatedAt"`
}

// CommunityMember - represents a member in a community
type CommunityMember struct {
	ID          int64               `json:"id"`
	Uid         uuid.UUID           `json:"uid"`
	CommunityID int64               `json:"community"`
	MemberID    int64               `json:"member"`
	Role        CommunityMemberRole `json:"role"`
	JoinedAt    time.Time           `json:"joinedAt"`
	LeaveAt     *time.Time          `json:"leaveAt,omitempty"`
	IsActive    bool                `json:"isActive"`
	CreatedAt   time.Time           `json:"createdAt"`
	UpdatedAt   time.Time           `json:"updatedAt"`
}

type Comment struct {
	ID              int64     `json:"id"`
	Uid             uuid.UUID `json:"uid"`
	Text            *string   `json:"text"`
	TargetPostID    int64     `json:"post"`
	ParentCommentID *int64    `json:"parentComment,omitempty"`
	StickerID       *int64    `json:"sticker,omitempty"`
	AuthorID        int64     `json:"author"`
	IsActive        bool      `json:"isActive"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

func NewComment(text *string, targetPostID int64, parentCommentID, stickerID *int64, authorID int64) *Comment {
	now := time.Now()
	return &Comment{
		ID:              rand.Int64(),
		Uid:             uuid.New(),
		Text:            text,
		TargetPostID:    targetPostID,
		ParentCommentID: parentCommentID,
		StickerID:       stickerID,
		AuthorID:        authorID,
		IsActive:        true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// CommentWithMedia - junction table for comments and media
type CommentWithMedia struct {
	CommentID int64 `json:"commentId"`
	MediaID   int64 `json:"mediaId"`
	Order     int   `json:"order"`
}

type Like struct {
	ID        int64     `json:"id"`
	Uid       uuid.UUID `json:"uid"`
	PostID    *int64    `json:"postID"`
	CommentID *int64    `json:"commentID"`
	AuthorID  int64     `json:"author"`
	IsActive  bool      `json:"isActive"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func NewLikeToPost(postID int64, authorID int64) *Like {
	now := time.Now()
	return &Like{
		ID:        rand.Int64(),
		Uid:       uuid.New(),
		PostID:    &postID,
		CommentID: nil,
		AuthorID:  authorID,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func NewLikeToComment(commentID int64, authorID int64) *Like {
	now := time.Now()
	return &Like{
		ID:        rand.Int64(),
		Uid:       uuid.New(),
		PostID:    nil,
		CommentID: &commentID,
		AuthorID:  authorID,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

type Friendship struct {
	Friend1ID   int64            `json:"friend1"`
	Friend2ID   int64            `json:"friend2"`
	REquesterID int64            `json:"requester"`
	Status      FriendshipStatus `json:"status"`
	IsActive    bool             `json:"isActive"`
	CreatedAt   time.Time        `json:"createdAt"`
	UpdatedAt   time.Time        `json:"updatedAt"`
}

type Stickerpack struct {
	ID        int64     `json:"id"`
	Uid       uuid.UUID `json:"uid"`
	Title     *string   `json:"title"`
	AuthorID  *int64    `json:"author"`
	IsActive  bool      `json:"isActive"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Sticker struct {
	ID         int64     `json:"id"`
	Uid        uuid.UUID `json:"uid"`
	Size       int       `json:"size"`
	IndexOrder int       `json:"indexOrder"`
	PackID     *int64    `json:"pack"`
	IsActive   bool      `json:"isActive"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updateAt"`
}

type Session struct {
	SessionID SessionID `json:"id"`
	UserID    int64     `json:"user"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiredAt time.Time `json:"expiredAt"`
}

type Ad struct {
	ID          int64     `json:"id"`
	Uid         uuid.UUID `json:"uid"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Link        string    `json:"link"`
	MediaID     int64     `json:"media,omitempty"`
	AuthorID    int64     `json:"author"`
	IsActive    bool      `json:"isActive"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// AdMeta - metadata for advertisements
type AdMeta struct {
	ID        int64     `json:"id"`
	Uid       uuid.UUID `json:"uid"`
	AdID      int64     `json:"adId"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	IsActive  bool      `json:"isActive"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Reaction struct {
	ID        int64        `json:"id"`
	Uid       uuid.UUID    `json:"uid"`
	MessageID int64        `json:"message"`
	Type      ReactionType `json:"type"`
	AuthorID  int64        `json:"author"`
	IsActive  bool         `json:"isActive"`
	CreatedAt time.Time    `json:"createdAt"`
	UpdatedAt time.Time    `json:"updatedAt"`
}

type Repost struct {
	ID       int64     `json:"id"`
	Uid      uuid.UUID `json:"uid"`
	AuthorID int64     `json:"authorId"` // Profile
	ChatID   int64     `json:"chatId"`
	PostID   int64     `json:"postId"`
}

func NewRepost(authorID, chatID, postID int64) *Repost {
	return &Repost{
		ID:       rand.Int64(),
		Uid:      uuid.New(),
		AuthorID: authorID,
		ChatID:   chatID,
		PostID:   postID,
	}
}
