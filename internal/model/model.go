package model

import (
	"time"
)

type ID int64

type UserID ID
type MediaID ID
type DocumentID ID
type ProfileID ID
type ChatID ID
type GroupID ID
type LikeID ID
type FriendshipID ID
type StickerID ID
type MessageID ID
type PostID ID
type CommentID ID
type StickerpackID ID
type SessionID ID

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

type FriendshipStatus int

const (
	FriendshipPending FriendshipStatus = iota
	FriendshipAccepted
	FriendshipDeclined
	FriendshipBlocked
)

type MediaType int

const (
	Image MediaType = iota
	Video
)

// models structs

type User struct {
	ID        UserID    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	Password  string    `json:"-"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	IsActive  bool      `json:"isActive"`
}

func NewUser(id UserID, username, email, phone, password string) User {
	return User{
		ID:        id,
		Username:  username,
		Email:     email,
		Phone:     phone,
		Password:  password,
		CreatedAt: time.Now(),
		IsActive:  true,
	}
}

type Profile struct {
	ID            ProfileID  `json:"id"`
	FirstName     string     `json:"firstName"`
	LastName      string     `json:"lastName"`
	Bio           string     `json:"bio"`
	BirthdayDate  *time.Time `json:"birthdayDate"`
	Gender        string     `json:"gender"`
	ProfileUserID UserID     `json:"user"`
	AvatarID      *MediaID   `json:"avatar,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
	IsActive      bool       `json:"isActive"`
}

func NewProfile(id ProfileID, firstName, lastName, bio string, birthday *time.Time, gender string, userID UserID, avatarID *MediaID) Profile {
	return Profile{
		ID:            id,
		FirstName:     firstName,
		LastName:      lastName,
		Bio:           bio,
		BirthdayDate:  birthday,
		Gender:        gender,
		ProfileUserID: userID,
		AvatarID:      avatarID,
		IsActive:      true,
	}
}

// Необходимо валидировать только один источник загрузки: Prorile или Group
type Media struct {
	ID        MediaID    `json:"id"`
	Type      MediaType  `json:"type"`
	Extension string     `json:"extension"`
	Link      string     `json:"link"`
	Size      int        `json:"size"`
	Height    int        `json:"height"`
	Width     int        `json:"width"`
	ProfileID *ProfileID `json:"profile,omitempty"`
	GroupID   *GroupID   `json:"group,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	IsDeleted bool       `json:"isDeleted"`
}

// Необходимо валидировать только один источник загрузки: Prorile или Group
type Document struct {
	ID        DocumentID `json:"id"`
	Link      string     `json:"link"`
	Extension string     `json:"extension"`
	Size      int        `json:"size"`
	ProfileID *ProfileID `json:"profile,omitempty"`
	GroupID   *GroupID   `json:"group,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	IsDeleted bool       `json:"isDeleted"`
}

// Необходимо валидировать только один источник публикации: Prorile или Group
// Необходимо валидировать только один тип контента: Media или Document
type Post struct {
	ID          PostID       `json:"id"`
	Text        string       `json:"text,omitempty"`
	ViewCount   int          `json:"viewCount"`
	LikeCount   int          `json:"likeCount"`
	RepostCount int          `json:"repostCount"`
	ProfileID   *ProfileID   `json:"profile,omitempty"`
	GroupID     *GroupID     `json:"group,omitempty"`
	MediaIDs    []MediaID    `json:"media,omitempty"`
	DocumentIDs []DocumentID `json:"documents,omitempty"`
	CreatedAt   time.Time    `json:"createdAt"`
	UpdatedAt   time.Time    `json:"updatedAt"`
	IsActive    bool         `json:"isActive"`
}

func NewPost(id PostID, text string, viewCount, likeCount, repostCount int, profileID *ProfileID, groupID *GroupID, mediasIDs *[]MediaID, documentIDs *[]DocumentID) Post {
	return Post{
		ID:          id,
		Text:        text,
		ViewCount:   viewCount,
		LikeCount:   likeCount,
		RepostCount: repostCount,
		ProfileID:   profileID,
		GroupID:     groupID,
		MediaIDs:    *mediasIDs,
		DocumentIDs: *documentIDs,
		CreatedAt:   time.Now(),
		IsActive:    true,
	}
}

type ChatMember struct {
	ID        ChatID    `json:"chat"`
	ProfileID ProfileID `json:"profile"`
	JoinedAt  time.Time `json:"joinedAt"`
	Role      string    `json:"role"`
}

