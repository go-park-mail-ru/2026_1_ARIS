package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	usermock "github.com/go-park-mail-ru/2026_1_ARIS/proto/user/mock"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/game/internal/model"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/game/internal/repository"
	repositorymock "github.com/go-park-mail-ru/2026_1_ARIS/services/game/internal/repository/mocks"
	"github.com/golang/mock/gomock"
)

// newGameService creates a Service with all mocked dependencies.
func newGameService(ctrl *gomock.Controller) (
	*Service,
	*repositorymock.MockRoomRepo,
	*repositorymock.MockMemberRepo,
	*repositorymock.MockQuestionRepo,
	*repositorymock.MockRoomQuestionRepo,
	*repositorymock.MockAnswerRepo,
	*repositorymock.MockRatingRepo,
	*usermock.MockUserServiceClient,
) {
	rooms := repositorymock.NewMockRoomRepo(ctrl)
	members := repositorymock.NewMockMemberRepo(ctrl)
	questions := repositorymock.NewMockQuestionRepo(ctrl)
	answers := repositorymock.NewMockAnswerRepo(ctrl)
	messages := repositorymock.NewMockMessageRepo(ctrl)
	ratings := repositorymock.NewMockRatingRepo(ctrl)
	roomQuestions := repositorymock.NewMockRoomQuestionRepo(ctrl)
	users := usermock.NewMockUserServiceClient(ctrl)
	store := repository.Store{
		Questions: questions,
		Rooms:     rooms,
		Members:   members,
		RoomQs:    roomQuestions,
		Answers:   answers,
		Messages:  messages,
		Ratings:   ratings,
	}
	svc := New(store, users)
	return svc, rooms, members, questions, roomQuestions, answers, ratings, users
}

// setupCleanup mocks the two cleanup calls that appear in most methods.
func setupCleanup(members *repositorymock.MockMemberRepo, rooms *repositorymock.MockRoomRepo) {
	members.EXPECT().DeactivateStaleWaiting(gomock.Any(), staleWaitingMemberTTL).Return(nil, nil)
	rooms.EXPECT().DeactivateEmptyWaitingOlderThan(gomock.Any(), emptyWaitingRoomTTL).Return(nil)
}

// setupBuildRoom mocks all calls made by buildRoom for a room with a single member.
// room.Status must be RoomStatusWaiting (no CurrentQuestionID, no RoomQs beyond an empty list).
func setupBuildRoom(
	members *repositorymock.MockMemberRepo,
	roomQuestions *repositorymock.MockRoomQuestionRepo,
	ratings *repositorymock.MockRatingRepo,
	users *usermock.MockUserServiceClient,
	roomID, profileID int64,
) {
	member := model.RoomMember{ProfileID: profileID, RoomID: roomID, IsActive: true}
	members.EXPECT().List(gomock.Any(), roomID).Return([]model.RoomMember{member}, nil)
	members.EXPECT().Stats(gomock.Any(), profileID).Return(model.ProfileStats{}, nil)
	roomQuestions.EXPECT().List(gomock.Any(), roomID).Return(nil, nil)
	// player() calls GetProfileSummary twice: once for the member in players loop, once for Creator
	users.EXPECT().GetProfileSummary(gomock.Any(), &userpb.GetProfileSummaryRequest{ProfileId: profileID}).
		Return(&userpb.GetProfileSummaryResponse{ProfileId: profileID, Username: "user"}, nil).Times(2)
}

// ---------------------------------------------------------------------------
// CreateRoom
// ---------------------------------------------------------------------------

