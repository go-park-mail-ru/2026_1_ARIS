package usecase

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/services/game/internal/model"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/game/internal/repository"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGameHelpersNormalizeAndPassword(t *testing.T) {
	t.Parallel()

	in := CreateRoomInput{Title: " Room ", GameType: "", MaxPlayers: 99, QuestionCount: -1, AnswerTimeoutSec: 1, RoundPauseSec: 99}
	require.NoError(t, normalizeCreateInput(&in))
	require.Equal(t, "Room", in.Title)
	require.Equal(t, model.DefaultGameType, in.GameType)
	require.Equal(t, 8, in.MaxPlayers)
	require.Equal(t, 5, in.QuestionCount)
	require.Equal(t, 3, in.AnswerTimeoutSec)
	require.Equal(t, 60, in.RoundPauseSec)

	_, err := normalizeRoomTitle("")
	require.ErrorIs(t, err, ErrInvalidInput)
	_, err = normalizeRoomMessageText(" ")
	require.ErrorIs(t, err, ErrInvalidInput)
	text, err := normalizeRoomMessageText(" hello ")
	require.NoError(t, err)
	require.Equal(t, "hello", text)

	hash := passwordHashPtr(" secret ")
	require.NotNil(t, hash)
	require.True(t, passwordMatches(hash, "secret"))
	require.False(t, passwordMatches(hash, "bad"))
	require.True(t, passwordMatches(nil, ""))
	require.Nil(t, passwordHashPtr(" "))
	require.Nil(t, passwordValuePtr(" "))
	require.Equal(t, "secret", *passwordValuePtr(" secret "))

	room := model.Room{CreatedByProfileID: 10, PasswordValue: passwordValuePtr("secret")}
	require.Equal(t, "secret", roomPasswordForProfile(room, 10))
	require.Empty(t, roomPasswordForProfile(room, 11))
	require.Len(t, inviteCode(), 6)
}

func TestGameHelpersRatingAndWinners(t *testing.T) {
	t.Parallel()

	members := []model.RoomMember{
		{ProfileID: 3, Score: 0},
		{ProfileID: 1, Score: 5},
		{ProfileID: 2, Score: 3},
	}
	require.Equal(t, int64(1), *scoreWinner(members))
	require.Nil(t, scoreWinner([]model.RoomMember{{ProfileID: 1, Score: 1}, {ProfileID: 2, Score: 1}}))
	require.Nil(t, scoreWinner(nil))

	startedAt := time.Date(2026, time.May, 10, 12, 0, 0, 0, time.UTC)
	answers := []model.Answer{
		{ProfileID: 1, Answer: 10, Distance: 3, AnsweredAt: startedAt.Add(time.Second)},
		{ProfileID: 2, Answer: 8, Distance: 1, AnsweredAt: startedAt.Add(2 * time.Second)},
		{ProfileID: 3, Answer: 7, Distance: 1, AnsweredAt: startedAt.Add(time.Second)},
	}
	require.Equal(t, int64(3), *answerWinner(answers))
	require.Nil(t, answerWinner(nil))
	require.Equal(t, int64(1000), roundResultResponseTimeMs(answers[0], &startedAt))
	require.Equal(t, int64(0), roundResultResponseTimeMs(model.Answer{AnsweredAt: startedAt.Add(-time.Second)}, &startedAt))
	require.Equal(t, "0", roundResultAnswerGroupKey(0))
	require.NotZero(t, roundResultPositivePointCount(3, answers, &startedAt))
	require.NotZero(t, roundResultMaxRevealIndex(4, answers))

	require.Equal(t, []int64{1, 2, 3}, memberProfileIDs(members))
	require.Len(t, ratingGroupHash([]int64{3, 2, 1}), 64)
	require.Equal(t, 1.0, ratingWeightForOccurrence(1))
	require.Equal(t, 0.5, ratingWeightForOccurrence(2))
	require.Equal(t, 0.25, ratingWeightForOccurrence(3))
	require.Zero(t, ratingWeightForOccurrence(4))
	require.InDelta(t, 0.5, ratingExpectedScore(1000, 1000), 0.0001)
	require.Equal(t, map[int]int{0: 1, 3: 1, 5: 1}, ratingScoreCounts(members))
	require.Equal(t, 1, ratingPlaces(members)[1])
	require.Equal(t, []int64{1, 2}, memberProfileIDs(publicRoomPlayableMembers(
		model.Room{IsPublicLobby: true, CreatedByProfileID: 3},
		members,
	)))

	deltas := ratingDeltas(members, map[int64]int{1: 1000, 2: 1000, 3: 1000}, 1)
	require.Positive(t, deltas[1])
	require.Negative(t, deltas[3])
	require.Empty(t, ratingDeltas(members, nil, 0))
	require.Empty(t, ratingDeltas([]model.RoomMember{{ProfileID: 1, Score: 1}}, nil, 1))
}

