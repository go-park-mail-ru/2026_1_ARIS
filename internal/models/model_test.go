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
	user := NewUser(pass, &phone, &email)
	assert.NotEqual(t, uuid.Nil, user.ID)
	assert.Equal(t, "test@test.com", *user.Email)
	assert.Equal(t, "+1234567890", *user.Phone)
	assert.Equal(t, pass, user.PasswordHash)
	assert.False(t, user.CreatedAt.IsZero())
}

func TestNewProfile(t *testing.T) {
	username := "testuser"
	profile := NewProfile(username, nil, true)
	assert.NotEqual(t, uuid.Nil, profile.ID)
	assert.Equal(t, username, profile.Username)
	assert.True(t, profile.IsActive)
}
