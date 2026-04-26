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
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/mocks"
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

func TestChatHandler_filterChatsForList(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChatSvc := mock_service.NewMockChatService(ctrl)
	mockMsgSvc := mock_service.NewMockMessageService(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)

	handler := &ChatHandler{
		chatService:    mockChatSvc,
		messageService: mockMsgSvc,
		userService:    mockUserSvc,
	}

	viewerUserID := int64(1)
	viewerProfileID := int64(10)
	chats := []models.Chat{
		{ID: 1},
		{ID: 2},
		{ID: 3},
	}

	mockMsgSvc.EXPECT().GetMessages(gomock.Any(), int64(1), 1, 0).Return([]models.Message{{ID: 1}}, nil)
	mockMsgSvc.EXPECT().GetMessages(gomock.Any(), int64(2), 1, 0).Return(nil, nil)
	mockUserSvc.EXPECT().GetUserProfileByUserAccountID(gomock.Any(), viewerUserID).Return(&models.UserProfile{ProfileID: viewerProfileID}, nil)
	mockChatSvc.EXPECT().GetChatMembers(gomock.Any(), int64(2)).Return([]models.ChatMember{{MemberID: viewerProfileID}, {MemberID: 20}}, nil)
	mockUserSvc.EXPECT().GetUserAccountByProfileID(gomock.Any(), int64(20)).Return(&models.UserAccount{Username: "sergeyshulginenko"}, nil)
	mockMsgSvc.EXPECT().GetMessages(gomock.Any(), int64(3), 1, 0).Return(nil, nil)
	mockUserSvc.EXPECT().GetUserProfileByUserAccountID(gomock.Any(), viewerUserID).Return(nil, errors.New("profile error"))

	filtered := handler.filterChatsForList(context.Background(), chats, viewerUserID)
	assert.Len(t, filtered, 2)
	assert.Equal(t, int64(1), filtered[0].ID)
	assert.Equal(t, int64(2), filtered[1].ID)
}

func TestChatHandler_mapMessageResponse_WithPointers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	handler := &ChatHandler{userService: mockUserSvc}

	parentID := int64(7)
	stickerID := int64(9)
	text := "hello"
	msg := models.Message{
		ID:              1,
		Uid:             uuid.New(),
		Text:            &text,
		AuthorID:        5,
		ChatID:          3,
		ParentMessageID: &parentID,
		StickerID:       &stickerID,
		IsActive:        true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	mockUserSvc.EXPECT().GetUserProfileByProfileID(gomock.Any(), int64(5)).Return(&models.UserProfile{FirstName: "A", LastName: "B"}, nil)

	resp := handler.mapMessageResponse(context.Background(), msg)
	if assert.NotNil(t, resp.ParentMessageID) {
		assert.Equal(t, "7", *resp.ParentMessageID)
	}
	if assert.NotNil(t, resp.StickerID) {
		assert.Equal(t, "9", *resp.StickerID)
	}
	assert.Equal(t, "A B", resp.AuthorName)
}

func TestChatHandler_resolveChatTitle_Fallbacks(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChatSvc := mock_service.NewMockChatService(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)
	handler := &ChatHandler{
		chatService: mockChatSvc,
		userService: mockUserSvc,
	}

	chatModel := models.Chat{ID: 1, Title: "Fallback", Type: models.PrivateChat}

	mockUserSvc.EXPECT().GetUserProfileByUserAccountID(gomock.Any(), int64(1)).Return(nil, errors.New("profile error"))
	assert.Equal(t, "Fallback", handler.resolveChatTitle(context.Background(), chatModel, 1))

	mockUserSvc.EXPECT().GetUserProfileByUserAccountID(gomock.Any(), int64(1)).Return(&models.UserProfile{ProfileID: 10}, nil)
	mockChatSvc.EXPECT().GetChatMembers(gomock.Any(), int64(1)).Return(nil, errors.New("members error"))
	assert.Equal(t, "Fallback", handler.resolveChatTitle(context.Background(), chatModel, 1))
}

