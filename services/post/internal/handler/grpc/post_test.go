package grpc

import (
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/services/post/internal/usecase"
)

func TestToProtoFeed(t *testing.T) {
	avatar := "https://cdn/avatar"
	createdAt := time.Date(2026, 5, 27, 10, 0, 0, 0, time.UTC)
	resp := toProtoFeed(usecase.FeedResult{
		Cursor:  "next",
		HasMore: true,
		Posts: []usecase.FeedPost{{
			ID:   1,
			Text: "text",
			Author: usecase.Author{
				ID: 10, FirstName: "Ann", LastName: "User", Username: "ann", AvatarURL: &avatar,
			},
			CreatedAt: createdAt,
			Likes:     2,
			Comments:  3,
			Reposts:   4,
			Medias:    []usecase.Media{{UID: "media-uid", MimeType: "image/png", URL: "https://cdn/media"}},
			Files:     []usecase.Media{{UID: "file-uid", MimeType: "text/plain", URL: "https://cdn/file"}},
		}},
	})

	if !resp.GetHasMore() || resp.GetNextCursor() != "next" || len(resp.GetPosts()) != 1 {
		t.Fatalf("unexpected feed response: %+v", resp)
	}
	post := resp.GetPosts()[0]
	if post.GetId() != "1" || post.GetAuthor().GetId() != "10" || post.GetAuthor().GetAvatarLink() != avatar {
		t.Fatalf("unexpected post response: %+v", post)
	}
	if post.GetCreatedAt() != createdAt.Format(time.RFC3339Nano) || post.GetLikes() != 2 || post.GetComments() != 3 || post.GetReposts() != 4 {
		t.Fatalf("unexpected counters/date: %+v", post)
	}
	if len(post.GetMedias()) != 1 || post.GetMedias()[0].GetId() != "media-uid" || len(post.GetFiles()) != 1 || post.GetFiles()[0].GetId() != "file-uid" {
		t.Fatalf("unexpected media/file response: %+v", post)
	}
}

func TestDerefString(t *testing.T) {
	value := "value"
	if derefString(&value) != value {
		t.Fatal("expected derefString to return value")
	}
	if derefString(nil) != "" {
		t.Fatal("expected nil derefString to return empty string")
	}
}
