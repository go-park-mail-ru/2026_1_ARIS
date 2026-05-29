package http

//go:generate mockgen -source=service.go -destination=mocks/service_mock.go -package=mocks

import (
	"context"

	"github.com/go-park-mail-ru/2026_1_ARIS/services/game/internal/usecase"
)

type GameService interface {
	SetNotifier(usecase.Notifier)
	CreateRoom(context.Context, int64, usecase.CreateRoomInput) (usecase.Room, error)
	JoinRoom(context.Context, int64, string, string, string) (usecase.Room, error)
	ListRooms(context.Context, int64, int, int) ([]usecase.Room, error)
	GetRoom(context.Context, int64, int64) (usecase.Room, error)
	DisbandRoom(context.Context, int64, int64) error
	LeaveRoom(context.Context, int64, int64) error
	KickPlayer(context.Context, int64, int64, int64) error
	SetReady(context.Context, int64, int64, bool) error
	SetReplayReady(context.Context, int64, int64, bool) (usecase.Room, error)
	UpdateRoomPassword(context.Context, int64, int64, string) error
	UpdateRoomTitle(context.Context, int64, int64, string) error
	UpdateRoomRanked(context.Context, int64, int64, bool) error
	AssignAdmin(context.Context, int64, int64, int64) error
	StartRoom(context.Context, int64, int64) (usecase.Room, error)
	SubmitAnswer(context.Context, int64, int64, float64) (usecase.Room, error)
	PauseRoom(context.Context, int64, int64) (usecase.Room, error)
	ForceResumeRoom(context.Context, int64, int64) (usecase.Room, error)
	ListRoomMessages(context.Context, int64, int64, int, int) ([]usecase.RoomMessage, error)
	SendRoomMessage(context.Context, int64, int64, string) (usecase.RoomMessage, error)
	History(context.Context, int64, int, int) ([]usecase.HistoryItem, error)
	Stats(context.Context, int64) (usecase.Stats, error)
	Leaderboard(context.Context, string, int, int) (usecase.Leaderboard, error)
	ListQuestions(context.Context, string, bool, int, int) ([]usecase.Question, error)
	CreateQuestion(context.Context, int64, usecase.QuestionInput) (usecase.Question, error)
	UpdateQuestion(context.Context, int64, int64, usecase.QuestionInput) (usecase.Question, error)
	DeleteQuestion(context.Context, int64, int64) error
	TouchWaitingRoomMember(context.Context, int64, int64) error
	LeaveWaitingRoomOnDisconnect(context.Context, int64, int64) error
}
