package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	usermock "github.com/go-park-mail-ru/2026_1_ARIS/proto/user/mock"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/auth/internal/model"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
)

type changePasswordSessionRepo struct {
	session *model.Session
	deleted model.SessionID
}

func (r *changePasswordSessionRepo) Save(context.Context, model.Session) error {
	return nil
}

func (r *changePasswordSessionRepo) Delete(_ context.Context, id model.SessionID) error {
	r.deleted = id
	return nil
}

func (r *changePasswordSessionRepo) GetByID(_ context.Context, id model.SessionID) (*model.Session, error) {
	if r.session == nil || r.session.SessionID != id {
		return nil, errors.New("not found")
	}
	return r.session, nil
}

func newPasswordHash(t *testing.T, password string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err)
	return string(hash)
}

func TestServiceChangePasswordSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	users := usermock.NewMockUserServiceClient(ctrl)
	sessionID := model.SessionID("session-1")
	sessions := &changePasswordSessionRepo{
		session: &model.Session{
			SessionID: sessionID,
			UserID:    42,
			ExpiredAt: time.Now().Add(time.Hour),
		},
	}
	service := New(sessions, users, nil)
	oldHash := newPasswordHash(t, "old-secret")

	users.EXPECT().
		GetAuthUserByAccount(gomock.Any(), &userpb.GetAuthUserByAccountRequest{UserAccountId: 42}).
		Return(&userpb.AuthUserResponse{UserAccountId: 42, Login: "neo"}, nil)
	users.EXPECT().
		GetCredentialsByLogin(gomock.Any(), &userpb.GetCredentialsByLoginRequest{Login: "neo"}).
		Return(&userpb.GetCredentialsByLoginResponse{UserAccountId: 42, PasswordHash: oldHash}, nil)
	users.EXPECT().
		UpdatePasswordHash(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req *userpb.UpdatePasswordHashRequest, _ ...grpc.CallOption) (*userpb.UpdatePasswordHashResponse, error) {
			require.Equal(t, int64(42), req.GetUserAccountId())
			require.NoError(t, bcrypt.CompareHashAndPassword([]byte(req.GetPasswordHash()), []byte("new-secret")))
			return &userpb.UpdatePasswordHashResponse{Ok: true}, nil
		})

	err := service.ChangePassword(context.Background(), string(sessionID), ChangePasswordInput{
		OldPassword:  "old-secret",
		NewPassword1: "new-secret",
		NewPassword2: "new-secret",
	})

	require.NoError(t, err)
	require.Equal(t, sessionID, sessions.deleted)
}

func TestServiceChangePasswordValidation(t *testing.T) {
	tests := []struct {
		name    string
		input   ChangePasswordInput
		wantErr error
	}{
		{
			name:    "passwords mismatch",
			input:   ChangePasswordInput{OldPassword: "old-secret", NewPassword1: "new-secret", NewPassword2: "other-secret"},
			wantErr: ErrPasswordMismatch,
		},
		{
			name:    "too short",
			input:   ChangePasswordInput{OldPassword: "old-secret", NewPassword1: "short", NewPassword2: "short"},
			wantErr: ErrWeakPassword,
		},
		{
			name:    "too long",
			input:   ChangePasswordInput{OldPassword: "old-secret", NewPassword1: "this-password-is-too-long", NewPassword2: "this-password-is-too-long"},
			wantErr: ErrWeakPassword,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			users := usermock.NewMockUserServiceClient(ctrl)
			sessions := &changePasswordSessionRepo{
				session: &model.Session{
					SessionID: "session-1",
					UserID:    42,
					ExpiredAt: time.Now().Add(time.Hour),
				},
			}
			service := New(sessions, users, nil)

			err := service.ChangePassword(context.Background(), "session-1", tc.input)

			require.ErrorIs(t, err, tc.wantErr)
			require.Empty(t, sessions.deleted)
		})
	}
}

func TestServiceChangePasswordRejectsCurrentPassword(t *testing.T) {
	ctrl := gomock.NewController(t)
	users := usermock.NewMockUserServiceClient(ctrl)
	sessions := &changePasswordSessionRepo{
		session: &model.Session{
			SessionID: "session-1",
			UserID:    42,
			ExpiredAt: time.Now().Add(time.Hour),
		},
	}
	service := New(sessions, users, nil)
	oldHash := newPasswordHash(t, "old-secret")

	users.EXPECT().
		GetAuthUserByAccount(gomock.Any(), gomock.Any()).
		Return(&userpb.AuthUserResponse{UserAccountId: 42, Login: "neo"}, nil)
	users.EXPECT().
		GetCredentialsByLogin(gomock.Any(), gomock.Any()).
		Return(&userpb.GetCredentialsByLoginResponse{UserAccountId: 42, PasswordHash: oldHash}, nil)

	err := service.ChangePassword(context.Background(), "session-1", ChangePasswordInput{
		OldPassword:  "old-secret",
		NewPassword1: "old-secret",
		NewPassword2: "old-secret",
	})

	require.ErrorIs(t, err, ErrPasswordReuse)
	require.Empty(t, sessions.deleted)
}

func TestServiceChangePasswordRejectsWrongOldPassword(t *testing.T) {
	ctrl := gomock.NewController(t)
	users := usermock.NewMockUserServiceClient(ctrl)
	sessions := &changePasswordSessionRepo{
		session: &model.Session{
			SessionID: "session-1",
			UserID:    42,
			ExpiredAt: time.Now().Add(time.Hour),
		},
	}
	service := New(sessions, users, nil)
	oldHash := newPasswordHash(t, "old-secret")

	users.EXPECT().
		GetAuthUserByAccount(gomock.Any(), gomock.Any()).
		Return(&userpb.AuthUserResponse{UserAccountId: 42, Login: "neo"}, nil)
	users.EXPECT().
		GetCredentialsByLogin(gomock.Any(), gomock.Any()).
		Return(&userpb.GetCredentialsByLoginResponse{UserAccountId: 42, PasswordHash: oldHash}, nil)

	err := service.ChangePassword(context.Background(), "session-1", ChangePasswordInput{
		OldPassword:  "wrong-secret",
		NewPassword1: "new-secret",
		NewPassword2: "new-secret",
	})

	require.ErrorIs(t, err, ErrInvalidCredentials)
	require.Empty(t, sessions.deleted)
}

func TestServiceChangePasswordRequiresSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	users := usermock.NewMockUserServiceClient(ctrl)
	service := New(&changePasswordSessionRepo{}, users, nil)

	err := service.ChangePassword(context.Background(), "", ChangePasswordInput{
		OldPassword:  "old-secret",
		NewPassword1: "new-secret",
		NewPassword2: "new-secret",
	})

	require.ErrorIs(t, err, ErrSessionNotFound)
}
