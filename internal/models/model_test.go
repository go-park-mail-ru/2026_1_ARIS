package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewUser(t *testing.T) {
	email := "test@test.com"
	phone := "+1234567890"
	pass := "hash"
	username := "kokniside"
	user := NewUserAccount(username, &email, &phone, pass)
	assert.NotEqual(t, uuid.Nil, user.Uid)
	assert.Equal(t, "test@test.com", *user.Email)
	assert.Equal(t, "+1234567890", *user.Phone)
	assert.Equal(t, pass, user.PasswordHash)
	assert.Equal(t, username, user.Username)
	assert.False(t, user.CreatedAt.IsZero())
}

func TestNewUserProfile(t *testing.T) {
	userAccountID := int64(123)
	profileID := int64(456)
	firstName := "John"
	lastName := "Doe"
	bio := "Software Engineer"
	birthday := time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)
	gender := Male

	profile := NewUserProfile(userAccountID, profileID, firstName, lastName, &bio, birthday, gender)

	assert.NotNil(t, profile)
	assert.NotZero(t, profile.ID)
	assert.NotEqual(t, uuid.Nil, profile.Uid)
	assert.Equal(t, userAccountID, profile.UserAccountID)
	assert.Equal(t, profileID, profile.ProfileID)
	assert.Equal(t, firstName, profile.FirstName)
	assert.Equal(t, lastName, profile.LastName)
	assert.Equal(t, &bio, profile.Bio)
	assert.Equal(t, birthday, profile.BirthdayDate)
	assert.Equal(t, gender, profile.Gender)
	assert.True(t, profile.IsActive)
	assert.WithinDuration(t, time.Now(), profile.CreatedAt, time.Second)
	assert.WithinDuration(t, time.Now(), profile.UpdatedAt, time.Second)
}

func TestNewMedia(t *testing.T) {
	name := "avatar.jpg"
	extension := ".jpg"
	uid := uuid.New()
	description := "Profile picture"
	mimeType := "image/jpeg"
	link := "https://example.com/avatar.jpg"
	authorID := int64(789)

	media := NewMedia(name, extension, uid, &description, mimeType, link, authorID)

	assert.NotNil(t, media)
	assert.NotZero(t, media.ID)
	assert.Equal(t, uid, media.Uid)
	assert.Equal(t, name, media.Name)
	assert.Equal(t, extension, media.Extension)
	assert.Equal(t, authorID, media.AuthorID)
	assert.Equal(t, &description, media.Description)
	assert.Equal(t, mimeType, media.MimeType)
	assert.Equal(t, link, media.Link)
	assert.Equal(t, 0, media.Size) // Size is not set in constructor, zero by default
	assert.True(t, media.IsActive)
	assert.WithinDuration(t, time.Now(), media.CreatedAt, time.Second)
	assert.WithinDuration(t, time.Now(), media.UpdatedAt, time.Second)
}

func TestNewProfile(t *testing.T) {
	var avatarID *int64
	val := int64(100)
	avatarID = &val

	profile := NewProfile(avatarID)

	assert.NotNil(t, profile)
	assert.NotZero(t, profile.ID)
	assert.NotEqual(t, uuid.Nil, profile.Uid)
	assert.Equal(t, avatarID, profile.AvatarID)
	assert.True(t, profile.IsActive)
	assert.WithinDuration(t, time.Now(), profile.CreatedAt, time.Second)
	assert.WithinDuration(t, time.Now(), profile.UpdatedAt, time.Second)
}

func TestNewPost(t *testing.T) {
	text := "Hello, world!"
	authorID := int64(42)
	isPublicDemo := true
	allowComments := false

	post := NewPost(&text, authorID, isPublicDemo, allowComments)

	assert.NotNil(t, post)
	assert.NotZero(t, post.ID)
	assert.NotEqual(t, uuid.Nil, post.Uid)
	assert.Equal(t, &text, post.Text)
	assert.Equal(t, authorID, post.AuthorID)
	assert.Equal(t, isPublicDemo, post.IsPublicDemo)
	assert.Equal(t, allowComments, post.AllowComments)
	assert.True(t, post.IsActive)
	assert.WithinDuration(t, time.Now(), post.CreatedAt, time.Second)
	assert.WithinDuration(t, time.Now(), post.UpdatedAt, time.Second)
}

