package session

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestSessionService_Create_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockSessionRepo(ctrl)
	service := NewSessionService(mockRepo)

	ctx := context.Background()
	userID := int64(42)

	// Ожидаем вызов Save с любым контекстом и сессией
	mockRepo.EXPECT().
		Save(gomock.Any(), gomock.AssignableToTypeOf(models.Session{})).
		Return(nil).
		Times(1)

	session, err := service.Create(ctx, userID)

	assert.NoError(t, err)
	assert.NotNil(t, session)
	assert.Equal(t, userID, session.UserID)
	assert.NotEmpty(t, session.SessionID)
	assert.WithinDuration(t, time.Now(), session.CreatedAt, time.Second)
	assert.WithinDuration(t, time.Now().Add(sessionTTL), session.ExpiredAt, time.Second)
}

func TestSessionService_Create_InvalidUserID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockSessionRepo(ctrl)
	service := NewSessionService(mockRepo)

	ctx := context.Background()
	invalidUserIDs := []int64{0, -1, -100}

	for _, uid := range invalidUserIDs {
		// Ожидаем, что Save НЕ будет вызван (0 раз)
		mockRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Times(0)

		session, err := service.Create(ctx, uid)

		assert.Error(t, err)
		assert.Nil(t, session)
		assert.EqualError(t, err, "invalid user id")
	}
}

func TestSessionService_Create_RepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockSessionRepo(ctrl)
	service := NewSessionService(mockRepo)

	ctx := context.Background()
	userID := int64(42)
	expectedErr := errors.New("database connection lost")

	mockRepo.EXPECT().
		Save(gomock.Any(), gomock.AssignableToTypeOf(models.Session{})).
		Return(expectedErr).
		Times(1)

	session, err := service.Create(ctx, userID)

	assert.Error(t, err)
	assert.EqualError(t, err, expectedErr.Error())
	assert.Nil(t, session)
}

func TestSessionService_Get_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockSessionRepo(ctrl)
	service := NewSessionService(mockRepo)

	ctx := context.Background()
	sessionID := models.SessionID("test-session-id")
	expectedSession := &models.Session{
		SessionID: sessionID,
		UserID:    42,
		CreatedAt: time.Now(),
		ExpiredAt: time.Now().Add(time.Hour),
	}

	mockRepo.EXPECT().
		GetByID(gomock.Any(), sessionID).
		Return(expectedSession, nil).
		Times(1)

	session, err := service.Get(ctx, sessionID)

	assert.NoError(t, err)
	assert.Equal(t, expectedSession, session)
}

func TestSessionService_Get_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockSessionRepo(ctrl)
	service := NewSessionService(mockRepo)

	ctx := context.Background()
	sessionID := models.SessionID("non-existent")
	expectedErr := errors.New("session not found")

	mockRepo.EXPECT().
		GetByID(gomock.Any(), sessionID).
		Return(nil, expectedErr).
		Times(1)

	session, err := service.Get(ctx, sessionID)

	assert.Error(t, err)
	assert.EqualError(t, err, expectedErr.Error())
	assert.Nil(t, session)
}

func TestSessionService_Delete_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockSessionRepo(ctrl)
	service := NewSessionService(mockRepo)

	ctx := context.Background()
	sessionID := models.SessionID("test-session-id")

	mockRepo.EXPECT().
		Delete(gomock.Any(), sessionID).
		Return(nil).
		Times(1)

	err := service.Delete(ctx, sessionID)
	assert.NoError(t, err)
}

func TestSessionService_Delete_RepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockSessionRepo(ctrl)
	service := NewSessionService(mockRepo)

	ctx := context.Background()
	sessionID := models.SessionID("test-session-id")
	expectedErr := errors.New("delete failed")

	mockRepo.EXPECT().
		Delete(gomock.Any(), sessionID).
		Return(expectedErr).
		Times(1)

	err := service.Delete(ctx, sessionID)

	assert.Error(t, err)
	assert.EqualError(t, err, expectedErr.Error())
}

func TestSessionTTL(t *testing.T) {
	assert.Equal(t, 24*time.Hour, sessionTTL)
}
