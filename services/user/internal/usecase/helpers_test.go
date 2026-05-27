package usecase

import (
	"errors"
	"testing"
	"time"

	cache "github.com/go-park-mail-ru/2026_1_ARIS/pkg/tarantool"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/user/internal/model"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/user/internal/repository"
	"github.com/stretchr/testify/require"
)

func TestUserUsecaseNormalizationHelpers(t *testing.T) {
	t.Parallel()

	require.Equal(t, "user", normalizeUsername(" User "))
	require.Equal(t, "vk42", normalizeOAuthUsername(" VK-42! "))
	require.Empty(t, normalizeOAuthUsername("x"))
	require.Len(t, oauthUsernameCandidate("verylongpreferredusername", "vk", "42", 1), 20)
	require.Equal(t, "short", oauthUsernameCandidate("short", "vk", "42", 0))
	require.Equal(t, "Иван", normalizeProfileName(" Иван123 ", "fallback"))
	require.Equal(t, "fallback", normalizeProfileName("123", "fallback"))
	require.True(t, isProfileNameRune('Я'))
	require.False(t, isProfileNameRune('1'))

	email := " test@example.com "
	require.Equal(t, "test@example.com", *normalizeOptionalEmail(&email))
	blank := " "
	require.Nil(t, normalizeOptionalEmail(&blank))
	require.Nil(t, normalizeOptionalEmail(nil))

	require.Equal(t, time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC), oauthBirthday("2000-01-02"))
	require.Equal(t, 1970, oauthBirthday("bad").Year())
	require.Contains(t, oauthPasswordHash("vk", "42"), "oauth:")
	require.Equal(t, model.Male, normalizeGender(model.Male))
	require.Equal(t, model.Female, normalizeGender(model.Gender("other")))
	require.Equal(t, "x", derefString(stringPtr("x")))
	require.Empty(t, derefString(nil))
	require.Equal(t, "query", normalizeSearchQuery(" query "))
}

func TestUserUsecaseCacheMappers(t *testing.T) {
	t.Parallel()

	now := time.Now()
	email := "test@example.com"
	avatarID := int64(55)
	user := &AuthUser{
		UserAccountID: 1,
		UserProfileID: 2,
		ProfileID:     3,
		Login:         "login",
		Email:         &email,
		FirstName:     "First",
		LastName:      "Last",
		AvatarID:      &avatarID,
		CreatedAt:     now,
		IsOnline:      true,
		LastSeenAt:    &now,
	}

	cachedUser := authUserToCache(user)
	require.Equal(t, user.UserAccountID, cachedUser.UserAccountID)
	require.Equal(t, user.Login, authUserFromCache(&cachedUser).Login)
	require.Nil(t, authUserFromCache(nil))
	require.Zero(t, authUserToCache(nil).UserAccountID)

	summary := summaryFromAuthUser(user)
	require.Equal(t, user.ProfileID, summary.ProfileID)
	require.Equal(t, user.Login, summary.Username)
	require.Equal(t, user.Login, authUserFromSummary(&summary).Login)
	require.Nil(t, authUserFromSummary(nil))
	require.Zero(t, summaryFromAuthUser(nil).ProfileID)

	details := &ProfileDetails{
		ProfileID:     3,
		UserProfileID: 2,
		UserAccountID: 1,
		Username:      "login",
		AvatarID:      &avatarID,
		FirstName:     "First",
		LastName:      "Last",
		Email:         &email,
		Gender:        model.Male,
		BirthdayDate:  now,
		Education:     []Education{{Institution: stringPtr("MSTU"), Group: stringPtr("IU")}},
		Work:          []Work{{Company: stringPtr("ARIS"), JobTitle: stringPtr("dev")}},
		IsOnline:      true,
		LastSeenAt:    &now,
	}
	require.Equal(t, details.Username, summaryFromProfileDetails(details).Username)
	require.Zero(t, summaryFromProfileDetails(nil).ProfileID)

	cachedDetails := profileDetailsToCache(details)
	require.Len(t, cachedDetails.Education, 1)
	require.Equal(t, model.Male, profileDetailsFromCache(&cachedDetails).Gender)
	require.Nil(t, profileDetailsFromCache(nil))
	require.Zero(t, profileDetailsToCache(nil).ProfileID)
}

func TestUserUsecaseSettingsAndErrors(t *testing.T) {
	t.Parallel()

	settings := defaultSettings(42)
	require.Equal(t, model.LanguageRU, settings.Language)
	require.Equal(t, model.ThemeLight, settings.Theme)

	language := model.LanguageEN
	theme := model.ThemeDark
	applySettingsUpdate(settings, repository.SettingsUpdate{Language: &language, Theme: &theme})
	require.Equal(t, model.LanguageEN, settings.Language)
	require.Equal(t, model.ThemeDark, settings.Theme)

	require.True(t, validFriendshipStatus(string(model.FriendshipPending)))
	require.True(t, validFriendshipStatus(string(model.FriendshipAccepted)))
	require.False(t, validFriendshipStatus("bad"))

	require.NoError(t, normalizeFriendshipMutationError(nil))
	require.ErrorIs(t, normalizeFriendshipMutationError(repository.ErrNoRowsAffected), ErrFriendshipNotExists)
	require.ErrorIs(t, normalizeFriendshipMutationError(repository.ErrFriendshipNotFound), ErrFriendshipNotExists)
	require.ErrorIs(t, normalizeFriendshipMutationError(repository.ErrAlreadyExists), ErrAlreadyFriends)
	boom := errors.New("boom")
	require.ErrorIs(t, normalizeFriendshipMutationError(boom), boom)

	require.ErrorIs(t, normalizeAccountError(repository.ErrUserAccountNotFound), ErrUserAccountNotFound)
	require.ErrorIs(t, normalizeProfileError(repository.ErrProfileNotFound), ErrProfileNotFound)
	require.ErrorIs(t, normalizeUserProfileError(repository.ErrUserProfileNotFound), ErrUserProfileNotFound)
	require.ErrorIs(t, normalizeAccountError(boom), boom)
	require.ErrorIs(t, normalizeProfileError(boom), boom)
	require.ErrorIs(t, normalizeUserProfileError(boom), boom)

	_ = cache.ProfileSummary{}
}

func stringPtr(value string) *string {
	return &value
}
