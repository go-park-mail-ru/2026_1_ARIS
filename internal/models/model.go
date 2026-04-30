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
	PrivateChat ChatType = "personal"
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
)

type ReactionType string

const (
	ReactionLike  ReactionType = "👍"
	ReactionLove  ReactionType = "❤️"
	ReactionLaugh ReactionType = "😂"
	ReactionSad   ReactionType = "😢"
	ReactionAngry ReactionType = "😡"
)

// type Gender int

type Gender string

const (
	Male   Gender = "male"
	Female Gender = "female"
)

type MessageStatus int

const (
	NotSend MessageStatus = iota
	Sending
	Send
	Read
)

type LanguageSetting string

const (
	LanguageRU LanguageSetting = "RU"
	LanguageEN LanguageSetting = "EN"
)

type ThemeSetting string

const (
	ThemeLight ThemeSetting = "light"
	ThemeDark  ThemeSetting = "dark"
)

type TicketCategory int

const (
	CategoryBug TicketCategory = iota
	CategoryFeatureRequest
	CotegoryComplaint
	CategoryQuestion
	CategoryOther
)

type TicketStatus int

const (
	TicketStatusOpen TicketStatus = iota
	TicketStatusInProgress
	TicketStatusWaitingUser
	TicketStatusClosed
)

type TicketPriority int

const (
	TicketPriorityLow TicketPriority = iota
	TicketPriorityMedium
	TicketPriorityHigh
)

type SupportRole string

const (
	SupportRoleUser      SupportRole = "user"
	SupportRoleSupportL1 SupportRole = "support_l1"
	SupportRoleSupportL2 SupportRole = "support_l2"
	SupportRoleAdmin     SupportRole = "admin"
)

type SupportProfileRole struct {
	ProfileID int64       `db:"profile_id"`
	Role      SupportRole `db:"role"`
}

type SupportTicket struct {
	ID              int64          `db:"id"`
	Uid             uuid.UUID      `db:"uid"`
	ProfileID       int64          `db:"profile_id"`
	Login           string         `db:"login"`
	Email           string         `db:"email"`
	Category        TicketCategory `db:"category"`
	Title           string         `db:"title"`
	Description     string         `db:"description"`
	Status          TicketStatus   `db:"status"`
	Priority        TicketPriority `db:"priority"`
	Line            int            `db:"line"`
	AssignedAgentID *int64         `db:"assigned_agent_id"`
	Rating          *int           `db:"rating"`
	CreatedAt       time.Time      `db:"created_at"`
	UpdatedAt       time.Time      `db:"updated_at"`
	ClosedAt        *time.Time     `db:"closed_at"`
}

func NewSupportTicket(profileID int64, login, email string, category TicketCategory, title, description string) *SupportTicket {
	now := time.Now()

	return &SupportTicket{
		ID:          rand.Int64(),
		Uid:         uuid.New(),
		ProfileID:   profileID,
		Login:       login,
		Email:       email,
		Category:    category,
		Title:       title,
		Description: description,
		Status:      TicketStatusOpen,
		CreatedAt:   now,
		UpdatedAt:   now,
		ClosedAt:    nil,
		Priority:    TicketPriorityLow,
		Line:        1,
	}
}

type TicketWithMedia struct {
	TicketID int64 `db:"ticket_id"`
	MediaID  int64 `db:"media_id"`
	Order    int   `db:"sort_order"`
}

func NewTicketWithMedia(ticketID, mediaID int64, order int) *TicketWithMedia {
	return &TicketWithMedia{
		TicketID: ticketID,
		MediaID:  mediaID,
		Order:    order,
	}
}

type SupportTicketStats struct {
	TotalCount         int64            `json:"total"`
	OpenCount          int64            `json:"open"`
	InProgressCount    int64            `json:"inProgress"`
	WaitingUserCount   int64            `json:"waitingUser"`
	ClosedCount        int64            `json:"closed"`
	ByCategory         map[string]int64 `json:"byCategory"`
	ByLine             map[string]int64 `json:"byLine"`
	AverageRating      *float64         `json:"avgRating"`
	RatingDistribution map[string]int64 `json:"ratingDistribution"`
}

type SupportTicketMessage struct {
	ID         int64       `db:"id"`
	TicketID   int64       `db:"ticket_id"`
	Text       string      `db:"text"`
	AuthorID   int64       `db:"author_id"`
	AuthorRole SupportRole `db:"author_role"`
	CreatedAt  time.Time   `db:"created_at"`
}

