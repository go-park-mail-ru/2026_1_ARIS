package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	elasticsearch "github.com/elastic/go-elasticsearch/v8"
	esindex "github.com/go-park-mail-ru/2026_1_ARIS/pkg/elasticsearch"
	mediapb "github.com/go-park-mail-ru/2026_1_ARIS/proto/media"
	mediamock "github.com/go-park-mail-ru/2026_1_ARIS/proto/media/mock"
	"github.com/golang/mock/gomock"
	"google.golang.org/grpc"
)

func TestSearch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	requestedSizes := make(map[string]int)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Size int `json:"size"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode search body: %v", err)
		}
		requestedSizes[strings.Trim(r.URL.Path, "/")] = body.Size
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
		switch {
		case strings.Contains(r.URL.Path, esindex.IndexUsers):
			_, _ = w.Write([]byte(`{"hits":{"hits":[{"_source":{"user_account_id":7,"profile_id":70,"username":"ann","first_name":"Ann","last_name":"User","avatar_id":11}}]}}`))
		case strings.Contains(r.URL.Path, esindex.IndexCommunities):
			_, _ = w.Write([]byte(`{"hits":{"hits":[{"_source":{"community_id":2,"profile_id":20,"username":"community","title":"Title","bio":"Bio","community_type":"public","avatar_id":12,"cover_media_id":13}}]}}`))
		case strings.Contains(r.URL.Path, esindex.IndexPosts):
			_, _ = w.Write([]byte(`{"hits":{"hits":[{"_source":{"post_id":3,"post_text":"Post","author_id":8,"author_profile_id":80,"author_username":"bob","author_first_name":"Bob","author_last_name":"Writer","author_avatar_id":14,"community_id":2,"created_at":"2026-05-27T10:00:00Z"}}]}}`))
		default:
			t.Fatalf("unexpected search path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{server.URL}})
	if err != nil {
		t.Fatalf("create es client: %v", err)
	}
	media := mediamock.NewMockMediaServiceClient(ctrl)
	mediaURLs := map[int64]string{
		11: "https://cdn/user",
		12: "https://cdn/community",
		13: "https://cdn/cover",
		14: "https://cdn/author",
	}
	media.EXPECT().
		GetMediaURL(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req *mediapb.GetMediaURLRequest, _ ...grpc.CallOption) (*mediapb.GetMediaURLResponse, error) {
			url, ok := mediaURLs[req.GetMediaId()]
			if !ok {
				t.Fatalf("unexpected media id: %d", req.GetMediaId())
			}
			return &mediapb.GetMediaURLResponse{Url: url}, nil
		}).
		Times(len(mediaURLs))

	result, err := New(client, media).Search(context.Background(), " query ", 100)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(result.Users) != 1 || *result.Users[0].AvatarURL != "https://cdn/user" {
		t.Fatalf("unexpected users: %+v", result.Users)
	}
	if len(result.Communities) != 1 || *result.Communities[0].AvatarURL != "https://cdn/community" || *result.Communities[0].CoverURL != "https://cdn/cover" {
		t.Fatalf("unexpected communities: %+v", result.Communities)
	}
	if len(result.Posts) != 1 || *result.Posts[0].AuthorAvatarURL != "https://cdn/author" || result.Posts[0].CreatedAt.IsZero() {
		t.Fatalf("unexpected posts: %+v", result.Posts)
	}
	for path, size := range requestedSizes {
		if size != maxLimit {
			t.Fatalf("expected capped size for %s to be %d, got %d", path, maxLimit, size)
		}
	}
}

func TestSearchInvalidInput(t *testing.T) {
	if _, err := New(nil, nil).Search(context.Background(), "   ", 10); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestMediaURL(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	if got := New(nil, nil).mediaURL(context.Background(), int64Ptr(1)); got != nil {
		t.Fatalf("expected nil media URL without client, got %#v", got)
	}

	media := mediamock.NewMockMediaServiceClient(ctrl)
	media.EXPECT().GetMediaURL(gomock.Any(), &mediapb.GetMediaURLRequest{MediaId: 9}).Return(&mediapb.GetMediaURLResponse{Url: "   "}, nil)
	if got := New(nil, media).mediaURL(context.Background(), int64Ptr(9)); got != nil {
		t.Fatalf("expected blank media URL to be ignored, got %#v", got)
	}
}

func int64Ptr(value int64) *int64 {
	return &value
}
