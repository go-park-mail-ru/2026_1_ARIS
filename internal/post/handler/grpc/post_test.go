package grpc

import (
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/post/service"
	"github.com/stretchr/testify/require"
)

func TestToProtoFeed(t *testing.T) {
	avatar := "https://cdn.test/avatar.png"
	createdAt := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	resp := toProtoFeed(service.FeedResult{
		Cursor:  "next",
		HasMore: true,
		Posts: []service.FeedPost{{
			ID:        10,
			Text:      "hello",
			Author:    service.Author{ID: 20, FirstName: "Neo", LastName: "Anderson", Username: "neo", AvatarURL: &avatar},
			CreatedAt: createdAt,
			Likes:     1,
			Comments:  2,
			Reposts:   3,
			Medias:    []service.Media{{UID: "media-uid", MimeType: "image/png", URL: "https://cdn.test/m.png"}},
		}},
	})

	require.Equal(t, "next", resp.NextCursor)
	require.True(t, resp.HasMore)
	require.Len(t, resp.Posts, 1)
	require.Equal(t, "10", resp.Posts[0].Id)
	require.Equal(t, "20", resp.Posts[0].Author.Id)
	require.Equal(t, avatar, resp.Posts[0].Author.AvatarLink)
	require.Equal(t, "media-uid", resp.Posts[0].Medias[0].Id)
	require.Equal(t, "", derefString(nil))
}
