package grpc

import (
	"errors"
	"testing"
	"time"

	authservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/auth/service"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	authpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/auth"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAuthGrpcMappingHelpers(t *testing.T) {
	require.NotNil(t, New(nil))
	require.Equal(t, models.Male, fromProtoGender(authpb.Gender_GENDER_MALE))
	require.Equal(t, models.Female, fromProtoGender(authpb.Gender_GENDER_FEMALE))

	avatar := "https://cdn.test/a.png"
	now := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	resp := toProtoAuthResponse(&authservice.AuthResult{
		User: authservice.User{
			UserAccountID: 10,
			ProfileID:     20,
			FirstName:     "Neo",
			LastName:      "Anderson",
			AvatarURL:     &avatar,
			CreatedAt:     now,
		},
		Session: models.Session{SessionID: "sid", UserID: 10, ExpiredAt: now.Add(time.Hour)},
	})

	require.Equal(t, int64(10), resp.User.UserAccountId)
	require.Equal(t, int64(20), resp.User.ProfileId)
	require.Equal(t, avatar, resp.User.GetAvatarUrl())
	require.Equal(t, "sid", resp.Session.Id)

	require.Equal(t, codes.AlreadyExists, status.Code(toStatus(authservice.ErrLoginAlreadyExists)))
	require.Equal(t, codes.Unauthenticated, status.Code(toStatus(authservice.ErrInvalidCredentials)))
	require.Equal(t, codes.Unauthenticated, status.Code(toStatus(authservice.ErrSessionNotFound)))
	require.Equal(t, codes.InvalidArgument, status.Code(toStatus(authservice.ErrInvalidInput)))
	require.Equal(t, codes.InvalidArgument, status.Code(toStatus(authservice.ErrInvalidBirthday)))
	require.Equal(t, codes.InvalidArgument, status.Code(toStatus(authservice.ErrTooYoung)))
	require.Equal(t, codes.Internal, status.Code(toStatus(errors.New("boom"))))
}