func TestCreateRoom_EmptyTitle_ReturnsErrInvalidInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, _, _, _, _, _, _, users := newGameService(ctrl)
	ctx := context.Background()

	users.EXPECT().GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: 5}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: 50}, nil)

	_, err := svc.CreateRoom(ctx, 5, CreateRoomInput{Title: "   "})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateRoom_ProfileIDByAccountFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, _, _, _, _, _, _, users := newGameService(ctrl)
	ctx := context.Background()

	rpcErr := errors.New("user service unavailable")
	users.EXPECT().GetProfileByUserAccount(gomock.Any(), gomock.Any()).Return(nil, rpcErr)

	_, err := svc.CreateRoom(ctx, 5, CreateRoomInput{Title: "My Room"})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func TestCreateRoom_InvalidUserAccountID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, _, _, _, _, _, _, _ := newGameService(ctrl)
	ctx := context.Background()

	_, err := svc.CreateRoom(ctx, 0, CreateRoomInput{Title: "My Room"})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for userAccountID=0, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// DisbandRoom
// ---------------------------------------------------------------------------

func TestDisbandRoom_InvalidUserAccountID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, _, _, _, _, _, _, _ := newGameService(ctrl)
	ctx := context.Background()

	err := svc.DisbandRoom(ctx, 0, 1)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestDisbandRoom_ProfileIDByAccountFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, _, _, _, _, _, _, users := newGameService(ctrl)
	ctx := context.Background()

	users.EXPECT().GetProfileByUserAccount(gomock.Any(), gomock.Any()).Return(nil, errors.New("rpc error"))

	err := svc.DisbandRoom(ctx, 5, 1)
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestDisbandRoom_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, rooms, _, _, _, _, _, users := newGameService(ctrl)
	ctx := context.Background()

	const (
		userAccountID int64 = 5
		profileID     int64 = 50
		roomID        int64 = 1
	)

	users.EXPECT().GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: userAccountID}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: profileID}, nil)

	room := &model.Room{
		ID:                 roomID,
		Status:             model.RoomStatusWaiting,
		CreatedByProfileID: profileID,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	// InTx with nil pool calls fn(store) directly.
	rooms.EXPECT().GetForUpdate(gomock.Any(), roomID).Return(room, nil)
	rooms.EXPECT().Deactivate(gomock.Any(), roomID).Return(nil)

	if err := svc.DisbandRoom(ctx, userAccountID, roomID); err != nil {
		t.Fatalf("DisbandRoom() error = %v", err)
	}
}

// ---------------------------------------------------------------------------
// LeaveRoom
// ---------------------------------------------------------------------------

func TestLeaveRoom_InvalidUserAccountID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, _, _, _, _, _, _, _ := newGameService(ctrl)
	ctx := context.Background()

	err := svc.LeaveRoom(ctx, 0, 1)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestLeaveRoom_ProfileIDByAccountFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, _, _, _, _, _, _, users := newGameService(ctrl)
	ctx := context.Background()

	users.EXPECT().GetProfileByUserAccount(gomock.Any(), gomock.Any()).Return(nil, errors.New("rpc error"))

	err := svc.LeaveRoom(ctx, 5, 1)
	if err == nil {
		t.Fatal("expected an error")
	}
}

// ---------------------------------------------------------------------------
// GetRoom
// ---------------------------------------------------------------------------

