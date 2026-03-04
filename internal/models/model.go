package models

import (
	"time"

	"github.com/google/uuid"
)

// models Types

type ChatType int

const (
	PrivateChat ChatType = iota
	GroupChat
)

type GroupType int

const (
	PublicGroup GroupType = iota
	PrivateGroup
)

type GroupRole int

const (
	Admin GroupRole = iota
	Manager
	Member
)

type FriendshipStatus int

const (
	FriendshipPending FriendshipStatus = iota
	FriendshipAccepted
	FriendshipDeclined
	FriendshipBlocked
)

type ReactionType int

const (
	ReactionLike ReactionType = iota
	ReactionLove
	ReactionLaugh
	ReactionSad
	ReactionAngry
)

type Gender int

const (
	male Gender = iota
	female
)

type MessageStatus int

const (
	NotSend MessageStatus = iota
	Senging
	Send
	Read
)

// models structs

type User struct {
	ID           uuid.UUID `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	Phone        string    `json:"phone"`
	PasswordHash string    `json:"-"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func NewUser(username, email, phone, passwordHash string) User {
	userID := uuid.New()
	return User{
		ID:           userID,
		Username:     username,
		Email:        email,
		Phone:        phone,
		PasswordHash: passwordHash,
		UpdatedAt:    time.Now(),
	}
}

// UserProfile - user-specific profile information
type UserProfile struct {
	UserID       uuid.UUID  `json:"userId"`
	FirstName    string     `json:"firstName"`
	LastName     string     `json:"lastName"`
	Bio          string     `json:"bio"`
	BirthdayDate *time.Time `json:"birthdayDate,omitempty"`
	Gender       Gender     `json:"gender"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

func NewUserProfile(user User, firstName, lastName, bio string, birthday *time.Time, gender Gender) UserProfile {
	return UserProfile{
		UserID:       user.ID,
		FirstName:    firstName,
		LastName:     lastName,
		Bio:          bio,
		BirthdayDate: birthday,
		Gender:       gender,
		UpdatedAt:    time.Now(),
	}
}

// Media type with properties from schema
type Media struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Extension   string    `json:"extension"`
	Description string    `json:"description,omitempty"`
	MimeType    string    `json:"mimeType"`
	Link        string    `json:"link"`
	Size        int       `json:"size"`
	CreatedAt   time.Time `json:"createdAt"`
	IsDeleted   bool      `json:"isDeleted"`
}

func NewMedia(name, extension, description, mimeType, link string, size int, isDeleted bool) Media {
	mediaID := uuid.New()

	return Media{
		ID:          mediaID,
		Name:        name,
		Extension:   extension,
		Description: description,
		MimeType:    mimeType,
		Link:        link,
		Size:        size,
		CreatedAt:   time.Now(),
		IsDeleted:   isDeleted,
	}
}

// Abstract profile for both users and groups
type Profile struct {
	ID        uuid.UUID  `json:"id"`
	AvatarID  *uuid.UUID `json:"avatar,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	IsActive  bool       `json:"isActive"`
}

func NewProfile(avatar *Media, isActive bool) Profile {
	var avatarID *uuid.UUID

	if avatar != nil {
		avatarID = &avatar.ID
	}

	profileID := uuid.New()

	now := time.Now()

	return Profile{
		ID:        profileID,
		AvatarID:  avatarID,
		CreatedAt: now,
		UpdatedAt: now,
		IsActive:  isActive,
	}
}

// GroupProfile - group-specific profile information
type ProfileGroup struct {
	ProfileID uuid.UUID `json:"profileId"`
	GroupID   uuid.UUID `json:"profileGroupId"`
}

func NewProfileGroup(profile Profile, group Group) ProfileGroup {
	return ProfileGroup{
		profile.ID,
		group.ID,
	}
}

type ProfileUser struct {
	ProfileID            uuid.UUID `json:"profileId"`
	ProfileUserPforileID uuid.UUID `json:"profileUserId"`
}

func NewProfileUser(profile Profile, userProfile UserProfile) ProfileUser {
	return ProfileUser{
		ProfileID:            profile.ID,
		ProfileUserPforileID: userProfile.UserID,
	}
}

// Post model matching schema
type Post struct {
	ID        uuid.UUID `json:"id"`
	Text      string    `json:"text,omitempty"`
	AuthorID  uuid.UUID `json:"authorId"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	IsActive  bool      `json:"isActive"`
}

func NewPost(text string, author Profile, isActive bool) Post {
	postID := uuid.New()

	now := time.Now()

	return Post{
		ID:        postID,
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

// Chat model matching schema
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
	ProfileID uuid.UUID  `json:"profile"`
	JoinedAt  time.Time  `json:"joinedAt"`
	LeaveAt   *time.Time `json:"leaveAt,omitempty"`
	Role      string     `json:"role"`
}

// Message model matching schema
type Message struct {
	ID              uuid.UUID     `json:"id"`
	Text            string        `json:"text"`
	ParentMessageID *uuid.UUID    `json:"parentMessage,omitempty"`
	ChatID          uuid.UUID     `json:"chat"`
	Status          MessageStatus `json:"status"`
	ProfileID       *uuid.UUID    `json:"profile,omitempty"`
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

// Group model matching schema
type Group struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Type        GroupType `json:"type"`
	OwnerID     uuid.UUID `json:"owner"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// GroupMember - represents a member in a group
type GroupMember struct {
	ID        uuid.UUID  `json:"id"`
	GroupID   uuid.UUID  `json:"group"`
	ProfileID uuid.UUID  `json:"profile"`
	JoinedAt  time.Time  `json:"joinedAt"`
	LeaveAt   *time.Time `json:"leaveAt,omitempty"`
	Role      GroupRole  `json:"role"`
}

// Comment model matching schema
type Comment struct {
	ID              uuid.UUID  `json:"id"`
	Text            string     `json:"text"`
	TargetPostID    uuid.UUID  `json:"post"`
	ParentCommentID *uuid.UUID `json:"parentComment,omitempty"`
	StickerID       *uuid.UUID `json:"sticker,omitempty"`
	ProfileID       *uuid.UUID `json:"profile,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
	IsDeleted       bool       `json:"isDeleted"`
}

// CommentWithMedia - junction table for comments and media
type CommentWithMedia struct {
	CommentID uuid.UUID `json:"commentId"`
	MediaID   uuid.UUID `json:"mediaId"`
	Order     int       `json:"order"`
}

// Like model - represents a like action
type Like struct {
	ID        uuid.UUID `json:"id"`
	AuthorID  uuid.UUID `json:"author"`
	CreatedAt time.Time `json:"createdAt"`
}

// LikeToPost - junction table for likes to posts
type LikeToPost struct {
	LikeID uuid.UUID `json:"likeId"`
	PostID uuid.UUID `json:"postId"`
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
	IsDeleted  bool      `json:"isDeleted"`
}

// Session model
type Session struct {
	ID             string    `json:"id"`
	ProfileID      uuid.UUID `json:"profile"`
	ExpiredAt      time.Time `json:"expiredAt"`
	CreatedAt      time.Time `json:"createdAt"`
	ExpiresAt      time.Time `json:"expiresAt"`
	LastActivityAt time.Time `json:"lastActivityAt"`
}

// Ad model - represents an advertisement
type Ad struct {
	ID          uuid.UUID  `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	Link        string     `json:"link"`
	MediaID     *uuid.UUID `json:"media,omitempty"`
	AuthorID    uuid.UUID  `json:"author"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	IsDeleted   bool       `json:"isDeleted"`
}

// AdMeta - metadata for advertisements
type AdMeta struct {
	AdID  uuid.UUID `json:"adId"`
	Key   string    `json:"key"`
	Value string    `json:"value"`
}

// Reaction model - represents a reaction to a message
type Reaction struct {
	ID        uuid.UUID    `json:"id"`
	MessageID uuid.UUID    `json:"message"`
	Type      ReactionType `json:"type"`
	AuthorID  uuid.UUID    `json:"author"`
	CreatedAt time.Time    `json:"createdAt"`
}