// models structs
// credentials данные
type UserAccount struct {
	ID           int64     `db:"id"`
	Uid          uuid.UUID `db:"uid"`
	Username     string    `db:"username"`
	Email        *string   `db:"email"`
	Phone        *string   `db:"phone"`
	PasswordHash string    `db:"password_hash"`
	IsActive     bool      `db:"is_active"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
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
	ID            int64     `db:"id" json:"id"`
	Uid           uuid.UUID `db:"uid" json:""`
	UserAccountID int64     `db:"user_account_id" json:"userAccountId"`
	ProfileID     int64     `db:"profile_id" json:"profileId"`
	FirstName     string    `db:"first_name" json:"firstName"`
	LastName      string    `db:"last_name" json:"lastName"`
	Bio           *string   `db:"bio,omitempty" json:"bio,omitempty"`
	BirthdayDate  time.Time `db:"birthday_date,omitempty" json:"birthdayDate"`
	Gender        Gender    `db:"gender" json:"gender"`

	NativeTown  *string `db:"native_town,omitempty" json:"nativeTown,omitempty"`
	Town        *string `db:"town,omitempty" json:"town,omitempty"`
	Institution *string `db:"institution,omitempty" json:"institution,omitempty"`
	Group       *string `db:"study_group,omitempty" json:"group,omitempty"`
	Company     *string `db:"company,omitempty" json:"company,omitempty"`
	JobTitle    *string `db:"job_title,omitempty" json:"jobTitle,omitempty"`
	Interests   *string `db:"interests,omitempty" json:"interests,omitempty"`
	FavMusic    *string `db:"fav_music,omitempty" json:"favMusic,omitempty"`

	IsActive  bool      `db:"is_active" json:"isActive"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
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
	ID          int64     `db:"id"`
	Uid         uuid.UUID `db:"uid"`
	Name        string    `db:"media_name"`
	AuthorID    int64     `db:"author_id"`
	Extension   string    `db:"extension"`
	Description *string   `db:"description,omitempty"`
	MimeType    string    `db:"mime_type"`
	Link        string    `db:"link"`
	Size        int       `db:"size"`
	IsActive    bool      `db:"is_active"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

func NewMedia(name, extension string, uid uuid.UUID, description *string, mimeType, link string, authorID int64) *Media {
	now := time.Now()
	return &Media{
		ID:          rand.Int64(),
		Uid:         uid,
		Name:        name,
		Extension:   extension,
		Description: description,
		MimeType:    mimeType,
		Link:        link,
		AuthorID:    authorID,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// Abstract profile for both users and groups
type Profile struct {
	ID        int64     `db:"id"`
	Uid       uuid.UUID `db:"uid"`
	AvatarID  *int64    `db:"avatar_id,omitempty"`
	IsActive  bool      `db:"is_active"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
	//LastSeenAt time.Time `db:"last_seen_at"`
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
	ID            int64     `db:"id"`
	Uid           uuid.UUID `db:"uid"`
	Text          *string   `db:"post_text,omitempty"`
	AuthorID      int64     `db:"author_id"` // to Profile
	IsPublicDemo  bool      `db:"is_public_demo"`
	AllowComments bool      `db:"allow_comments"`
	IsActive      bool      `db:"is_active"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}

func NewPost(text *string, authorID int64, isPublicDemo, allowComments bool) *Post {
	now := time.Now()

	return &Post{
		ID:            rand.Int64(),
		Uid:           uuid.New(),
		Text:          text,
		AuthorID:      authorID,
		IsActive:      true,
		IsPublicDemo:  isPublicDemo,
		AllowComments: allowComments,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// PostWithMedia - junction table for posts and media
type PostWithMedia struct {
	PostID  int64 `db:"post_id"`
	MediaID int64 `db:"media_id"`
	Order   int   `db:"sort_order"`
}

func NewPostWithMedia(postID, mediaID int64, order int) *PostWithMedia {
	return &PostWithMedia{
		PostID:  postID,
		MediaID: mediaID,
		Order:   order,
	}
}

type Chat struct {
	ID        int64     `db:"id"`
	Uid       uuid.UUID `db:"uid"`
	Type      ChatType  `db:"chat_type"`
	Title     string    `db:"title"`
	AvatarID  *int64    `db:"avatar_id,omitempty"`
	IsActive  bool      `db:"is_active"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func NewChat(chat_type ChatType, title string, avatarID *int64) *Chat {
	now := time.Now()
	return &Chat{
		ID:        rand.Int64(),
		Uid:       uuid.New(),
		Type:      chat_type,
		Title:     title,
		AvatarID:  avatarID,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// ChatMember - represents a member in a chat
type ChatMember struct {
	ID        int64      `db:"id" json:"id"`
	Uid       uuid.UUID  `db:"uid" json:"uid"`
	ChatID    int64      `db:"chat_id" json:"chat"`
	MemberID  int64      `db:"profile_id" json:"member"`
	JoinedAt  time.Time  `db:"joined_at" json:"joinedAt"`
	IsActive  bool       `json:"isActive"`
	LeaveAt   *time.Time `db:"leave_at" json:"leaveAt,omitempty"`
	CreatedAt time.Time  `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time  `db:"updated_at" json:"updatedAt"`
	Role      string     `db:"chat_role" json:"role"`
}

type Message struct {
	ID              int64     `db:"id" json:"id"`
	Uid             uuid.UUID `db:"uid" json:"uid"`
	Text            *string   `db:"message_text" json:"text,omitempty"`
	ParentMessageID *int64    `db:"parent_message_id" json:"parentMessage,omitempty"`
	ChatID          int64     `db:"chat_id" json:"chat"`
	AuthorID        int64     `db:"author_id" json:"authorId,omitempty"`
	StickerID       *int64    `db:"sticker_id" json:"sticker,omitempty"`
	IsActive        bool      `db:"is_active" json:"isActive"`
	CreatedAt       time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt       time.Time `db:"updated_at" json:"updatedAt"`
}

type UserMessageStatus struct {
	ProfileID int64
	MessageID int64
	Status    MessageStatus
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
	ID        int64     `db:"id"`
	Uid       uuid.UUID `db:"uid"`
	PostID    *int64    `db:"post_id"`
	CommentID *int64    `db:"comment_id"`
	AuthorID  int64     `db:"author_id"`
	IsActive  bool      `db:"is_active"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
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
	RequesterID int64            `json:"requester"`
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
	UpdatedAt  time.Time `json:"updatedAt"`
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

type UserSettings struct {
	UserAccountID int64           `db:"user_account_id" json:"userAccountID"`
	Language      LanguageSetting `db:"lang" json:"language"`
	Theme         ThemeSetting    `db:"theme" json:"theme"`
}