func TestNewPost_NilText(t *testing.T) {
	post := NewPost(nil, 42, true, true)

	assert.NotNil(t, post)
	assert.Nil(t, post.Text)
}

func TestNewPostWithMedia(t *testing.T) {
	postID := int64(1)
	mediaID := int64(2)
	order := 3

	junction := NewPostWithMedia(postID, mediaID, order)

	assert.NotNil(t, junction)
	assert.Equal(t, postID, junction.PostID)
	assert.Equal(t, mediaID, junction.MediaID)
	assert.Equal(t, order, junction.Order)
}

func TestNewChat(t *testing.T) {
	chatType := GroupChat
	title := "General"
	var avatarID *int64 = nil

	chat := NewChat(chatType, title, avatarID)

	assert.NotNil(t, chat)
	assert.NotZero(t, chat.ID)
	assert.NotEqual(t, uuid.Nil, chat.Uid)
	assert.Equal(t, chatType, chat.Type)
	assert.Equal(t, title, chat.Title)
	assert.Equal(t, avatarID, chat.AvatarID)
	assert.True(t, chat.IsActive)
	assert.WithinDuration(t, time.Now(), chat.CreatedAt, time.Second)
	assert.WithinDuration(t, time.Now(), chat.UpdatedAt, time.Second)
}

func TestNewComment(t *testing.T) {
	text := "Great post!"
	targetPostID := int64(100)
	parentCommentID := int64(200)
	stickerID := int64(300)
	authorID := int64(42)

	comment := NewComment(&text, targetPostID, &parentCommentID, &stickerID, authorID)

	assert.NotNil(t, comment)
	assert.NotZero(t, comment.ID)
	assert.NotEqual(t, uuid.Nil, comment.Uid)
	assert.Equal(t, &text, comment.Text)
	assert.Equal(t, targetPostID, comment.TargetPostID)
	assert.Equal(t, &parentCommentID, comment.ParentCommentID)
	assert.Equal(t, &stickerID, comment.StickerID)
	assert.Equal(t, authorID, comment.AuthorID)
	assert.True(t, comment.IsActive)
	assert.WithinDuration(t, time.Now(), comment.CreatedAt, time.Second)
	assert.WithinDuration(t, time.Now(), comment.UpdatedAt, time.Second)
}

func TestNewComment_NilText(t *testing.T) {
	comment := NewComment(nil, 100, nil, nil, 42)

	assert.NotNil(t, comment)
	assert.Nil(t, comment.Text)
}

func TestNewLikeToPost(t *testing.T) {
	postID := int64(123)
	authorID := int64(456)

	like := NewLikeToPost(postID, authorID)

	assert.NotNil(t, like)
	assert.NotZero(t, like.ID)
	assert.NotEqual(t, uuid.Nil, like.Uid)
	assert.Equal(t, &postID, like.PostID)
	assert.Nil(t, like.CommentID)
	assert.Equal(t, authorID, like.AuthorID)
	assert.True(t, like.IsActive)
	assert.WithinDuration(t, time.Now(), like.CreatedAt, time.Second)
	assert.WithinDuration(t, time.Now(), like.UpdatedAt, time.Second)
}

func TestNewLikeToComment(t *testing.T) {
	commentID := int64(789)
	authorID := int64(101)

	like := NewLikeToComment(commentID, authorID)

	assert.NotNil(t, like)
	assert.NotZero(t, like.ID)
	assert.NotEqual(t, uuid.Nil, like.Uid)
	assert.Nil(t, like.PostID)
	assert.Equal(t, &commentID, like.CommentID)
	assert.Equal(t, authorID, like.AuthorID)
	assert.True(t, like.IsActive)
	assert.WithinDuration(t, time.Now(), like.CreatedAt, time.Second)
	assert.WithinDuration(t, time.Now(), like.UpdatedAt, time.Second)
}

func TestNewRepost(t *testing.T) {
	authorID := int64(111)
	chatID := int64(222)
	postID := int64(333)

	repost := NewRepost(authorID, chatID, postID)

	assert.NotNil(t, repost)
	assert.NotZero(t, repost.ID)
	assert.NotEqual(t, uuid.Nil, repost.Uid)
	assert.Equal(t, authorID, repost.AuthorID)
	assert.Equal(t, chatID, repost.ChatID)
	assert.Equal(t, postID, repost.PostID)
}
