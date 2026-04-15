package message

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	repomocks "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestMessageServiceSendMessage(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := repomocks.NewMockMessageRepo(ctrl)
	svc := NewMessageService(repo)
	repo.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, m *models.Message) error {
		require.Equal(t, int64(3), m.ChatID)
		m.ID = 99
		return nil
	})
	got, err := svc.SendMessage(context.Background(), 3, 7, "hi")
	require.NoError(t, err)
	require.Equal(t, int64(99), got.ID)
	require.Equal(t, int64(7), got.AuthorID)
}

func TestMessageServiceSendMessageSaveError(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := repomocks.NewMockMessageRepo(ctrl)
	svc := NewMessageService(repo)
	repo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(errors.New("fail"))
	_, err := svc.SendMessage(context.Background(), 1, 1, "x")
	require.EqualError(t, err, "fail")
}

func TestMessageServiceGetMessages(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := repomocks.NewMockMessageRepo(ctrl)
	svc := NewMessageService(repo)
	want := []models.Message{{ID: 1, ChatID: 2}}
	repo.EXPECT().GetByChatID(gomock.Any(), int64(2), 10, 0).Return(want, nil)
	got, err := svc.GetMessages(context.Background(), 2, 10, 0)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestMessageServiceUpdateMessage(t *testing.T) {
	t.Parallel()
	text := "old"
	newText := "new"
	msg := &models.Message{ID: 5, AuthorID: 10, Text: &text, UpdatedAt: time.Now()}

	t.Run("get error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		repo := repomocks.NewMockMessageRepo(ctrl)
		svc := NewMessageService(repo)
		repo.EXPECT().GetByID(gomock.Any(), int64(5)).Return(nil, errors.New("nf"))
		_, err := svc.UpdateMessage(context.Background(), 5, 10, newText)
		require.EqualError(t, err, "nf")
	})

	t.Run("forbidden", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		repo := repomocks.NewMockMessageRepo(ctrl)
		svc := NewMessageService(repo)
		repo.EXPECT().GetByID(gomock.Any(), int64(5)).Return(&models.Message{ID: 5, AuthorID: 99}, nil)
		_, err := svc.UpdateMessage(context.Background(), 5, 10, newText)
		require.ErrorContains(t, err, "forbidden")
	})

	t.Run("update error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		repo := repomocks.NewMockMessageRepo(ctrl)
		svc := NewMessageService(repo)
		repo.EXPECT().GetByID(gomock.Any(), int64(5)).Return(msg, nil)
		repo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(errors.New("db"))
		_, err := svc.UpdateMessage(context.Background(), 5, 10, newText)
		require.EqualError(t, err, "db")
	})

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		repo := repomocks.NewMockMessageRepo(ctrl)
		svc := NewMessageService(repo)
		repo.EXPECT().GetByID(gomock.Any(), int64(5)).Return(msg, nil)
		repo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
		got, err := svc.UpdateMessage(context.Background(), 5, 10, newText)
		require.NoError(t, err)
		require.Equal(t, newText, *got.Text)
	})
}