func TestChatHandler_resolveChatTargetUserAccountID_Branches(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	handler := &ChatHandler{userService: mockUserSvc}

	mockUserSvc.EXPECT().GetUserAccountByProfileID(gomock.Any(), int64(2)).Return(&models.UserAccount{ID: 22}, nil)
	id, err := handler.resolveChatTargetUserAccountID(context.Background(), 2)
	assert.NoError(t, err)
	assert.Equal(t, int64(22), id)

	mockUserSvc.EXPECT().GetUserAccountByProfileID(gomock.Any(), int64(3)).Return(nil, errors.New("not profile"))
	mockUserSvc.EXPECT().GetUserProfileByUserAccountID(gomock.Any(), int64(3)).Return(&models.UserProfile{ID: 3}, nil)
	id, err = handler.resolveChatTargetUserAccountID(context.Background(), 3)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), id)

	mockUserSvc.EXPECT().GetUserAccountByProfileID(gomock.Any(), int64(4)).Return(nil, errors.New("not profile"))
	mockUserSvc.EXPECT().GetUserProfileByUserAccountID(gomock.Any(), int64(4)).Return(nil, errors.New("not account"))
	mockUserSvc.EXPECT().GetUserAccountByProfileID(gomock.Any(), int64(4)).Return(nil, errors.New("still missing"))
	_, err = handler.resolveChatTargetUserAccountID(context.Background(), 4)
	assert.Error(t, err)
}

func TestEnsureGuaranteedDialog_UserAccountRepoNil(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	handler := &ChatHandler{
		userAccountRepo: nil,
		chatService:     nil,
	}

	// Просто вызываем, не должно быть паники
	handler.ensureGuaranteedDialog(context.Background(), 123)
	// Проверяем, что моки не вызывались (если бы были)
}

func TestEnsureGuaranteedDialog_GetCurrentUserError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mocks.NewMockUserAccountRepo(ctrl)
	handler := &ChatHandler{
		userAccountRepo: mockUserRepo,
		chatService:     nil,
	}

	ctx := context.Background()
	userID := int64(123)

	mockUserRepo.EXPECT().
		Get(gomock.Any(), userID).
		Return(nil, errors.New("db error"))

	handler.ensureGuaranteedDialog(ctx, userID)
	// Никаких дальнейших вызовов не должно быть
}

func TestEnsureGuaranteedDialog_FindsSergey(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChatSvc := mock_service.NewMockChatService(ctrl)
	mockMsgSvc := mock_service.NewMockMessageService(ctrl)
	mockUserAccountRepo := mock_repo.NewMockUserAccountRepo(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)
	hub := websocket.NewHub()
	go hub.Run()

	handler := NewChatHandler(mockChatSvc, mockMsgSvc, mockUserAccountRepo, mockUserSvc, hub)

	ctx := context.Background()
	userID := int64(123)
	currentUser := &models.UserAccount{ID: userID, Username: "testuser"}

	sergeyUser := &models.UserAccount{ID: 456, Username: "sergeyshulginenko"}

	mockUserAccountRepo.EXPECT().Get(ctx, userID).Return(currentUser, nil)
	mockUserAccountRepo.EXPECT().GetByUsername(ctx, "sergeyshulginenko").Return(sergeyUser, nil)

	mockChatSvc.EXPECT().
		CreatePrivateChat(ctx, currentUser.ID, sergeyUser.ID).
		Return(nil, nil)

	handler.ensureGuaranteedDialog(ctx, userID)
}

func TestEnsureGuaranteedDialog_FindsSecondTarget(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChatSvc := mock_service.NewMockChatService(ctrl)
	mockMsgSvc := mock_service.NewMockMessageService(ctrl)
	mockUserAccountRepo := mock_repo.NewMockUserAccountRepo(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)
	hub := websocket.NewHub()
	go hub.Run()

	handler := NewChatHandler(mockChatSvc, mockMsgSvc, mockUserAccountRepo, mockUserSvc, hub)

	ctx := context.Background()
	userID := int64(123)
	currentUser := &models.UserAccount{ID: userID, Username: "testuser"}

	targetUser := &models.UserAccount{ID: 789, Username: "ffffff"}

	mockUserAccountRepo.EXPECT().Get(ctx, userID).Return(currentUser, nil)

	// Первый целевой пользователь не найден
	mockUserAccountRepo.EXPECT().GetByUsername(ctx, "sergeyshulginenko").Return(nil, errors.New("not found"))
	// Второй найден
	mockUserAccountRepo.EXPECT().GetByUsername(ctx, "ffffff").Return(targetUser, nil)

	mockChatSvc.EXPECT().
		CreatePrivateChat(ctx, currentUser.ID, targetUser.ID).
		Return(nil, nil)

	handler.ensureGuaranteedDialog(ctx, userID)
}

