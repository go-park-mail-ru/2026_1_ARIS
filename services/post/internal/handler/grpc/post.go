package grpc

import (
	"context"
	"strconv"
	"time"

	postpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/post"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/post/internal/usecase"
)

type Server struct {
	postpb.UnimplementedPostServiceServer
	post *usecase.Service
}

func New(post *usecase.Service) *Server {
	return &Server{post: post}
}

func (s *Server) GetFeed(ctx context.Context, req *postpb.GetFeedRequest) (*postpb.FeedResponse, error) {
	feed, err := s.post.GetFeed(ctx, req.GetCursor(), int(req.GetLimit()))
	if err != nil {
		return nil, usecase.ToStatus(err)
	}
	return toProtoFeed(feed), nil
}

func (s *Server) GetPublicFeed(ctx context.Context, req *postpb.GetFeedRequest) (*postpb.FeedResponse, error) {
	feed, err := s.post.GetPublicFeed(ctx, req.GetCursor(), int(req.GetLimit()))
	if err != nil {
		return nil, usecase.ToStatus(err)
	}
	return toProtoFeed(feed), nil
}

func toProtoFeed(feed usecase.FeedResult) *postpb.FeedResponse {
	posts := make([]*postpb.FeedPost, 0, len(feed.Posts))
	for _, post := range feed.Posts {
		medias := make([]*postpb.Media, 0, len(post.Medias))
		for _, media := range post.Medias {
			medias = append(medias, &postpb.Media{
				Id:        media.UID,
				MimeType:  media.MimeType,
				MediaLink: media.URL,
			})
		}
		files := make([]*postpb.Media, 0, len(post.Files))
		for _, file := range post.Files {
			files = append(files, &postpb.Media{
				Id:        file.UID,
				MimeType:  file.MimeType,
				MediaLink: file.URL,
			})
		}
		posts = append(posts, &postpb.FeedPost{
			Id:   strconv.FormatInt(post.ID, 10),
			Text: post.Text,
			Author: &postpb.Author{
				Id:         strconv.FormatInt(post.Author.ID, 10),
				FirstName:  post.Author.FirstName,
				LastName:   post.Author.LastName,
				Username:   post.Author.Username,
				AvatarLink: derefString(post.Author.AvatarURL),
			},
			CreatedAt: post.CreatedAt.UTC().Format(time.RFC3339Nano),
			Likes:     int32(post.Likes),
			Comments:  int32(post.Comments),
			Reposts:   int32(post.Reposts),
			Medias:    medias,
			Files:     files,
		})
	}
	return &postpb.FeedResponse{Posts: posts, NextCursor: feed.Cursor, HasMore: feed.HasMore}
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