func TestGameHelpersEventsQuestionsAndErrors(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.May, 10, 12, 0, 0, 0, time.UTC)
	later := now.Add(time.Minute)
	room := &model.Room{Status: model.RoomStatusActive, NextQuestionAt: &later}
	require.False(t, roomEventDue(room, now))
	require.True(t, roomEventDue(room, later))
	require.Equal(t, &later, nextRoomEvent(room))

	room.PausedByProfileID = int64Ptr(1)
	room.PauseStartedAt = &now
	room.PauseUntilAt = &later
	require.True(t, isRoomPaused(room))
	require.False(t, roomEventDue(room, now))
	require.True(t, roomEventDue(room, later))
	require.Equal(t, &later, nextRoomEvent(room))
	require.Nil(t, nextRoomEvent(nil))

	votes, required := forceResumeVotes([]model.RoomMember{
		{ProfileID: 1, ForceResumeRequested: true},
		{ProfileID: 2, ForceResumeRequested: true},
		{ProfileID: 3},
	}, 1)
	require.Equal(t, 1, votes)
	require.Equal(t, 2, required)

	question, err := mapQuestionInput(QuestionInput{GameType: "  ", Text: LocalizedText{RU: " сколько? ", EN: " how many? "}, CorrectAnswer: 42, IsActive: true})
	require.NoError(t, err)
	require.Equal(t, model.DefaultGameType, question.GameType)
	require.Equal(t, "сколько?", question.TextRU)
	require.Equal(t, "how many?", question.TextEN)
	_, err = mapQuestionInput(QuestionInput{Text: LocalizedText{RU: " ", EN: "valid text"}, CorrectAnswer: 1})
	require.ErrorIs(t, err, ErrInvalidInput)
	_, err = mapQuestionInput(QuestionInput{Text: LocalizedText{RU: "текст", EN: "text"}, CorrectAnswer: math.Inf(1)})
	require.ErrorIs(t, err, ErrInvalidInput)

	mapped := mapQuestions([]model.Question{question})
	require.Len(t, mapped, 1)
	require.Equal(t, 42.0, mapQuestionForRoom(question, true).CorrectAnswer)
	require.Zero(t, mapQuestionForRoom(question, false).CorrectAnswer)
	require.Equal(t, model.DefaultGameType, normalizeGameType(" "))
	require.Equal(t, 50, normalizeLimit(0, 50, 100))
	require.Equal(t, 100, normalizeLimit(500, 50, 100))
	require.Zero(t, normalizeOffset(-1))
	require.Nil(t, timePtrString(nil))
	require.NotNil(t, timePtrString(&now))
	require.Zero(t, int64PtrValue(nil))
	require.Equal(t, int64(7), int64PtrValue(int64Ptr(7)))

	require.ErrorIs(t, mapRepoErr(repository.ErrNotFound), ErrNotFound)
	require.ErrorIs(t, mapGRPCErr(status.Error(codes.NotFound, "missing")), ErrNotFound)
	require.ErrorIs(t, mapGRPCErr(status.Error(codes.InvalidArgument, "bad")), ErrInvalidInput)
	require.True(t, isUniqueViolation(&pgconn.PgError{Code: "23505"}))
	require.True(t, isRoomTitleUniqueViolation(&pgconn.PgError{Code: "23505", ConstraintName: "game_room_active_waiting_title_unique_idx"}))
	require.False(t, isRoomTitleUniqueViolation(errors.New("plain")))

	season := currentRatingSeason("classic", time.Date(2026, time.May, 20, 0, 0, 0, 0, time.UTC))
	require.Equal(t, 1, season.SeasonNumber)
	require.Equal(t, "classic", season.GameType)
	require.Contains(t, season.Title, "Май")
	require.Equal(t, "Январь", russianMonthName(time.January))
	require.Equal(t, "%!Month(99)", russianMonthName(time.Month(99)))
	require.Equal(t, season.Title, mapRatingSeason(season).Title)
}

func int64Ptr(value int64) *int64 {
	return &value
}