func TestEnsureGuaranteedDialog_TargetUserIsSelf(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChatSvc := mock_service.NewMockChatService(ctrl)
	mockMsgSvc := mock_service.NewMockMessageService(ctrl)
	mockUserAccountRepo := mock_repo.NewMockUserAccountRepo(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)
	hub := websocket.NewHub()
	go hub.Run()

	handler := NewChatHandler(mockChatSvc, mockMsgSvc, mockUserAccountRepo, mockUserSvc, hub)

	ctx := context.Background()
	userID := int64(123)
	currentUser := &models.UserAccount{ID: userID, Username: "sergeyshulginenko"} // сам себе

	mockUserAccountRepo.EXPECT().Get(ctx, userID).Return(currentUser, nil)

	// Ожидаем, что после проверки целевого пользователя (который равен текущему),
	// продолжится к следующему или общему списку.
	// Здесь мы проверяем, что GetByUsername вызовется, но CreatePrivateChat не будет,
	// а затем переход к общему списку.
	mockUserAccountRepo.EXPECT().GetByUsername(ctx, "sergeyshulginenko").Return(currentUser, nil)
	mockUserAccountRepo.EXPECT().GetByUsername(ctx, "ffffff").Return(nil, errors.New("not found"))

	// Список пользователей
	otherUser := &models.UserAccount{ID: 999, Username: "other"}
	mockUserAccountRepo.EXPECT().List(ctx, 0, 20).Return([]models.UserAccount{*otherUser}, nil)

	mockChatSvc.EXPECT().
		CreatePrivateChat(ctx, currentUser.ID, otherUser.ID).
		Return(nil, nil)

	handler.ensureGuaranteedDialog(ctx, userID)
}

func TestEnsureGuaranteedDialog_ListUsersError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChatSvc := mock_service.NewMockChatService(ctrl)
	mockMsgSvc := mock_service.NewMockMessageService(ctrl)
	mockUserAccountRepo := mock_repo.NewMockUserAccountRepo(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)
	hub := websocket.NewHub()
	go hub.Run()

	handler := NewChatHandler(mockChatSvc, mockMsgSvc, mockUserAccountRepo, mockUserSvc, hub)

	ctx := context.Background()
	userID := int64(123)
	currentUser := &models.UserAccount{ID: userID, Username: "testuser"}

	mockUserAccountRepo.EXPECT().Get(ctx, userID).Return(currentUser, nil)
	// Оба целевых не найдены
	mockUserAccountRepo.EXPECT().GetByUsername(ctx, "sergeyshulginenko").Return(nil, errors.New("not found"))
	mockUserAccountRepo.EXPECT().GetByUsername(ctx, "ffffff").Return(nil, errors.New("not found"))

	mockUserAccountRepo.EXPECT().List(ctx, 0, 20).Return(nil, errors.New("db error"))

	handler.ensureGuaranteedDialog(ctx, userID)
	// Никаких вызовов CreatePrivateChat не должно быть
}

func TestEnsureGuaranteedDialog_FirstFromList(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChatSvc := mock_service.NewMockChatService(ctrl)
	mockMsgSvc := mock_service.NewMockMessageService(ctrl)
	mockUserAccountRepo := mock_repo.NewMockUserAccountRepo(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)
	hub := websocket.NewHub()
	go hub.Run()

	handler := NewChatHandler(mockChatSvc, mockMsgSvc, mockUserAccountRepo, mockUserSvc, hub)

	ctx := context.Background()
	userID := int64(123)
	currentUser := &models.UserAccount{ID: userID, Username: "testuser"}

	mockUserAccountRepo.EXPECT().Get(ctx, userID).Return(currentUser, nil)

	// Целевые пользователи не найдены
	mockUserAccountRepo.EXPECT().GetByUsername(ctx, "sergeyshulginenko").Return(nil, errors.New("not found"))
	mockUserAccountRepo.EXPECT().GetByUsername(ctx, "ffffff").Return(nil, errors.New("not found"))

	// Список пользователей: первый — сам текущий, второй — другой
	users := []models.UserAccount{
		{ID: userID, Username: "testuser"},
		{ID: 999, Username: "other"},
	}
	mockUserAccountRepo.EXPECT().List(ctx, 0, 20).Return(users, nil)

	mockChatSvc.EXPECT().
		CreatePrivateChat(ctx, currentUser.ID, int64(999)).
		Return(nil, nil)

	handler.ensureGuaranteedDialog(ctx, userID)
}

