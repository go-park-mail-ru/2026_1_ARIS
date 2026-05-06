package repository

import (
	legacycomment "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/comment"
	legacycommunity "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/community"
	legacylike "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/like"
	legacypost "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/post"
	legacyrepost "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/repost"
)

type Store struct {
	Posts         legacypost.PostRepo
	PostWithMedia legacypost.PostWithMediaRepo
	Comments      legacycomment.CommentRepo
	Likes         legacylike.LikeRepo
	Reposts       legacyrepost.RepostRepo
	Communities   legacycommunity.CommunityRepo
}

func NewStore(
	posts legacypost.PostRepo,
	postWithMedia legacypost.PostWithMediaRepo,
	comments legacycomment.CommentRepo,
	likes legacylike.LikeRepo,
	reposts legacyrepost.RepostRepo,
	communities ...legacycommunity.CommunityRepo,
) Store {
	var communityRepo legacycommunity.CommunityRepo
	if len(communities) > 0 {
		communityRepo = communities[0]
	}
	return Store{
		Posts:         posts,
		PostWithMedia: postWithMedia,
		Comments:      comments,
		Likes:         likes,
		Reposts:       reposts,
		Communities:   communityRepo,
	}
}
