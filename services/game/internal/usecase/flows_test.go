package usecase

import (
	"context"
	"testing"
	"time"

	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	usermock "github.com/go-park-mail-ru/2026_1_ARIS/proto/user/mock"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/game/internal/model"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/game/internal/repository"
	repositorymock "github.com/go-park-mail-ru/2026_1_ARIS/services/game/internal/repository/mocks"
	"github.com/golang/mock/gomock"
)

func TestServiceListQuestions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	questions := repositorymock.NewMockQuestionRepo(ctrl)
	service := New(repository.Store{Questions: questions}, nil)

	unit := "km"
	questions.EXPECT().
		List(gomock.Any(), model.DefaultGameType, true, 100, 0).
		Return([]model.Question{{ID: 1, Text: "Question", CorrectAnswer: 42, AnswerUnit: &unit, IsActive: true}}, nil)

	result, err := service.ListQuestions(ctx, " ", true, 0, -10)
	if err != nil {
		t.Fatalf("ListQuestions() error = %v", err)
	}
	if len(result) != 1 || result[0].ID != 1 || result[0].AnswerUnit == nil || *result[0].AnswerUnit != "km" {
		t.Fatalf("unexpected questions: %+v", result)
	}
}

func TestServiceLeaderboard(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	ratings := repositorymock.NewMockRatingRepo(ctrl)
	users := usermock.NewMockUserServiceClient(ctrl)
	service := New(repository.Store{Ratings: ratings}, users)

	ratings.EXPECT().EnsureSeason(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, season *model.RatingSeason) error {
		season.ID = 44
		return nil
	})
	ratings.EXPECT().Leaderboard(gomock.Any(), int64(44), 50, 0).Return([]model.LeaderboardEntry{
		{Rank: 1, ProfileID: 70, Rating: 1100, GamesPlayed: 10, Wins: 7, Draws: 1},
	}, nil)
	users.EXPECT().GetProfileSummary(gomock.Any(), &userpb.GetProfileSummaryRequest{ProfileId: 70}).Return(&userpb.GetProfileSummaryResponse{
		ProfileId: 70, UserAccountId: 7, FirstName: "Ann", LastName: "Winner", Username: "ann",
	}, nil)

	board, err := service.Leaderboard(ctx, "", 0, -1)
	if err != nil {
		t.Fatalf("Leaderboard() error = %v", err)
	}
	if board.GameType != model.DefaultGameType || board.Season.SeasonNumber == 0 || len(board.Entries) != 1 {
		t.Fatalf("unexpected leaderboard: %+v", board)
	}
	if board.Entries[0].Player.Name != "Ann Winner" || board.Entries[0].Rating != 1100 {
		t.Fatalf("unexpected leaderboard entry: %+v", board.Entries[0])
	}
}

func TestServiceStats(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	users := usermock.NewMockUserServiceClient(ctrl)
	rooms := repositorymock.NewMockRoomRepo(ctrl)
	members := repositorymock.NewMockMemberRepo(ctrl)
	service := New(repository.Store{Rooms: rooms, Members: members}, users)

	users.EXPECT().GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: 8}).Return(&userpb.GetProfileByUserAccountResponse{ProfileId: 80}, nil)
	rooms.EXPECT().ExpiredActiveIDsForProfile(gomock.Any(), int64(80)).Return(nil, nil)
	members.EXPECT().DeactivateStaleWaiting(gomock.Any(), staleWaitingMemberTTL).Return(nil, nil)
	rooms.EXPECT().DeactivateEmptyWaitingOlderThan(gomock.Any(), emptyWaitingRoomTTL).Return(nil)
	members.EXPECT().Stats(gomock.Any(), int64(80)).Return(model.ProfileStats{Played: 3, Won: 2, Lost: 1, Drawn: 0}, nil)

	stats, err := service.Stats(ctx, 8)
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}
	if stats.Played != 3 || stats.Won != 2 || stats.Lost != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

func TestServiceRoomMessages(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	users := usermock.NewMockUserServiceClient(ctrl)
	rooms := repositorymock.NewMockRoomRepo(ctrl)
	members := repositorymock.NewMockMemberRepo(ctrl)
	messages := repositorymock.NewMockMessageRepo(ctrl)
	service := New(repository.Store{Rooms: rooms, Members: members, Messages: messages}, users)

	createdAt := time.Date(2026, 5, 27, 10, 0, 0, 0, time.UTC)
	users.EXPECT().GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: 9}).Return(&userpb.GetProfileByUserAccountResponse{ProfileId: 90}, nil)
	rooms.EXPECT().Get(gomock.Any(), int64(5)).Return(&model.Room{ID: 5}, nil)
	members.EXPECT().IsMember(gomock.Any(), int64(5), int64(90)).Return(true, nil)
	messages.EXPECT().List(gomock.Any(), int64(5), 100, 0).Return([]model.RoomMessage{{ID: 1, RoomID: 5, ProfileID: 90, Text: "hello", CreatedAt: createdAt}}, nil)
	users.EXPECT().GetProfileSummary(gomock.Any(), &userpb.GetProfileSummaryRequest{ProfileId: 90}).Return(&userpb.GetProfileSummaryResponse{
		ProfileId: 90, UserAccountId: 9, FirstName: "Ann", LastName: "Player", Username: "ann",
	}, nil)

	list, err := service.ListRoomMessages(ctx, 9, 5, 0, -1)
	if err != nil {
		t.Fatalf("ListRoomMessages() error = %v", err)
	}
	if len(list) != 1 || list[0].Author.Name != "Ann Player" || list[0].CreatedAt != createdAt.Format(time.RFC3339Nano) {
		t.Fatalf("unexpected messages: %+v", list)
	}

	users.EXPECT().GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: 9}).Return(&userpb.GetProfileByUserAccountResponse{ProfileId: 90}, nil)
	rooms.EXPECT().Get(gomock.Any(), int64(5)).Return(&model.Room{ID: 5}, nil)
	members.EXPECT().IsMember(gomock.Any(), int64(5), int64(90)).Return(true, nil)
	messages.EXPECT().Add(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, message *model.RoomMessage) error {
		message.ID = 2
		message.CreatedAt = createdAt.Add(time.Minute)
		return nil
	})
	users.EXPECT().GetProfileSummary(gomock.Any(), &userpb.GetProfileSummaryRequest{ProfileId: 90}).Return(&userpb.GetProfileSummaryResponse{
		ProfileId: 90, UserAccountId: 9, Username: "ann",
	}, nil)

	sent, err := service.SendRoomMessage(ctx, 9, 5, " hello ")
	if err != nil {
		t.Fatalf("SendRoomMessage() error = %v", err)
	}
	if sent.ID != 2 || sent.Text != "hello" || sent.Author.Name != "ann" {
		t.Fatalf("unexpected sent message: %+v", sent)
	}
}
