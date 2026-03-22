package models

import (
	"testing"

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

func TestNewProfile(t *testing.T) {
	profile := NewProfile(nil)
	assert.NotEqual(t, uuid.Nil, profile.Uid)
	assert.True(t, profile.IsActive)
}