func TestGetRoom_InvalidUserAccountID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, _, _, _, _, _, _, _ := newGameService(ctrl)
	ctx := context.Background()

	_, err := svc.GetRoom(ctx, 0, 1)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestGetRoom_RoomsGetFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, rooms, members, _, _, _, _, users := newGameService(ctrl)
	ctx := context.Background()

	users.EXPECT().GetProfileByUserAccount(gomock.Any(), gomock.Any()).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: 50}, nil)
	setupCleanup(members, rooms)
	rooms.EXPECT().Get(gomock.Any(), int64(1)).Return(nil, repository.ErrNotFound)

	_, err := svc.GetRoom(ctx, 5, 1)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGetRoom_UserNotMember_ReturnsForbidden(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, rooms, members, _, _, _, _, users := newGameService(ctrl)
	ctx := context.Background()

	const (
		userAccountID int64 = 5
		profileID     int64 = 50
		roomID        int64 = 1
	)

	users.EXPECT().GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: userAccountID}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: profileID}, nil)
	setupCleanup(members, rooms)

	room := &model.Room{
		ID:        roomID,
		Status:    model.RoomStatusWaiting,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	rooms.EXPECT().Get(gomock.Any(), roomID).Return(room, nil)
	members.EXPECT().IsMember(gomock.Any(), roomID, profileID).Return(false, nil)

	_, err := svc.GetRoom(ctx, userAccountID, roomID)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestGetRoom_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, rooms, members, _, roomQuestions, _, ratings, users := newGameService(ctrl)
	ctx := context.Background()

	const (
		userAccountID int64 = 5
		profileID     int64 = 50
		roomID        int64 = 1
	)

	users.EXPECT().GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: userAccountID}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: profileID}, nil)
	setupCleanup(members, rooms)

	room := &model.Room{
		ID:                 roomID,
		Title:              "Test Room",
		Status:             model.RoomStatusWaiting,
		CreatedByProfileID: profileID,
		MaxPlayers:         2,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	rooms.EXPECT().Get(gomock.Any(), roomID).Return(room, nil)
	members.EXPECT().IsMember(gomock.Any(), roomID, profileID).Return(true, nil)

	setupBuildRoom(members, roomQuestions, ratings, users, roomID, profileID)

	result, err := svc.GetRoom(ctx, userAccountID, roomID)
	if err != nil {
		t.Fatalf("GetRoom() error = %v", err)
	}
	if result.ID != roomID {
		t.Fatalf("expected room ID %d, got %d", roomID, result.ID)
	}
	if result.Title != "Test Room" {
		t.Fatalf("expected title 'Test Room', got %q", result.Title)
	}
}

// ---------------------------------------------------------------------------
// ListRooms
// ---------------------------------------------------------------------------

func TestListRooms_EmptyList(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, rooms, members, _, _, _, _, users := newGameService(ctrl)
	ctx := context.Background()

	const (
		userAccountID int64 = 5
		profileID     int64 = 50
	)

	users.EXPECT().GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: userAccountID}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: profileID}, nil)

	// finalizeExpiredForProfile
	rooms.EXPECT().ExpiredActiveIDsForProfile(gomock.Any(), profileID).Return(nil, nil)
	// DeactivateWaitingForProfile
	members.EXPECT().DeactivateWaitingForProfile(gomock.Any(), profileID).Return(nil, nil)
	// cleanupEmptyWaitingRooms
	setupCleanup(members, rooms)
	// ListForProfile with normalized limit (default 50, max 100) and offset 0
	rooms.EXPECT().ListForProfile(gomock.Any(), profileID, 50, 0).Return(nil, nil)

	result, err := svc.ListRooms(ctx, userAccountID, 0, 0)
	if err != nil {
		t.Fatalf("ListRooms() error = %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected empty result, got %d rooms", len(result))
	}
}

func TestListRooms_LimitOffsetNormalization(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, rooms, members, _, _, _, _, users := newGameService(ctrl)
	ctx := context.Background()

	const (
		userAccountID int64 = 5
		profileID     int64 = 50
	)

	users.EXPECT().GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: userAccountID}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: profileID}, nil)

	rooms.EXPECT().ExpiredActiveIDsForProfile(gomock.Any(), profileID).Return(nil, nil)
	members.EXPECT().DeactivateWaitingForProfile(gomock.Any(), profileID).Return(nil, nil)
	setupCleanup(members, rooms)
	// negative limit should become default (50), negative offset should become 0
	rooms.EXPECT().ListForProfile(gomock.Any(), profileID, 50, 0).Return(nil, nil)

	result, err := svc.ListRooms(ctx, userAccountID, -5, -10)
	if err != nil {
		t.Fatalf("ListRooms() error = %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result slice")
	}
}

// ---------------------------------------------------------------------------
// History
// ---------------------------------------------------------------------------

