package repository

import (
	"context"
	"errors"
	"testing"

	repomocks "github.com/go-park-mail-ru/2026_1_ARIS/services/post/internal/repository/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestClickhouseRecommendationCandidates(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	conn := repomocks.NewMockCHConn(ctrl)
	repo := NewClickhouseRecommendationRepo(conn)

	conn.EXPECT().Query(gomock.Any(), gomock.Any(), gomock.Any()).Return(int64Rows(ctrl, []int64{10, 11}), nil)
	conn.EXPECT().Query(gomock.Any(), gomock.Any(), gomock.Any()).Return(int64Rows(ctrl, []int64{11, 12}), nil)
	conn.EXPECT().Query(gomock.Any(), gomock.Any(), gomock.Any()).Return(int64Rows(ctrl, []int64{12, 13}), nil)
	conn.EXPECT().Query(gomock.Any(), gomock.Any(), gomock.Any()).Return(int64Rows(ctrl, []int64{13, 14}), nil)

	got, err := repo.GetForYouCandidates(context.Background(), ForYouInput{
		ProfileID:       1,
		FriendIDs:       []int64{2, 3},
		Limit:           20,
		ExcludeAuthorID: 1,
	})

	require.NoError(t, err)
	require.Equal(t, []int64{10, 11, 12, 13, 14}, got)
}

func TestClickhouseRecommendationQueries(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	conn := repomocks.NewMockCHConn(ctrl)
	repo := NewClickhouseRecommendationRepo(conn)

	conn.EXPECT().Query(gomock.Any(), gomock.Any(), gomock.Any()).Return(int64Rows(ctrl, []int64{1, 2}), nil)
	trending, err := repo.GetTrendingCandidates(context.Background(), 10, 5)
	require.NoError(t, err)
	require.Equal(t, []int64{1, 2}, trending)

	conn.EXPECT().Query(gomock.Any(), gomock.Any(), gomock.Any()).Return(affinityRows(ctrl, [][2]int64{{7, 2500}, {8, 500}}), nil)
	authors, err := repo.GetAuthorAffinity(context.Background(), 1, 30)
	require.NoError(t, err)
	require.Equal(t, map[int64]float64{7: 2.5, 8: 0.5}, authors)

	conn.EXPECT().Query(gomock.Any(), gomock.Any(), gomock.Any()).Return(affinityRows(ctrl, [][2]int64{{9, 1500}}), nil)
	communities, err := repo.GetCommunityAffinity(context.Background(), 1, 30)
	require.NoError(t, err)
	require.Equal(t, map[int64]float64{9: 1.5}, communities)

	conn.EXPECT().Query(gomock.Any(), gomock.Any(), gomock.Any()).Return(statsRows(ctrl, []struct {
		postID int64
		stats  PostStats
	}{{postID: 4, stats: PostStats{Views: 100, Likes: 2, Comments: 3, Reposts: 1}}}), nil)
	stats, err := repo.GetPostStats(context.Background(), []int64{4})
	require.NoError(t, err)
	require.Equal(t, PostStats{Views: 100, Likes: 2, Comments: 3, Reposts: 1}, stats[4])

	empty, err := repo.GetPostStats(context.Background(), nil)
	require.NoError(t, err)
	require.Empty(t, empty)
}

func TestClickhouseRecommendationQueryErrors(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	conn := repomocks.NewMockCHConn(ctrl)
	repo := NewClickhouseRecommendationRepo(conn)
	dbErr := errors.New("clickhouse down")

	conn.EXPECT().Query(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, dbErr)
	_, err := repo.GetTrendingCandidates(context.Background(), 10, 5)
	require.ErrorIs(t, err, dbErr)

	conn.EXPECT().Query(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, dbErr)
	_, err = repo.GetAuthorAffinity(context.Background(), 1, 30)
	require.ErrorIs(t, err, dbErr)

	conn.EXPECT().Query(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, dbErr)
	_, err = repo.GetCommunityAffinity(context.Background(), 1, 30)
	require.ErrorIs(t, err, dbErr)

	conn.EXPECT().Query(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, dbErr)
	_, err = repo.GetPostStats(context.Background(), []int64{1})
	require.ErrorIs(t, err, dbErr)
}

func TestScorePost(t *testing.T) {
	t.Parallel()

	base := ScorePost(PostStats{Views: 100, Likes: 2, Comments: 1, Reposts: 1}, 0, 0, false, false)
	withSignals := ScorePost(PostStats{Views: 100, Likes: 2, Comments: 1, Reposts: 1}, 1.5, 2, true, true)

	require.Greater(t, base, 0.0)
	require.Greater(t, withSignals, base)
	require.InDelta(t, base+3.0+2.5+3.0+3.0, withSignals, 0.0001)
}

func int64Rows(ctrl *gomock.Controller, ids []int64) *repomocks.MockCHRows {
	rows := repomocks.NewMockCHRows(ctrl)
	for _, id := range ids {
		id := id
		rows.EXPECT().Next().Return(true)
		rows.EXPECT().Scan(gomock.Any()).DoAndReturn(func(dest ...any) error {
			*(dest[0].(*int64)) = id
			return nil
		})
	}
	rows.EXPECT().Next().Return(false)
	rows.EXPECT().Err().Return(nil)
	rows.EXPECT().Close().Return(nil)
	return rows
}

func affinityRows(ctrl *gomock.Controller, values [][2]int64) *repomocks.MockCHRows {
	rows := repomocks.NewMockCHRows(ctrl)
	for _, value := range values {
		value := value
		rows.EXPECT().Next().Return(true)
		rows.EXPECT().Scan(gomock.Any(), gomock.Any()).DoAndReturn(func(dest ...any) error {
			*(dest[0].(*int64)) = value[0]
			*(dest[1].(*int64)) = value[1]
			return nil
		})
	}
	rows.EXPECT().Next().Return(false)
	rows.EXPECT().Err().Return(nil)
	rows.EXPECT().Close().Return(nil)
	return rows
}

func statsRows(ctrl *gomock.Controller, values []struct {
	postID int64
	stats  PostStats
}) *repomocks.MockCHRows {
	rows := repomocks.NewMockCHRows(ctrl)
	for _, value := range values {
		value := value
		rows.EXPECT().Next().Return(true)
		rows.EXPECT().Scan(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(dest ...any) error {
			*(dest[0].(*int64)) = value.postID
			*(dest[1].(*int64)) = value.stats.Views
			*(dest[2].(*int64)) = value.stats.Likes
			*(dest[3].(*int64)) = value.stats.Comments
			*(dest[4].(*int64)) = value.stats.Reposts
			return nil
		})
	}
	rows.EXPECT().Next().Return(false)
	rows.EXPECT().Err().Return(nil)
	rows.EXPECT().Close().Return(nil)
	return rows
}
