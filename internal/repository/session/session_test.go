package session

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestSessionRedis_Save(t *testing.T) {
	db, mock := redismock.NewClientMock()
	repo := &sessionRedis{client: db}

	session := models.Session{
		SessionID: "s1",
		ExpiredAt: time.Now().Add(time.Hour),
	}

	data, _ := json.Marshal(session)
	ttl := time.Until(session.ExpiredAt)

	mock.ExpectSet("s1", data, ttl).SetVal("OK")

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(context.Background(), mockLogger)

	err := repo.Save(ctx, session)
	require.NoError(t, err)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSessionRedis_Delete(t *testing.T) {
	db, mock := redismock.NewClientMock()
	repo := &sessionRedis{client: db}

	mock.ExpectDel("s1").SetVal(1)

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(context.Background(), mockLogger)

	err := repo.Delete(ctx, models.SessionID("s1"))
	require.NoError(t, err)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSessionRedis_GetByID_Success(t *testing.T) {
	db, mock := redismock.NewClientMock()
	repo := &sessionRedis{client: db}

	session := models.Session{
		SessionID: "s1",
		ExpiredAt: time.Now().Add(time.Hour),
	}

	data, _ := json.Marshal(session)

	mock.ExpectGet("s1").SetVal(string(data))

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(context.Background(), mockLogger)

	got, err := repo.GetByID(ctx, models.SessionID("s1"))
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, session.SessionID, got.SessionID)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSessionRedis_GetByID_NotFound(t *testing.T) {
	db, mock := redismock.NewClientMock()
	repo := &sessionRedis{client: db}

	mock.ExpectGet("missing").RedisNil()

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(context.Background(), mockLogger)

	got, err := repo.GetByID(ctx, models.SessionID("missing"))
	require.Error(t, err)
	require.Nil(t, got)
	require.ErrorIs(t, err, xerrors.SessionNotFound)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSessionRedis_GetByID_RedisError(t *testing.T) {
	db, mock := redismock.NewClientMock()
	repo := &sessionRedis{client: db}

	mock.ExpectGet("s1").SetErr(errors.New("redis failed"))

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(context.Background(), mockLogger)

	got, err := repo.GetByID(ctx, models.SessionID("s1"))
	require.Error(t, err)
	require.Nil(t, got)
	require.EqualError(t, err, "redis failed")

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSessionRedis_GetByID_BadJSON(t *testing.T) {
	db, mock := redismock.NewClientMock()
	repo := &sessionRedis{client: db}

	mock.ExpectGet("s1").SetVal("{invalid json")

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(context.Background(), mockLogger)

	got, err := repo.GetByID(ctx, models.SessionID("s1"))
	require.Error(t, err)
	require.Nil(t, got)

	require.NoError(t, mock.ExpectationsWereMet())
}