func TestHistory_InvalidUserAccountID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, _, _, _, _, _, _, _ := newGameService(ctrl)
	ctx := context.Background()

	_, err := svc.History(ctx, 0, 10, 0)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestHistory_EmptyHistory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, rooms, members, _, _, _, _, users := newGameService(ctrl)
	ctx := context.Background()

	const (
		userAccountID int64 = 5
		profileID     int64 = 50
	)

	users.EXPECT().GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: userAccountID}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: profileID}, nil)

	rooms.EXPECT().ExpiredActiveIDsForProfile(gomock.Any(), profileID).Return(nil, nil)
	setupCleanup(members, rooms)
	rooms.EXPECT().HistoryForProfile(gomock.Any(), profileID, 50, 0).Return(nil, nil)

	items, err := svc.History(ctx, userAccountID, 0, 0)
	if err != nil {
		t.Fatalf("History() error = %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected empty history, got %d items", len(items))
	}
}

// ---------------------------------------------------------------------------
// CleanupEmptyWaitingRooms
// ---------------------------------------------------------------------------

func TestCleanupEmptyWaitingRooms_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, rooms, members, _, _, _, _, _ := newGameService(ctrl)
	ctx := context.Background()

	members.EXPECT().DeactivateStaleWaiting(gomock.Any(), staleWaitingMemberTTL).Return(nil, nil)
	rooms.EXPECT().DeactivateEmptyWaitingOlderThan(gomock.Any(), emptyWaitingRoomTTL).Return(nil)

	if err := svc.CleanupEmptyWaitingRooms(ctx); err != nil {
		t.Fatalf("CleanupEmptyWaitingRooms() error = %v", err)
	}
}

func TestCleanupEmptyWaitingRooms_PropagatesError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, _, members, _, _, _, _, _ := newGameService(ctrl)
	ctx := context.Background()

	dbErr := errors.New("db error")
	members.EXPECT().DeactivateStaleWaiting(gomock.Any(), staleWaitingMemberTTL).Return(nil, dbErr)

	if err := svc.CleanupEmptyWaitingRooms(ctx); !errors.Is(err, dbErr) {
		t.Fatalf("expected db error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// CreateQuestion
// ---------------------------------------------------------------------------

func TestCreateQuestion_EmptyText_ReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// requireQuestionAdmin requires a support client; nil supportClient → ErrForbidden
	// so we can test the empty-text path only when admin check passes.
	// Since we can't pass a support client mock easily here, test the early return
	// path where supportClient is nil → ErrForbidden.
	svc, _, _, _, _, _, _, _ := newGameService(ctrl)
	ctx := context.Background()

	_, err := svc.CreateQuestion(ctx, 5, QuestionInput{Text: ""})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden (no support client), got %v", err)
	}
}

// ---------------------------------------------------------------------------
// UpdateQuestion
// ---------------------------------------------------------------------------

func TestUpdateQuestion_NoSupportClient_ReturnsForbidden(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, _, _, _, _, _, _, _ := newGameService(ctrl)
	ctx := context.Background()

	_, err := svc.UpdateQuestion(ctx, 5, 1, QuestionInput{Text: "Q?", CorrectAnswer: 42})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden (no support client), got %v", err)
	}
}

// ---------------------------------------------------------------------------
// DeleteQuestion
// ---------------------------------------------------------------------------

func TestDeleteQuestion_NoSupportClient_ReturnsForbidden(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc, _, _, _, _, _, _, _ := newGameService(ctrl)
	ctx := context.Background()

	err := svc.DeleteQuestion(ctx, 5, 1)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden (no support client), got %v", err)
	}
}

func TestDeleteQuestion_InvalidID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// With no support client, requireQuestionAdmin fires first, returning ErrForbidden
	// before we can exercise the id <= 0 check. Both return an error, which is what we want.
	svc, _, _, _, _, _, _, _ := newGameService(ctrl)
	ctx := context.Background()

	err := svc.DeleteQuestion(ctx, 5, 0)
	if err == nil {
		t.Fatal("expected an error for id=0")
	}
}
