package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestConstructors(t *testing.T) {
	email := "user@example.com"
	phone := "+79990000000"
	bio := "bio"
	description := "desc"
	text := "hello"
	avatarID := int64(42)

	userAccount := NewUserAccount("tester", &email, &phone, "hash")
	require.Equal(t, "tester", userAccount.Username)
	require.Equal(t, &email, userAccount.Email)
	require.Equal(t, &phone, userAccount.Phone)
	require.Equal(t, "hash", userAccount.PasswordHash)
	require.True(t, userAccount.IsActive)
	require.NotEqual(t, uuid.Nil, userAccount.Uid)

	birthday := time.Date(2000, time.January, 2, 0, 0, 0, 0, time.UTC)
	userProfile := NewUserProfile(1, 2, "Ivan", "Petrov", &bio, birthday, Male)
	require.Equal(t, int64(1), userProfile.UserAccountID)
	require.Equal(t, int64(2), userProfile.ProfileID)
	require.Equal(t, "Ivan", userProfile.FirstName)
	require.Equal(t, "Petrov", userProfile.LastName)
	require.Equal(t, &bio, userProfile.Bio)
	require.Equal(t, birthday, userProfile.BirthdayDate)
	require.Equal(t, Male, userProfile.Gender)
	require.True(t, userProfile.IsActive)

	mediaUID := uuid.New()
	media := NewMedia("pic", "jpg", mediaUID, &description, "image/jpeg", "https://example.com", 7)
	require.Equal(t, mediaUID, media.Uid)
	require.Equal(t, "pic", media.Name)
	require.Equal(t, "jpg", media.Extension)
	require.Equal(t, &description, media.Description)
	require.Equal(t, "image/jpeg", media.MimeType)
	require.Equal(t, "https://example.com", media.Link)
	require.Equal(t, int64(7), media.AuthorID)
	require.True(t, media.IsActive)

	profile := NewProfile(&avatarID)
	require.Equal(t, &avatarID, profile.AvatarID)
	require.True(t, profile.IsActive)
	require.NotEqual(t, uuid.Nil, profile.Uid)

	post := NewPost(&text, 9, true, false)
	require.Equal(t, &text, post.Text)
	require.Equal(t, int64(9), post.AuthorID)
	require.True(t, post.IsPublicDemo)
	require.False(t, post.AllowComments)
	require.True(t, post.IsActive)

	postWithMedia := NewPostWithMedia(11, 12, 3)
	require.Equal(t, int64(11), postWithMedia.PostID)
	require.Equal(t, int64(12), postWithMedia.MediaID)
	require.Equal(t, 3, postWithMedia.Order)
}
