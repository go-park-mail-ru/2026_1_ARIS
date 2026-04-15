package dto

import (
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/stretchr/testify/assert"
)

func ptrStr(s string) *string                  { return &s }
func ptrTime(t time.Time) *time.Time           { return &t }
func ptrGender(g models.Gender) *models.Gender { return &g }

func TestUpdateUserAccountDTO_HasUpdates(t *testing.T) {
	tests := []struct {
		name     string
		dto      UpdateUserAccountDTO
		expected bool
	}{
		{"all nil", UpdateUserAccountDTO{}, false},
		{"email set", UpdateUserAccountDTO{Email: ptrStr("a@b.com")}, true},
		{"phone set", UpdateUserAccountDTO{Phone: ptrStr("+123")}, true},
		{"username set", UpdateUserAccountDTO{Username: ptrStr("user")}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.dto.HasUpdates())
		})
	}
}

func TestUpdateUserProfileDTO_HasUpdates(t *testing.T) {
	tests := []struct {
		name     string
		dto      UpdateUserProfileDTO
		expected bool
	}{
		{"all nil", UpdateUserProfileDTO{}, false},
		{"FirstName set", UpdateUserProfileDTO{FirstName: ptrStr("A")}, true},
		{"LastName set", UpdateUserProfileDTO{LastName: ptrStr("B")}, true},
		{"Bio set", UpdateUserProfileDTO{Bio: ptrStr("bio")}, true},
		{"BirthdayDate set", UpdateUserProfileDTO{BirthdayDate: ptrTime(time.Now())}, true},
		{"Gender set", UpdateUserProfileDTO{Gender: ptrGender(models.Male)}, true},
		{"NativeTown set", UpdateUserProfileDTO{NativeTown: ptrStr("Town")}, true},
		{"Town set", UpdateUserProfileDTO{Town: ptrStr("City")}, true},
		{"Institution set", UpdateUserProfileDTO{Institution: ptrStr("Uni")}, true},
		{"Group set", UpdateUserProfileDTO{Group: ptrStr("Group")}, true},
		{"Company set", UpdateUserProfileDTO{Company: ptrStr("Company")}, true},
		{"JobTitle set", UpdateUserProfileDTO{JobTitle: ptrStr("Dev")}, true},
		{"Interests set", UpdateUserProfileDTO{Interests: ptrStr("Music")}, true},
		{"FavMusic set", UpdateUserProfileDTO{FavMusic: ptrStr("Rock")}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.dto.HasUpdates())
		})
	}
}

func TestUserSettingsUpdate_IsEmpty(t *testing.T) {
	lang := models.LanguageRU
	theme := models.ThemeDark

	tests := []struct {
		name     string
		update   UserSettingsUpdate
		expected bool
	}{
		{"both nil", UserSettingsUpdate{}, true},
		{"only language", UserSettingsUpdate{Language: &lang}, false},
		{"only theme", UserSettingsUpdate{Theme: &theme}, false},
		{"both set", UserSettingsUpdate{Language: &lang, Theme: &theme}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.update.IsEmpty())
		})
	}
}
