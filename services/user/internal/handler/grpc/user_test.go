package grpc

import (
	"errors"
	"testing"
	"time"

	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/user/internal/model"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/user/internal/usecase"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestToAuthUserResponse(t *testing.T) {
	email := "ann@example.com"
	avatarID := int64(15)
	createdAt := time.Date(2026, 5, 27, 10, 0, 0, 0, time.UTC)
	resp := toAuthUserResponse(&usecase.AuthUser{
		UserAccountID: 1,
		UserProfileID: 2,
		ProfileID:     3,
		Login:         "ann",
		Email:         &email,
		FirstName:     "Ann",
		LastName:      "User",
		AvatarID:      &avatarID,
		CreatedAt:     createdAt,
	})

	if resp.GetUserAccountId() != 1 || resp.GetUserProfileId() != 2 || resp.GetProfileId() != 3 || resp.GetLogin() != "ann" {
		t.Fatalf("unexpected auth user response: %+v", resp)
	}
	if resp.GetEmail() != email || resp.GetAvatarId() != avatarID || resp.GetCreatedAt() != createdAt.Format(time.RFC3339Nano) {
		t.Fatalf("unexpected optional fields: %+v", resp)
	}
}

func TestGenderMapping(t *testing.T) {
	if fromProtoGender(userpb.Gender_GENDER_MALE) != model.Male {
		t.Fatal("expected male proto gender")
	}
	if fromProtoGender(userpb.Gender_GENDER_FEMALE) != model.Female {
		t.Fatal("expected female proto gender")
	}
	if fromProtoGender(userpb.Gender_GENDER_UNSPECIFIED) != "" {
		t.Fatal("expected unspecified proto gender to map to empty value")
	}
	if toRequiredGender(userpb.Gender_GENDER_MALE) != model.Male {
		t.Fatal("expected required male proto gender")
	}
	if toRequiredGender(userpb.Gender_GENDER_UNSPECIFIED) != model.Female {
		t.Fatal("expected required unspecified proto gender to default to female")
	}
}

func TestToStatus(t *testing.T) {
	cases := map[error]codes.Code{
		usecase.ErrUsernameTaken:        codes.AlreadyExists,
		usecase.ErrInvalidInput:         codes.InvalidArgument,
		usecase.ErrUserAccountNotFound:  codes.NotFound,
		usecase.ErrProfileNotFound:      codes.NotFound,
		usecase.ErrUserProfileNotFound:  codes.NotFound,
		errors.New("unexpected storage"): codes.Internal,
	}
	for err, want := range cases {
		if got := status.Code(toStatus(err)); got != want {
			t.Fatalf("toStatus(%v) = %v, want %v", err, got, want)
		}
	}
}
