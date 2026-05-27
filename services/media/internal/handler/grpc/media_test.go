package grpc

import (
	"context"
	"errors"
	"testing"

	mediapb "github.com/go-park-mail-ru/2026_1_ARIS/proto/media"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/media/internal/model"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/media/internal/repository"
	repositorymock "github.com/go-park-mail-ru/2026_1_ARIS/services/media/internal/repository/mocks"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/media/internal/usecase"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestServerGetMediaURL(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mediaRepo := repositorymock.NewMockMediaRepo(ctrl)
	server := New(usecase.New(repository.NewStore(mediaRepo, nil, ""), nil))

	mediaRepo.EXPECT().Get(gomock.Any(), int64(10)).Return(&model.Media{ID: 10, Link: "https://cdn/file.png"}, nil)
	resp, err := server.GetMediaURL(context.Background(), &mediapb.GetMediaURLRequest{MediaId: 10})
	if err != nil {
		t.Fatalf("GetMediaURL() error = %v", err)
	}
	if resp.GetUrl() != "https://cdn/file.png" {
		t.Fatalf("unexpected URL: %+v", resp)
	}
}

func TestServerGetMedia(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mediaRepo := repositorymock.NewMockMediaRepo(ctrl)
	server := New(usecase.New(repository.NewStore(mediaRepo, nil, ""), nil))

	id := uuid.New()
	mediaRepo.EXPECT().Get(gomock.Any(), int64(11)).Return(&model.Media{ID: 11, Uid: id, MimeType: "image/png", Link: "/api/media/11"}, nil)
	resp, err := server.GetMedia(context.Background(), &mediapb.GetMediaRequest{MediaId: 11})
	if err != nil {
		t.Fatalf("GetMedia() error = %v", err)
	}
	if resp.GetMediaId() != 11 || resp.GetUid() != id.String() || resp.GetMimeType() != "image/png" || resp.GetUrl() != "/api/media/11" {
		t.Fatalf("unexpected media response: %+v", resp)
	}
}

func TestToStatus(t *testing.T) {
	cases := map[error]codes.Code{
		usecase.ErrInvalidInput:   codes.InvalidArgument,
		usecase.ErrMediaNotFound:  codes.NotFound,
		errors.New("storage down"): codes.Internal,
	}
	for err, want := range cases {
		if got := status.Code(toStatus(err)); got != want {
			t.Fatalf("toStatus(%v) = %v, want %v", err, got, want)
		}
	}
}
