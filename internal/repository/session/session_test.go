package session

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/require"
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

	err := repo.Save(context.Background(), session)
	require.NoError(t, err)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSessionRedis_Delete(t *testing.T) {
	db, mock := redismock.NewClientMock()
	repo := &sessionRedis{client: db}

	mock.ExpectDel("s1").SetVal(1)

	err := repo.Delete(context.Background(), models.SessionID("s1"))
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

	got, err := repo.GetByID(context.Background(), models.SessionID("s1"))
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, session.SessionID, got.SessionID)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSessionRedis_GetByID_NotFound(t *testing.T) {
	db, mock := redismock.NewClientMock()
	repo := &sessionRedis{client: db}

	mock.ExpectGet("missing").RedisNil()

	got, err := repo.GetByID(context.Background(), models.SessionID("missing"))
	require.Error(t, err)
	require.Nil(t, got)
	require.ErrorIs(t, err, xerrors.SessionNotFound)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSessionRedis_GetByID_RedisError(t *testing.T) {
	db, mock := redismock.NewClientMock()
	repo := &sessionRedis{client: db}

	mock.ExpectGet("s1").SetErr(errors.New("redis failed"))

	got, err := repo.GetByID(context.Background(), models.SessionID("s1"))
	require.Error(t, err)
	require.Nil(t, got)
	require.EqualError(t, err, "redis failed")

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSessionRedis_GetByID_BadJSON(t *testing.T) {
	db, mock := redismock.NewClientMock()
	repo := &sessionRedis{client: db}

	mock.ExpectGet("s1").SetVal("{invalid json")

	got, err := repo.GetByID(context.Background(), models.SessionID("s1"))
	require.Error(t, err)
	require.Nil(t, got)

	require.NoError(t, mock.ExpectationsWereMet())
}