func TestEnsureGuaranteedDialog_NoOtherUsers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChatSvc := mock_service.NewMockChatService(ctrl)
	mockMsgSvc := mock_service.NewMockMessageService(ctrl)
	mockUserAccountRepo := mock_repo.NewMockUserAccountRepo(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)
	hub := websocket.NewHub()
	go hub.Run()

	handler := NewChatHandler(mockChatSvc, mockMsgSvc, mockUserAccountRepo, mockUserSvc, hub)

	ctx := context.Background()
	userID := int64(123)
	currentUser := &models.UserAccount{ID: userID, Username: "testuser"}

	mockUserAccountRepo.EXPECT().Get(ctx, userID).Return(currentUser, nil)

	mockUserAccountRepo.EXPECT().GetByUsername(ctx, "sergeyshulginenko").Return(nil, errors.New("not found"))
	mockUserAccountRepo.EXPECT().GetByUsername(ctx, "ffffff").Return(nil, errors.New("not found"))

	// Список содержит только текущего пользователя
	users := []models.UserAccount{{ID: userID, Username: "testuser"}}
	mockUserAccountRepo.EXPECT().List(ctx, 0, 20).Return(users, nil)

	handler.ensureGuaranteedDialog(ctx, userID)
	// Цикл завершится, не вызвав CreatePrivateChat
}

func TestChatHasMessages_NilMessageService(t *testing.T) {
	handler := &ChatHandler{messageService: nil}
	result := handler.chatHasMessages(context.Background(), 123)
	assert.False(t, result)
}

func TestChatHasMessages_ErrorGettingMessages(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMsgSvc := mock_service.NewMockMessageService(ctrl)
	handler := &ChatHandler{messageService: mockMsgSvc}

	chatID := int64(123)
	mockMsgSvc.EXPECT().
		GetMessages(gomock.Any(), chatID, 1, 0).
		Return(nil, errors.New("db error"))

	result := handler.chatHasMessages(context.Background(), chatID)
	assert.False(t, result)
}

func TestChatHasMessages_EmptyList(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMsgSvc := mock_service.NewMockMessageService(ctrl)
	handler := &ChatHandler{messageService: mockMsgSvc}

	chatID := int64(123)
	mockMsgSvc.EXPECT().
		GetMessages(gomock.Any(), chatID, 1, 0).
		Return([]models.Message{}, nil)

	result := handler.chatHasMessages(context.Background(), chatID)
	assert.False(t, result)
}

func TestChatHasMessages_HasMessages(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMsgSvc := mock_service.NewMockMessageService(ctrl)
	handler := &ChatHandler{messageService: mockMsgSvc}

	chatID := int64(123)
	messages := []models.Message{{ID: 1}}
	mockMsgSvc.EXPECT().
		GetMessages(gomock.Any(), chatID, 1, 0).
		Return(messages, nil)

	result := handler.chatHasMessages(context.Background(), chatID)
	assert.True(t, result)
}

func TestIsSergeySupportChat_NilServices(t *testing.T) {
	handler := &ChatHandler{chatService: nil, userService: nil}
	result := handler.isSergeySupportChat(context.Background(), models.Chat{ID: 1}, 123)
	assert.False(t, result)
}

func TestIsSergeySupportChat_GetViewerProfileIDError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChatSvc := mock_service.NewMockChatService(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)

	handler := &ChatHandler{
		chatService: mockChatSvc,
		userService: mockUserSvc,
	}

	ctx := context.Background()
	viewerUserID := int64(123)
	chat := models.Chat{ID: 456}

	// Исправлено: GetUserProfileByUserAccountID
	mockUserSvc.EXPECT().
		GetUserProfileByUserAccountID(gomock.Any(), viewerUserID).
		Return(nil, errors.New("not found"))

	result := handler.isSergeySupportChat(ctx, chat, viewerUserID)
	assert.False(t, result)
}

func TestIsSergeySupportChat_GetChatMembersError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChatSvc := mock_service.NewMockChatService(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)

	handler := &ChatHandler{
		chatService: mockChatSvc,
		userService: mockUserSvc,
	}

	ctx := context.Background()
	viewerUserID := int64(123)
	chat := models.Chat{ID: 456}
	viewerProfile := &models.UserProfile{ProfileID: 789} // или *models.Profile, зависит от возвращаемого типа

	mockUserSvc.EXPECT().
		GetUserProfileByUserAccountID(gomock.Any(), viewerUserID).
		Return(viewerProfile, nil)

	mockChatSvc.EXPECT().
		GetChatMembers(gomock.Any(), chat.ID).
		Return(nil, errors.New("db error"))

	result := handler.isSergeySupportChat(ctx, chat, viewerUserID)
	assert.False(t, result)
}

