package models

import (
	"time"

	"github.com/google/uuid"
)

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
type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	Phone        string    `json:"phone"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func NewUser(email, phone, passwordHash string) User {
	now := time.Now()
	return User{
		ID:           uuid.New(),
		Email:        email,
		Phone:        phone,
		PasswordHash: passwordHash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// UserProfile - user-specific profile information
// профиль пользователя
type UserProfile struct {
	ID           uuid.UUID  `json:"id"`
	UserID       uuid.UUID  `json:"userId"`
	ProfileID    uuid.UUID  `json:"profileId"` // Abstract-Profile
	FirstName    string     `json:"firstName"`
	LastName     string     `json:"lastName"`
	Bio          *string    `json:"bio,omitempty"`
	BirthdayDate *time.Time `json:"birthdayDate,omitempty"`
	Gender       Gender     `json:"gender"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

func NewUserProfile(user User, profile Profile, firstName, lastName string, bio *string, birthday *time.Time, gender Gender) UserProfile {
	now := time.Now()

	return UserProfile{
		ID:           uuid.New(),
		UserID:       user.ID,
		ProfileID:    profile.ID,
		FirstName:    firstName,
		LastName:     lastName,
		Bio:          bio,
		BirthdayDate: birthday,
		Gender:       gender,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

type Media struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Extension   string    `json:"extension"`
	Description *string   `json:"description,omitempty"`
	MimeType    string    `json:"mimeType"`
	Link        string    `json:"link"`
	Size        int       `json:"size"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	IsDeleted   bool      `json:"isDeleted"`
}

func NewMedia(name, extension string, description *string, mimeType, link string, size int, isDeleted bool) Media {
	now := time.Now()

	return Media{
		ID:          uuid.New(),
		Name:        name,
		Extension:   extension,
		Description: description,
		MimeType:    mimeType,
		Link:        link,
		Size:        size,
		CreatedAt:   now,
		UpdatedAt:   now,
		IsDeleted:   isDeleted,
	}
}

// Abstract profile for both users and groups
type Profile struct {
	ID        uuid.UUID  `json:"id"`
	AvatarID  *uuid.UUID `json:"avatar,omitempty"`
	Username  string     `json:"username"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	IsActive  bool       `json:"isActive"`
}

func NewProfile(username string, avatar *Media, isActive bool) Profile {
	var avatarID *uuid.UUID

	if avatar != nil {
		avatarID = &avatar.ID
	}

	now := time.Now()

	return Profile{
		ID:        uuid.New(),
		AvatarID:  avatarID,
		Username:  username,
		CreatedAt: now,
		UpdatedAt: now,
		IsActive:  isActive,
	}
}

type Post struct {
	ID        uuid.UUID `json:"id"`
	Text      *string   `json:"text,omitempty"`
	AuthorID  uuid.UUID `json:"authorId"` // to Profile
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	IsActive  bool      `json:"isActive"`
}

func NewPost(text *string, author Profile, isActive bool) Post {
	now := time.Now()

	return Post{
		ID:        uuid.New(),
		Text:      text,
		AuthorID:  author.ID,
		CreatedAt: now,
		UpdatedAt: now,
		IsActive:  isActive,
	}
}

// PostWithMedia - junction table for posts and media
type PostWithMedia struct {
	PostID  uuid.UUID `json:"postId"`
	MediaID uuid.UUID `json:"mediaId"`
	Order   int       `json:"order"`
}

func NewPostWithMedia(post Post, media Media, order int) PostWithMedia {
	return PostWithMedia{
		PostID:  post.ID,
		MediaID: media.ID,
		Order:   order,
	}
}

type Chat struct {
	ID        uuid.UUID  `json:"id"`
	TypeID    ChatType   `json:"type"`
	Title     string     `json:"title"`
	AvatarID  *uuid.UUID `json:"avatar,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	IsDeleted bool       `json:"isDeleted"`
}

// ChatMember - represents a member in a chat
type ChatMember struct {
	ID        uuid.UUID  `json:"id"`
	ChatID    uuid.UUID  `json:"chat"`
	MemberID  uuid.UUID  `json:"member"`
	JoinedAt  time.Time  `json:"joinedAt"`
	LeaveAt   *time.Time `json:"leaveAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updateAt"`
	Role      string     `json:"role"`
}

type Message struct {
	ID              uuid.UUID     `json:"id"`
	Text            *string       `json:"text,omitempty"`
	ParentMessageID *uuid.UUID    `json:"parentMessage,omitempty"`
	ChatID          uuid.UUID     `json:"chat"`
	Status          MessageStatus `json:"status"`
	AuthorID        *uuid.UUID    `json:"authorId,omitempty"`
	StickerID       *uuid.UUID    `json:"sticker,omitempty"`
	CreatedAt       time.Time     `json:"createdAt"`
	UpdatedAt       time.Time     `json:"updatedAt"`
	IsDeleted       bool          `json:"isDeleted"`
}

// MessageWithMedia - junction table for messages and media
type MessageWithMedia struct {
	MessageID uuid.UUID `json:"messageId"`
	MediaID   uuid.UUID `json:"mediaId"`
	Order     int       `json:"order"`
}

type Community struct {
	ID        uuid.UUID     `json:"id"`
	Title     string        `json:"title"`
	Bio       *string       `json:"bio,omitempty"`
	Type      CommunityType `json:"type"`
	OwnerID   uuid.UUID     `json:"owner"`     // Profile
	ProfileID uuid.UUID     `json:"profileId"` // Abstract-Profile
	CreatedAt time.Time     `json:"createdAt"`
	UpdatedAt time.Time     `json:"updatedAt"`
}

// CommunityMember - represents a member in a community
type CommunityMember struct {
	ID          uuid.UUID           `json:"id"`
	CommunityID uuid.UUID           `json:"community"`
	MemberID    uuid.UUID           `json:"member"`
	JoinedAt    time.Time           `json:"joinedAt"`
	LeaveAt     *time.Time          `json:"leaveAt,omitempty"`
	CreatedAt   time.Time           `json:"createdAt"`
	UpdatedAt   time.Time           `json:"updatedAt"`
	Role        CommunityMemberRole `json:"role"`
}

type Comment struct {
	ID              uuid.UUID  `json:"id"`
	Text            string     `json:"text"`
	TargetPostID    uuid.UUID  `json:"post"`
	ParentCommentID *uuid.UUID `json:"parentComment,omitempty"`
	StickerID       *uuid.UUID `json:"sticker,omitempty"`
	AuthorID        uuid.UUID  `json:"author"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
	IsDeleted       bool       `json:"isDeleted"`
}

func NewComment(text string, targetPostID uuid.UUID, parentCommentID, stickerID *uuid.UUID, authorID uuid.UUID, isDeleted bool) Comment {
	now := time.Now()

	return Comment{
		ID:              uuid.New(),
		Text:            text,
		TargetPostID:    targetPostID,
		ParentCommentID: parentCommentID,
		StickerID:       stickerID,
		AuthorID:        authorID,
		CreatedAt:       now,
		UpdatedAt:       now,
		IsDeleted:       isDeleted,
	}
}

// CommentWithMedia - junction table for comments and media
type CommentWithMedia struct {
	CommentID uuid.UUID `json:"commentId"`
	MediaID   uuid.UUID `json:"mediaId"`
	Order     int       `json:"order"`
}

type Like struct {
	ID        uuid.UUID `json:"id"`
	AuthorID  uuid.UUID `json:"author"`
	CreatedAt time.Time `json:"createdAt"`
}

func NewLike(author Profile) Like {
	return Like{
		ID:        uuid.New(),
		AuthorID:  author.ID,
		CreatedAt: time.Now(),
	}
}

// LikeToPost - junction table for likes to posts
type LikeToPost struct {
	LikeID uuid.UUID `json:"likeId"`
	PostID uuid.UUID `json:"postId"`
}

func NewLikeToPost(likeID, postID uuid.UUID) LikeToPost {
	return LikeToPost{
		LikeID: likeID,
		PostID: postID,
	}
}

// LikeToComment - junction table for likes to comments
type LikeToComment struct {
	LikeID    uuid.UUID `json:"likeId"`
	CommentID uuid.UUID `json:"commentId"`
}

type Friendship struct {
	Friend1ID uuid.UUID        `json:"friend1"`
	Friend2ID uuid.UUID        `json:"friend2"`
	Status    FriendshipStatus `json:"status"`
	CreatedAt time.Time        `json:"createdAt"`
	UpdatedAt time.Time        `json:"updatedAt"`
}

type Stickerpack struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	AuthorID  uuid.UUID
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	IsDeleted bool      `json:"isDeleted"`
}

