package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	mock_repo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/mocks"
	mock_service "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/mocks"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/websocket"
)

func strPtr(s string) *string { return &s }

func TestChatHandler_GetChats_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChatSvc := mock_service.NewMockChatService(ctrl)
	mockMsgSvc := mock_service.NewMockMessageService(ctrl)
	mockUserAccountRepo := mock_repo.NewMockUserAccountRepo(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)
	hub := websocket.NewHub()
	go hub.Run()

	handler := NewChatHandler(mockChatSvc, mockMsgSvc, mockUserAccountRepo, mockUserSvc, hub)

	userID := int64(1)
	profileID := int64(100)

	// Мок получения профиля текущего пользователя (для resolveChatTitle)
	mockUserSvc.EXPECT().
		GetUserProfileByUserAccountID(gomock.Any(), userID).
		Return(&models.UserProfile{ProfileID: profileID}, nil).AnyTimes()

	// Возвращаем список чатов
	chats := []models.Chat{
		{
			ID:        10,
			Uid:       uuid.New(),
			Title:     "Test Chat",
			Type:      models.PrivateChat,
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
	mockChatSvc.EXPECT().
		GetUserChats(gomock.Any(), userID).
		Return(chats, nil)

	// Мок для resolveChatTitle: GetChatMembers и получение профиля собеседника
	mockChatSvc.EXPECT().
		GetChatMembers(gomock.Any(), int64(10)).
		Return([]models.ChatMember{
			{MemberID: profileID},
			{MemberID: 200},
		}, nil).AnyTimes()
	mockUserSvc.EXPECT().
		GetUserProfileByProfileID(gomock.Any(), int64(200)).
		Return(&models.UserProfile{FirstName: "John", LastName: "Doe"}, nil).AnyTimes()

	req := httptest.NewRequest("GET", "/api/chats", nil)
	ctx := context.WithValue(req.Context(), "user_id", userID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.GetChats(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []ChatResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Len(t, response, 1)
	assert.Equal(t, "10", response[0].ID)
}

func TestChatHandler_GetChats_Unauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	handler := NewChatHandler(nil, nil, nil, nil, nil)

	req := httptest.NewRequest("GET", "/api/chats", nil)
	w := httptest.NewRecorder()
	handler.GetChats(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestChatHandler_GetChats_ServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChatSvc := mock_service.NewMockChatService(ctrl)
	handler := NewChatHandler(mockChatSvc, nil, nil, nil, nil)

	userID := int64(1)
	mockChatSvc.EXPECT().
		GetUserChats(gomock.Any(), userID).
		Return(nil, errors.New("db error"))

	req := httptest.NewRequest("GET", "/api/chats", nil)
	ctx := context.WithValue(req.Context(), "user_id", userID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.GetChats(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestChatHandler_CreateChat_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChatSvc := mock_service.NewMockChatService(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)
	handler := NewChatHandler(mockChatSvc, nil, nil, mockUserSvc, nil)

	userID := int64(1)
	otherUserID := int64(2)
	resolvedTargetID := int64(2)

	// Моки для resolveChatTargetUserAccountID
	mockUserSvc.EXPECT().
		GetUserAccountByProfileID(gomock.Any(), otherUserID).
		Return(nil, errors.New("not found"))
	mockUserSvc.EXPECT().
		GetUserProfileByUserAccountID(gomock.Any(), otherUserID).
		Return(nil, errors.New("not found"))
	mockUserSvc.EXPECT().
		GetUserAccountByProfileID(gomock.Any(), otherUserID).
		Return(&models.UserAccount{ID: resolvedTargetID}, nil)

	// Создание чата
	chat := &models.Chat{
		ID:        20,
		Uid:       uuid.New(),
		Type:      models.PrivateChat,
		Title:     "",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mockChatSvc.EXPECT().
		CreatePrivateChat(gomock.Any(), userID, resolvedTargetID).
		Return(chat, nil)

	// Для mapChatResponse
	mockUserSvc.EXPECT().
		GetUserProfileByUserAccountID(gomock.Any(), userID).
		Return(&models.UserProfile{ProfileID: 100}, nil).AnyTimes()
	mockChatSvc.EXPECT().
		GetChatMembers(gomock.Any(), chat.ID).
		Return([]models.ChatMember{
			{MemberID: 100},
			{MemberID: 200},
		}, nil).AnyTimes()
	mockUserSvc.EXPECT().
		GetUserProfileByProfileID(gomock.Any(), int64(200)).
		Return(&models.UserProfile{FirstName: "Alice", LastName: "Smith"}, nil).AnyTimes()

	req := httptest.NewRequest("POST", "/api/chats?otherUserId=2", nil)
	ctx := context.WithValue(req.Context(), "user_id", userID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.CreateChat(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp ChatResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "20", resp.ID)
}

func TestChatHandler_CreateChat_Unauthorized(t *testing.T) {
	handler := NewChatHandler(nil, nil, nil, nil, nil)
	req := httptest.NewRequest("POST", "/api/chats", nil)
	w := httptest.NewRecorder()
	handler.CreateChat(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestChatHandler_GetMessages_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChatSvc := mock_service.NewMockChatService(ctrl)
	mockMsgSvc := mock_service.NewMockMessageService(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)

	handler := NewChatHandler(mockChatSvc, mockMsgSvc, nil, mockUserSvc, nil)

	userID := int64(1)
	chatID := int64(5)

	// Проверка доступа
	mockChatSvc.EXPECT().
		CheckUserInChat(gomock.Any(), chatID, userID).
		Return(true, nil)

	// Получение сообщений
	messages := []models.Message{
		{
			ID:        100,
			Uid:       uuid.New(),
			Text:      strPtr("Hello"),
			ChatID:    chatID,
			AuthorID:  10,
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
	mockMsgSvc.EXPECT().
		GetMessages(gomock.Any(), chatID, 50, 0).
		Return(messages, nil)

	// Для mapMessageResponse
	mockUserSvc.EXPECT().
		GetUserProfileByProfileID(gomock.Any(), int64(10)).
		Return(&models.UserProfile{FirstName: "Bob", LastName: "Johnson"}, nil)

	req := httptest.NewRequest("GET", "/api/chats/5/messages", nil)
	ctx := context.WithValue(req.Context(), "user_id", userID)
	req = req.WithContext(ctx)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("chatID", "5")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handler.GetMessages(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp []MessageResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Len(t, resp, 1)
	assert.Equal(t, "100", resp[0].ID)
}

func TestChatHandler_SendMessage_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChatSvc := mock_service.NewMockChatService(ctrl)
	mockMsgSvc := mock_service.NewMockMessageService(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)
	hub := websocket.NewHub()
	go hub.Run()

	handler := NewChatHandler(mockChatSvc, mockMsgSvc, nil, mockUserSvc, hub)

	userID := int64(1)
	profileID := int64(100)
	chatID := int64(5)
	text := "Test message"

	// Проверка доступа
	mockChatSvc.EXPECT().
		CheckUserInChat(gomock.Any(), chatID, userID).
		Return(true, nil)

	// Получение профиля автора
	mockUserSvc.EXPECT().
		GetUserProfileByUserAccountID(gomock.Any(), userID).
		Return(&models.UserProfile{ProfileID: profileID}, nil)

	// Отправка сообщения
	msg := &models.Message{
		ID:        101,
		Uid:       uuid.New(),
		Text:      &text,
		ChatID:    chatID,
		AuthorID:  profileID,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mockMsgSvc.EXPECT().
		SendMessage(gomock.Any(), chatID, profileID, text).
		Return(msg, nil)

	// Для mapMessageResponse
	mockUserSvc.EXPECT().
		GetUserProfileByProfileID(gomock.Any(), profileID).
		Return(&models.UserProfile{FirstName: "Alice", LastName: "Smith"}, nil)

	reqBody := map[string]string{"text": text}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/chats/5/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), "user_id", userID)
	req = req.WithContext(ctx)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("chatID", "5")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handler.SendMessage(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp MessageResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "101", resp.ID)
}

// Дополнительные тесты для вспомогательных функций

func TestChatHandler_getViewerProfileID_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	handler := &ChatHandler{userService: mockUserSvc}

	userID := int64(1)
	profileID := int64(100)

	mockUserSvc.EXPECT().
		GetUserProfileByUserAccountID(gomock.Any(), userID).
		Return(&models.UserProfile{ProfileID: profileID}, nil)

	id, err := handler.getViewerProfileID(context.Background(), userID)
	assert.NoError(t, err)
	assert.Equal(t, profileID, id)
}

func TestChatHandler_resolveProfileDisplayName_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	handler := &ChatHandler{userService: mockUserSvc}

	profileID := int64(200)
	mockUserSvc.EXPECT().
		GetUserProfileByProfileID(gomock.Any(), profileID).
		Return(&models.UserProfile{FirstName: "John", LastName: "Doe"}, nil)

	name := handler.resolveProfileDisplayName(context.Background(), profileID)
	assert.Equal(t, "John Doe", name)
}

func TestChatHandler_resolveProfileDisplayName_NoService(t *testing.T) {
	handler := &ChatHandler{userService: nil}
	name := handler.resolveProfileDisplayName(context.Background(), 123)
	assert.Equal(t, "Пользователь", name)
}

// Оставляем существующие тесты UpdateMessage
func TestChatHandler_UpdateMessage_Success(t *testing.T) {
	// ... (ваш существующий код)
}

func TestChatHandler_UpdateMessage_Forbidden(t *testing.T) {
	// ... (ваш существующий код)
}