func TestIsSergeySupportChat_NoOtherMembers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChatSvc := mock_service.NewMockChatService(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)

	handler := &ChatHandler{
		chatService: mockChatSvc,
		userService: mockUserSvc,
	}

	ctx := context.Background()
	viewerUserID := int64(123)
	chat := models.Chat{ID: 456}
	viewerProfile := &models.UserProfile{ProfileID: 789}

	mockUserSvc.EXPECT().
		GetUserProfileByUserAccountID(gomock.Any(), viewerUserID).
		Return(viewerProfile, nil)

	members := []models.ChatMember{
		{MemberID: viewerProfile.ProfileID},
	}
	mockChatSvc.EXPECT().
		GetChatMembers(gomock.Any(), chat.ID).
		Return(members, nil)

	result := handler.isSergeySupportChat(ctx, chat, viewerUserID)
	assert.False(t, result)
}

func TestIsSergeySupportChat_OtherMemberAccountError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChatSvc := mock_service.NewMockChatService(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)

	handler := &ChatHandler{
		chatService: mockChatSvc,
		userService: mockUserSvc,
	}

	ctx := context.Background()
	viewerUserID := int64(123)
	chat := models.Chat{ID: 456}
	viewerProfile := &models.UserProfile{ProfileID: 789}
	otherProfileID := int64(999)

	mockUserSvc.EXPECT().
		GetUserProfileByUserAccountID(gomock.Any(), viewerUserID).
		Return(viewerProfile, nil)

	members := []models.ChatMember{
		{MemberID: viewerProfile.ProfileID},
		{MemberID: otherProfileID},
	}
	mockChatSvc.EXPECT().
		GetChatMembers(gomock.Any(), chat.ID).
		Return(members, nil)

	mockUserSvc.EXPECT().
		GetUserAccountByProfileID(gomock.Any(), otherProfileID).
		Return(nil, errors.New("not found"))

	result := handler.isSergeySupportChat(ctx, chat, viewerUserID)
	assert.False(t, result)
}

func TestIsSergeySupportChat_FoundSergey(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChatSvc := mock_service.NewMockChatService(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)

	handler := &ChatHandler{
		chatService: mockChatSvc,
		userService: mockUserSvc,
	}

	ctx := context.Background()
	viewerUserID := int64(123)
	chat := models.Chat{ID: 456}
	viewerProfile := &models.UserProfile{ProfileID: 789}
	otherProfileID := int64(999)

	mockUserSvc.EXPECT().
		GetUserProfileByUserAccountID(gomock.Any(), viewerUserID).
		Return(viewerProfile, nil)

	members := []models.ChatMember{
		{MemberID: viewerProfile.ProfileID},
		{MemberID: otherProfileID},
	}
	mockChatSvc.EXPECT().
		GetChatMembers(gomock.Any(), chat.ID).
		Return(members, nil)

	sergeyAccount := &models.UserAccount{Username: "sergeyshulginenko"}
	mockUserSvc.EXPECT().
		GetUserAccountByProfileID(gomock.Any(), otherProfileID).
		Return(sergeyAccount, nil)

	result := handler.isSergeySupportChat(ctx, chat, viewerUserID)
	assert.True(t, result)
}

func TestIsSergeySupportChat_NotSergey(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChatSvc := mock_service.NewMockChatService(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)

	handler := &ChatHandler{
		chatService: mockChatSvc,
		userService: mockUserSvc,
	}

	ctx := context.Background()
	viewerUserID := int64(123)
	chat := models.Chat{ID: 456}
	viewerProfile := &models.UserProfile{ProfileID: 789}
	otherProfileID := int64(999)

	mockUserSvc.EXPECT().
		GetUserProfileByUserAccountID(gomock.Any(), viewerUserID).
		Return(viewerProfile, nil)

	members := []models.ChatMember{
		{MemberID: viewerProfile.ProfileID},
		{MemberID: otherProfileID},
	}
	mockChatSvc.EXPECT().
		GetChatMembers(gomock.Any(), chat.ID).
		Return(members, nil)

	otherAccount := &models.UserAccount{Username: "not_sergey"}
	mockUserSvc.EXPECT().
		GetUserAccountByProfileID(gomock.Any(), otherProfileID).
		Return(otherAccount, nil)

	result := handler.isSergeySupportChat(ctx, chat, viewerUserID)
	assert.False(t, result)
}

