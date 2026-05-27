package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/services/game/internal/model"
	repomocks "github.com/go-park-mail-ru/2026_1_ARIS/services/game/internal/repository/mocks"
	"github.com/golang/mock/gomock"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func TestGameRepositoriesReturnDBErrors(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	db := repomocks.NewMockDB(ctrl)
	row := repomocks.NewMockRow(ctrl)
	dbErr := errors.New("db down")

	row.EXPECT().Scan(gomock.Any()).Return(dbErr).AnyTimes()
	db.EXPECT().QueryRow(gomock.Any(), gomock.Any(), gomock.Any()).Return(row).AnyTimes()
	db.EXPECT().Exec(gomock.Any(), gomock.Any(), gomock.Any()).Return(pgconn.CommandTag{}, dbErr).AnyTimes()
	db.EXPECT().Query(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, dbErr).AnyTimes()

	ctx := context.Background()
	store := newStore(db, nil)

	require.Error(t, store.Questions.Create(ctx, &model.Question{TextRU: "q", TextEN: "q", CorrectAnswer: 1}))
	require.Error(t, store.Questions.Update(ctx, &model.Question{ID: 1}))
	require.Error(t, store.Questions.Delete(ctx, 1))
	_, err := store.Questions.Get(ctx, 1)
	require.Error(t, err)
	_, err = store.Questions.List(ctx, model.DefaultGameType, true, 10, 0)
	require.Error(t, err)
	_, err = store.Questions.Random(ctx, model.DefaultGameType, 10)
	require.Error(t, err)

	require.Error(t, store.Rooms.Create(ctx, &model.Room{Title: "room"}))
	_, err = store.Rooms.Get(ctx, 1)
	require.Error(t, err)
	_, err = store.Rooms.GetByInviteCode(ctx, "ABC123")
	require.Error(t, err)
	_, err = store.Rooms.GetForUpdate(ctx, 1)
	require.Error(t, err)
	_, err = store.Rooms.GetWaitingCreatedByProfile(ctx, 1)
	require.Error(t, err)
	_, err = store.Rooms.ListForProfile(ctx, 1, 10, 0)
	require.Error(t, err)
	_, err = store.Rooms.ExpiredActiveIDsForProfile(ctx, 1)
	require.Error(t, err)
	_, err = store.Rooms.HistoryForProfile(ctx, 1, 10, 0)
	require.Error(t, err)
	require.Error(t, store.Rooms.Update(ctx, &model.Room{ID: 1}))
	require.Error(t, store.Rooms.UpdateTitle(ctx, 1, "title"))
	require.Error(t, store.Rooms.UpdateAdmin(ctx, 1, 2))
	require.Error(t, store.Rooms.Deactivate(ctx, 1))
	require.Error(t, store.Rooms.TouchEmptyWaiting(ctx, 1))
	require.Error(t, store.Rooms.DeactivateEmptyWaitingOlderThan(ctx, time.Minute))

	require.Error(t, store.Members.Add(ctx, 1, 2))
	require.Error(t, store.Members.Deactivate(ctx, 1, 2))
	_, err = store.Members.DeactivateWaitingForProfile(ctx, 2)
	require.Error(t, err)
	_, err = store.Members.DeactivateStaleWaiting(ctx, time.Minute)
	require.Error(t, err)
	require.Error(t, store.Members.TouchWaiting(ctx, 1, 2))
	require.Error(t, store.Members.SetReady(ctx, 1, 2, true))
	require.Error(t, store.Members.ClearReady(ctx, 1))
	require.Error(t, store.Members.ResetForReplay(ctx, 1))
	require.Error(t, store.Members.SetPauseUsed(ctx, 1, 2))
	require.Error(t, store.Members.SetForceResumeRequested(ctx, 1, 2, true))
	require.Error(t, store.Members.ClearForceResumeRequests(ctx, 1))
	_, err = store.Members.List(ctx, 1)
	require.Error(t, err)
	_, err = store.Members.IsMember(ctx, 1, 2)
	require.Error(t, err)
	require.Error(t, store.Members.IncrementScore(ctx, 1, 2))
	_, err = store.Members.Stats(ctx, 2)
	require.Error(t, err)

	require.Error(t, store.RoomQs.Add(ctx, 1, 2, 1))
	require.Error(t, store.RoomQs.Clear(ctx, 1))
	_, err = store.RoomQs.List(ctx, 1)
	require.Error(t, err)
	_, err = store.RoomQs.GetByPosition(ctx, 1, 1)
	require.Error(t, err)
	_, err = store.RoomQs.GetActive(ctx, 1)
	require.Error(t, err)
	require.Error(t, store.RoomQs.Update(ctx, &model.RoomQuestion{ID: 1}))

	require.Error(t, store.Answers.Add(ctx, &model.Answer{ProfileID: 1}))
	_, err = store.Answers.List(ctx, 1)
	require.Error(t, err)
	_, err = store.Answers.Count(ctx, 1)
	require.Error(t, err)

	require.Error(t, store.Messages.Add(ctx, &model.RoomMessage{RoomID: 1, ProfileID: 2, Text: "hi"}))
	_, err = store.Messages.List(ctx, 1, 10, 0)
	require.Error(t, err)

	require.Error(t, store.Ratings.EnsureSeason(ctx, &model.RatingSeason{GameType: model.DefaultGameType}))
	_, err = store.Ratings.EnsurePlayerRatings(ctx, 1, model.DefaultGameType, []int64{1, 2})
	require.Error(t, err)
	_, err = store.Ratings.CountMatchesForGroup(ctx, model.DefaultGameType, "hash", time.Now(), time.Now())
	require.Error(t, err)
	require.Error(t, store.Ratings.AddMatch(ctx, &model.RatingMatch{}))
	require.Error(t, store.Ratings.AddMatchPlayer(ctx, &model.RatingMatchPlayer{}))
	require.Error(t, store.Ratings.ApplyPlayerRatingChange(ctx, 1, 2, 3, true, false))
	_, err = store.Ratings.RatingChangesForRoom(ctx, 1)
	require.Error(t, err)
	_, err = store.Ratings.Leaderboard(ctx, 1, 10, 0)
	require.Error(t, err)
}

func TestGameRepositoriesNoRowsAffected(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	db := repomocks.NewMockDB(ctrl)
	db.EXPECT().Exec(gomock.Any(), gomock.Any(), gomock.Any()).Return(pgconn.NewCommandTag("UPDATE 0"), nil).AnyTimes()

	ctx := context.Background()
	store := newStore(db, nil)

	require.ErrorIs(t, store.Questions.Update(ctx, &model.Question{ID: 1}), ErrNotFound)
	require.ErrorIs(t, store.Questions.Delete(ctx, 1), ErrNotFound)
	require.ErrorIs(t, store.Rooms.Update(ctx, &model.Room{ID: 1}), ErrNotFound)
	require.ErrorIs(t, store.Rooms.UpdateTitle(ctx, 1, "title"), ErrNotFound)
	require.ErrorIs(t, store.Rooms.UpdateAdmin(ctx, 1, 2), ErrNotFound)
	require.ErrorIs(t, store.Rooms.Deactivate(ctx, 1), ErrNotFound)
}