type Chat struct {
	ID        ChatID       `json:"id"`
	Type      ChatType     `json:"type"`
	Title     string       `json:"title"`
	AvatarID  *MediaID     `json:"avatar,omitempty"`
	MemberIDs []ChatMember `json:"members,omitempty"`
	CreatedAt time.Time    `json:"createdAt"`
}

// Необходимо валидировать только одного отправителя сообщения: Prorile или Group
// Необходимо валидировать только один тип контента: Media или Document
type Message struct {
	ID            MessageID    `json:"id"`
	Text          string       `json:"text"`
	ParentMessage *MessageID   `json:"parentMessage,omitempty"`
	ChatID        ChatID       `json:"chat"`
	ProfileID     *ProfileID   `json:"profile,omitempty"`
	GroupID       *GroupID     `json:"group,omitempty"`
	MediaIDs      []MediaID    `json:"media,omitempty"`
	DocumentIDs   []DocumentID `json:"documents,omitempty"`
	CreatedAt     time.Time    `json:"createdAt"`
	UpdatedAt     time.Time    `json:"updatedAt"`
	IsDeleted     bool         `json:"isDeleted"`
}

type GroupMember struct {
	ID        GroupID   `json:"group"`
	ProfileID ProfileID `json:"profile"`
	JoinedAt  time.Time `json:"joinedAt"`
	Role      string    `json:"role"`
}

type Group struct {
	ID          GroupID       `json:"id"`
	Title       string        `json:"title"`
	Description string        `json:"description,omitempty"`
	Type        GroupType     `json:"type"`
	MemberCount int           `json:"memberCount"`
	PostCount   int           `json:"postCount"`
	OwnerID     ProfileID     `json:"owner"`
	MemberIDs   []GroupMember `json:"members,omitempty"`
	AvatarID    *MediaID      `json:"avatar,omitempty"`
	CreatedAt   time.Time     `json:"createdAt"`
	UpdatedAt   time.Time     `json:"updatedAt"`
	IsDeleted   bool          `json:"isDeleted"`
}

// Необходимо валидировать только один одного автора комментария: Prorile или Group
// Необходимо валидировать только один тип контента: Media или Document
type Comment struct {
	ID              CommentID    `json:"id"`
	Text            string       `json:"text"`
	LikeCount       int          `json:"likeCount"`
	TargetPostID    PostID       `json:"post"`
	ParentCommentID *CommentID   `json:"parentComment,omitempty"`
	ProfileID       *ProfileID   `json:"profile,omitempty"`
	GroupID         *GroupID     `json:"group,omitempty"`
	MediaIDs        []MediaID    `json:"media,omitempty"`
	DocumentIDs     []DocumentID `json:"documents,omitempty"`
	CreatedAt       time.Time    `json:"createdAt"`
	UpdatedAt       time.Time    `json:"updatedAt"`
	IsDeleted       bool         `json:"isDeleted"`
}

type Like struct {
	ID              LikeID     `json:"id"`
	ProfileID       *ProfileID `json:"profile,omitempty"`
	GroupID         *GroupID   `json:"group,omitempty"`
	TargetPostID    *PostID    `json:"post,omitempty"`
	TargetCommentID *CommentID `json:"comment,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
}

type Friendship struct {
	ID        FriendshipID     `json:"id"`
	Status    FriendshipStatus `json:"status"`
	Friend1ID ProfileID        `json:"friend1"`
	Friend2ID ProfileID        `json:"friend2"`
	CreatedAt time.Time        `json:"createdAt"`
	UpdatedAt time.Time        `json:"updatedAt"`
}

type Stickerpack struct {
	ID           StickerpackID `json:"id"`
	Title        string        `json:"title"`
	StickerCount int           `json:"stickerCount"`
	CreatedAt    time.Time     `json:"createdAt"`
	UpdatedAt    time.Time     `json:"updatedAt"`
	IsDeleted    bool          `json:"isDeleted"`
}

type Sticker struct {
	ID         StickerID     `json:"id"`
	Link       string        `json:"link"`
	Size       int           `json:"size"`
	IndexOrder int           `json:"indexOrder"`
	PackID     StickerpackID `json:"pack"`
	CreatedAt  time.Time     `json:"createdAt"`
	IsDeleted  bool          `json:"isDeleted"`
}

type Session struct {
	SessionID ID        `json:"id"`
	UserID    UserID    `json:"user"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiredAt time.Time `json:"expiredAt"`
}