func TestUpdateMessage_MissingUserID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	handler := &ChatHandler{} // зависимости не нужны

	req := httptest.NewRequest(http.MethodPut, "/messages/123", nil)
	rec := httptest.NewRecorder()

	handler.UpdateMessage(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "не авторизован")
}

func TestUpdateMessage_GetViewerProfileIDError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	handler := &ChatHandler{userService: mockUserSvc}

	ctx := context.WithValue(context.Background(), "user_id", int64(123))
	req := httptest.NewRequest(http.MethodPut, "/messages/456", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	mockUserSvc.EXPECT().
		GetUserProfileByUserAccountID(gomock.Any(), int64(123)).
		Return(nil, errors.New("profile not found"))

	handler.UpdateMessage(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "ошибка профиля пользователя")
}

func TestUpdateMessage_InvalidJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	handler := &ChatHandler{userService: mockUserSvc}

	ctx := context.WithValue(context.Background(), "user_id", int64(123))
	body := bytes.NewBufferString(`{"text":`)
	req := httptest.NewRequest(http.MethodPut, "/messages/123", body).WithContext(ctx)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("messageID", "123")
	ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	viewerProfile := &models.UserProfile{ProfileID: 789}
	mockUserSvc.EXPECT().
		GetUserProfileByUserAccountID(gomock.Any(), int64(123)).
		Return(viewerProfile, nil)

	handler.UpdateMessage(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "неверный формат запроса")
}

func TestUpdateMessage_EmptyText(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	handler := &ChatHandler{userService: mockUserSvc}

	ctx := context.WithValue(context.Background(), "user_id", int64(123))
	body := bytes.NewBufferString(`{"text": ""}`)
	req := httptest.NewRequest(http.MethodPut, "/messages/123", body).WithContext(ctx)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("messageID", "123")
	ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	viewerProfile := &models.UserProfile{ProfileID: 789}
	mockUserSvc.EXPECT().
		GetUserProfileByUserAccountID(gomock.Any(), int64(123)).
		Return(viewerProfile, nil)

	handler.UpdateMessage(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "текст сообщения не может быть пустым")
}

func TestUpdateMessage_ForbiddenError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockMsgSvc := mock_service.NewMockMessageService(ctrl)
	handler := &ChatHandler{
		userService:    mockUserSvc,
		messageService: mockMsgSvc,
	}

	ctx := context.WithValue(context.Background(), "user_id", int64(123))
	body := bytes.NewBufferString(`{"text": "new text"}`)
	req := httptest.NewRequest(http.MethodPut, "/messages/456", body).WithContext(ctx)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("messageID", "456")
	ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	viewerProfile := &models.UserProfile{ProfileID: 789}
	mockUserSvc.EXPECT().
		GetUserProfileByUserAccountID(gomock.Any(), int64(123)).
		Return(viewerProfile, nil)

	mockMsgSvc.EXPECT().
		UpdateMessage(gomock.Any(), int64(456), viewerProfile.ProfileID, "new text").
		Return(nil, errors.New("forbidden: you can only edit your own messages"))

	handler.UpdateMessage(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), "доступ запрещён")
}

func TestUpdateMessage_InternalError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockMsgSvc := mock_service.NewMockMessageService(ctrl)
	handler := &ChatHandler{
		userService:    mockUserSvc,
		messageService: mockMsgSvc,
	}

	ctx := context.WithValue(context.Background(), "user_id", int64(123))
	body := bytes.NewBufferString(`{"text": "new text"}`)
	req := httptest.NewRequest(http.MethodPut, "/messages/456", body).WithContext(ctx)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("messageID", "456")
	ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	viewerProfile := &models.UserProfile{ProfileID: 789}
	mockUserSvc.EXPECT().
		GetUserProfileByUserAccountID(gomock.Any(), int64(123)).
		Return(viewerProfile, nil)

	mockMsgSvc.EXPECT().
		UpdateMessage(gomock.Any(), int64(456), viewerProfile.ProfileID, "new text").
		Return(nil, errors.New("database error"))

	handler.UpdateMessage(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "ошибка обновления сообщения")
}