type Sticker struct {
	ID         uuid.UUID `json:"id"`
	Link       string    `json:"link"`
	Size       int       `json:"size"`
	IndexOrder int       `json:"indexOrder"`
	PackID     uuid.UUID `json:"pack"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updateAt"`
	IsDeleted  bool      `json:"isDeleted"`
}

type Session struct {
	ID             string    `json:"id"`
	ProfileID      uuid.UUID `json:"profile"`
	CreatedAt      time.Time `json:"createdAt"`
	ExpiresAt      time.Time `json:"expiresAt"`
	LastActivityAt time.Time `json:"lastActivityAt"`
}

type Ad struct {
	ID          uuid.UUID  `json:"id"`
	Title       string     `json:"title"`
	Description *string    `json:"description,omitempty"`
	Link        string     `json:"link"`
	MediaID     *uuid.UUID `json:"media,omitempty"`
	AuthorID    uuid.UUID  `json:"author"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	IsDeleted   bool       `json:"isDeleted"`
}

// AdMeta - metadata for advertisements
type AdMeta struct {
	AdID      uuid.UUID `json:"adId"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Reaction struct {
	ID        uuid.UUID    `json:"id"`
	MessageID uuid.UUID    `json:"message"`
	Type      ReactionType `json:"type"`
	AuthorID  uuid.UUID    `json:"author"`
	CreatedAt time.Time    `json:"createdAt"`
	UpdatedAt time.Time    `json:"updatedAt"`
}
