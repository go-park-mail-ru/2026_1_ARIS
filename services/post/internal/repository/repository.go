package repository

//go:generate mockgen -source=repository.go -destination=mocks/repository_mock.go -package=mocks DB,MembershipRepo,PostRepo,PostMediaRepo,CommentRepo,LikeRepo,RepostRepo
//go:generate mockgen -destination=mocks/pgx_mock.go -package=mocks github.com/jackc/pgx/v5 Row,Rows,Tx

import (
	"context"
	"errors"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/post/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

var (
	ErrPostNotFound    = errors.New("post not found")
	ErrCommentNotFound = errors.New("comment not found")
)

type DB interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Begin(context.Context) (pgx.Tx, error)
}

type Store struct {
	Posts       PostRepo
	PostMedia   PostMediaRepo
	Comments    CommentRepo
	Likes       LikeRepo
	Reposts     RepostRepo
	Memberships MembershipRepo
}

func NewStore(db DB) Store {
	return Store{
		Posts:       NewPostStorage(db),
		PostMedia:   NewPostMediaStorage(db),
		Comments:    NewCommentStorage(db),
		Likes:       NewLikeStorage(db),
		Reposts:     NewRepostStorage(db),
		Memberships: NewMembershipStorage(db),
	}
}

type MembershipRepo interface {
	GetMemberCommunityIDs(ctx context.Context, profileID int64) ([]int64, error)
}

type membershipStorage struct{ db DB }

func NewMembershipStorage(db DB) MembershipRepo { return &membershipStorage{db: db} }

func (s *membershipStorage) GetMemberCommunityIDs(ctx context.Context, profileID int64) ([]int64, error) {
	rows, err := s.db.Query(ctx,
		`SELECT community_id FROM community_member WHERE profile_id=$1 AND is_active=true`,
		profileID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

type PostRepo interface {
	Save(ctx context.Context, post model.Post) (int64, error)
	Delete(ctx context.Context, id int64) error
	Update(ctx context.Context, post model.Post) error
	Get(ctx context.Context, id int64) (*model.Post, error)
	GetAll(ctx context.Context) ([]model.Post, error)
	GetByAuthorID(ctx context.Context, authorID int64) ([]model.Post, error)
	GetByCommunityID(ctx context.Context, communityID int64) ([]model.Post, error)
	GetByIDs(ctx context.Context, ids []int64) ([]model.Post, error)
	GetFeedPage(ctx context.Context, authorIDs []int64, beforeTime *time.Time, beforeID *int64, limit int, publicOnly bool) ([]model.Post, error)
	GetRecentPublicCommunityPostIDs(ctx context.Context, excludeAuthorID int64, limit int) ([]int64, error)
}

type postStorage struct {
	db DB
}

func NewPostStorage(db DB) PostRepo {
	return &postStorage{db: db}
}

func (s *postStorage) Save(ctx context.Context, post model.Post) (int64, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	start := time.Now()
	row := tx.QueryRow(ctx, `
		INSERT INTO post (uid, post_text, author_id, community_id, is_public_demo, allow_comments)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, uuid.New(), post.Text, post.AuthorID, post.CommunityID, post.IsPublicDemo, post.AllowComments)
	logQuery(ctx, "postStorage.Save", start)

	var postID int64
	if err := row.Scan(&postID); err != nil {
		return 0, err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO search_outbox (entity_type, entity_id, operation)
		VALUES ('post', $1, 'upsert')
	`, postID); err != nil {
		return 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return postID, nil
}

func (s *postStorage) Delete(ctx context.Context, id int64) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	start := time.Now()

	if _, err := tx.Exec(ctx, `
		INSERT INTO search_outbox (entity_type, entity_id, operation)
		VALUES ('post', $1, 'delete')
	`, id); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM like_record WHERE post_id=$1 OR comment_id IN (SELECT id FROM comment WHERE post_id=$1)`, id); err != nil {
		return err
	}

	tag, err := tx.Exec(ctx, `DELETE FROM post WHERE id=$1`, id)
	logQuery(ctx, "postStorage.Delete", start)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrPostNotFound
	}

	return tx.Commit(ctx)
}

func (s *postStorage) Update(ctx context.Context, post model.Post) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	start := time.Now()
	tag, err := tx.Exec(ctx, `UPDATE post SET post_text=$1, updated_at=$2 WHERE id=$3`, post.Text, post.UpdatedAt, post.ID)
	logQuery(ctx, "postStorage.Update", start)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrPostNotFound
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO search_outbox (entity_type, entity_id, operation)
		VALUES ('post', $1, 'upsert')
	`, post.ID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *postStorage) Get(ctx context.Context, id int64) (*model.Post, error) {
	var post model.Post
	start := time.Now()
	err := pgxscan.Get(ctx, s.db, &post, `SELECT * FROM post WHERE id=$1`, id)
	logQuery(ctx, "postStorage.Get", start)
	if err != nil {
		if pgxscan.NotFound(err) {
			return nil, ErrPostNotFound
		}
		return nil, err
	}
	return &post, nil
}

func (s *postStorage) GetAll(ctx context.Context) ([]model.Post, error) {
	start := time.Now()
	rows, err := s.db.Query(ctx, `SELECT * FROM post WHERE is_active=TRUE`)
	logQuery(ctx, "postStorage.GetAll", start)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[model.Post])
}

func (s *postStorage) GetByAuthorID(ctx context.Context, authorID int64) ([]model.Post, error) {
	start := time.Now()
	rows, err := s.db.Query(ctx, `SELECT * FROM post WHERE author_id=$1 AND is_active=TRUE ORDER BY created_at DESC`, authorID)
	logQuery(ctx, "postStorage.GetByAuthorID", start)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[model.Post])
}

func (s *postStorage) GetByCommunityID(ctx context.Context, communityID int64) ([]model.Post, error) {
	start := time.Now()
	rows, err := s.db.Query(ctx, `SELECT * FROM post WHERE community_id=$1 AND is_active=TRUE ORDER BY created_at DESC`, communityID)
	logQuery(ctx, "postStorage.GetByCommunityID", start)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[model.Post])
}

func (s *postStorage) GetByIDs(ctx context.Context, ids []int64) ([]model.Post, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	start := time.Now()
	rows, err := s.db.Query(ctx, `SELECT * FROM post WHERE id=ANY($1) AND is_active=TRUE`, ids)
	logQuery(ctx, "postStorage.GetByIDs", start)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[model.Post])
}

func (s *postStorage) GetFeedPage(ctx context.Context, authorIDs []int64, beforeTime *time.Time, beforeID *int64, limit int, publicOnly bool) ([]model.Post, error) {
	start := time.Now()
	var rows pgx.Rows
	var err error
	if len(authorIDs) == 0 {
		// public feed: no author filter
		if beforeTime != nil && beforeID != nil {
			rows, err = s.db.Query(ctx, `
				SELECT * FROM post
				WHERE is_active = TRUE AND is_public_demo = $1
				  AND (created_at, id) < ($2, $3)
				ORDER BY created_at DESC, id DESC
				LIMIT $4
			`, publicOnly, *beforeTime, *beforeID, limit)
		} else {
			rows, err = s.db.Query(ctx, `
				SELECT * FROM post
				WHERE is_active = TRUE AND is_public_demo = $1
				ORDER BY created_at DESC, id DESC
				LIMIT $2
			`, publicOnly, limit)
		}
	} else {
		if beforeTime != nil && beforeID != nil {
			rows, err = s.db.Query(ctx, `
				SELECT p.* FROM post p
				LEFT JOIN community c ON c.id = p.community_id
				WHERE p.is_active = TRUE AND p.is_public_demo = $1
				  AND (
				    p.author_id = ANY($2)
				    OR (p.community_id IS NOT NULL AND c.is_active = TRUE AND c.community_type = 'public')
				  )
				  AND (p.created_at, p.id) < ($3, $4)
				ORDER BY p.created_at DESC, p.id DESC
				LIMIT $5
			`, publicOnly, authorIDs, *beforeTime, *beforeID, limit)
		} else {
			rows, err = s.db.Query(ctx, `
				SELECT p.* FROM post p
				LEFT JOIN community c ON c.id = p.community_id
				WHERE p.is_active = TRUE AND p.is_public_demo = $1
				  AND (
				    p.author_id = ANY($2)
				    OR (p.community_id IS NOT NULL AND c.is_active = TRUE AND c.community_type = 'public')
				  )
				ORDER BY p.created_at DESC, p.id DESC
				LIMIT $3
			`, publicOnly, authorIDs, limit)
		}
	}
	logQuery(ctx, "postStorage.GetFeedPage", start)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[model.Post])
}

func (s *postStorage) GetRecentPublicCommunityPostIDs(ctx context.Context, excludeAuthorID int64, limit int) ([]int64, error) {
	start := time.Now()
	rows, err := s.db.Query(ctx, `
		SELECT p.id
		FROM post p
		JOIN community c ON c.id = p.community_id
		WHERE p.is_active = TRUE
		  AND p.is_public_demo = FALSE
		  AND c.is_active = TRUE
		  AND c.community_type = 'public'
		  AND ($1::bigint <= 0 OR p.author_id <> $1)
		ORDER BY p.created_at DESC, p.id DESC
		LIMIT $2
	`, excludeAuthorID, limit)
	logQuery(ctx, "postStorage.GetRecentPublicCommunityPostIDs", start)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]int64, 0, limit)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

type PostMediaRepo interface {
	GetMediaByPostID(ctx context.Context, postID int64) []int64
	GetDetailedMediaByPostID(ctx context.Context, postID int64) ([]model.AttachedMedia, error)
	GetDetailedMediaByPostIDs(ctx context.Context, postIDs []int64) (map[int64][]model.AttachedMedia, error)
	GetMediaAuthorID(ctx context.Context, mediaID int64) (int64, error)
	Save(ctx context.Context, postWithMedia model.PostWithMedia) error
	DeleteByPostID(ctx context.Context, postID int64) error
}

type postMediaStorage struct {
	db DB
}

func NewPostMediaStorage(db DB) PostMediaRepo {
	return &postMediaStorage{db: db}
}

func (s *postMediaStorage) GetMediaByPostID(ctx context.Context, postID int64) []int64 {
	start := time.Now()
	rows, err := s.db.Query(ctx, `SELECT media_id FROM post_with_media WHERE post_id=$1 ORDER BY sort_order`, postID)
	logQuery(ctx, "postMediaStorage.GetMediaByPostID", start)
	if err != nil {
		return nil
	}
	defer rows.Close()

	mediaIDs, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (int64, error) {
		var mediaID int64
		return mediaID, row.Scan(&mediaID)
	})
	if err != nil {
		return nil
	}
	return mediaIDs
}

func (s *postMediaStorage) GetDetailedMediaByPostID(ctx context.Context, postID int64) ([]model.AttachedMedia, error) {
	start := time.Now()
	var items []model.AttachedMedia
	err := pgxscan.Select(ctx, s.db, &items, `
		SELECT pwm.media_id, m.uid, m.media_name, m.mime_type, m.link, m.author_id, pwm.sort_order
		FROM post_with_media pwm
		JOIN media m ON m.id=pwm.media_id AND m.is_active=TRUE
		WHERE pwm.post_id=$1
		ORDER BY pwm.sort_order
	`, postID)
	logQuery(ctx, "postMediaStorage.GetDetailedMediaByPostID", start)
	if err != nil && !pgxscan.NotFound(err) {
		return nil, err
	}
	return items, nil
}

func (s *postMediaStorage) GetDetailedMediaByPostIDs(ctx context.Context, postIDs []int64) (map[int64][]model.AttachedMedia, error) {
	result := make(map[int64][]model.AttachedMedia, len(postIDs))
	if len(postIDs) == 0 {
		return result, nil
	}
	start := time.Now()
	var items []struct {
		model.AttachedMedia
		PostID int64 `db:"post_id"`
	}
	err := pgxscan.Select(ctx, s.db, &items, `
		SELECT pwm.post_id, pwm.media_id, m.uid, m.media_name, m.mime_type, m.link, m.author_id, pwm.sort_order
		FROM post_with_media pwm
		JOIN media m ON m.id=pwm.media_id AND m.is_active=TRUE
		WHERE pwm.post_id=ANY($1)
		ORDER BY pwm.post_id, pwm.sort_order
	`, postIDs)
	logQuery(ctx, "postMediaStorage.GetDetailedMediaByPostIDs", start)
	if err != nil && !pgxscan.NotFound(err) {
		return nil, err
	}
	for _, item := range items {
		result[item.PostID] = append(result[item.PostID], item.AttachedMedia)
	}
	return result, nil
}

func (s *postMediaStorage) GetMediaAuthorID(ctx context.Context, mediaID int64) (int64, error) {
	start := time.Now()
	row := s.db.QueryRow(ctx, `SELECT author_id FROM media WHERE id=$1 AND is_active=TRUE`, mediaID)
	logQuery(ctx, "postMediaStorage.GetMediaAuthorID", start)

	var authorID int64
	if err := row.Scan(&authorID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, errors.New("media not found")
		}
		return 0, err
	}
	return authorID, nil
}

func (s *postMediaStorage) Save(ctx context.Context, postWithMedia model.PostWithMedia) error {
	start := time.Now()
	tag, err := s.db.Exec(ctx, `INSERT INTO post_with_media (post_id, media_id, sort_order) VALUES ($1, $2, $3)`, postWithMedia.PostID, postWithMedia.MediaID, postWithMedia.Order)
	logQuery(ctx, "postMediaStorage.Save", start)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errors.New("affected not on 1 row")
	}
	return nil
}

func (s *postMediaStorage) DeleteByPostID(ctx context.Context, postID int64) error {
	start := time.Now()
	_, err := s.db.Exec(ctx, `DELETE FROM post_with_media WHERE post_id=$1`, postID)
	logQuery(ctx, "postMediaStorage.DeleteByPostID", start)
	return err
}

type CommentRepo interface {
	GetCommentCount(ctx context.Context, postID int64) int
	GetTopLevelByPostID(ctx context.Context, postID int64, limit, offset int) ([]model.Comment, error)
	GetReplies(ctx context.Context, postID, parentCommentID int64, limit, offset int) ([]model.Comment, error)
	GetRepliesByParentIDs(ctx context.Context, postID int64, parentCommentIDs []int64, limit, offset int) (map[int64][]model.Comment, error)
	Get(ctx context.Context, id int64) (*model.Comment, error)
	Save(ctx context.Context, comment model.Comment) (int64, error)
	Update(ctx context.Context, comment model.Comment) error
	Delete(ctx context.Context, id int64) error
	GetCommentCountsBatch(ctx context.Context, postIDs []int64) (map[int64]int, error)
}

type commentStorage struct {
	db DB
}

func NewCommentStorage(db DB) CommentRepo {
	return &commentStorage{db: db}
}

func (s *commentStorage) GetCommentCount(ctx context.Context, postID int64) int {
	return countByQuery(ctx, s.db, `SELECT COUNT(*) FROM comment WHERE post_id=$1 AND is_active=TRUE`, "commentStorage.GetCommentCount", postID)
}

func (s *commentStorage) GetTopLevelByPostID(ctx context.Context, postID int64, limit, offset int) ([]model.Comment, error) {
	start := time.Now()
	var comments []model.Comment
	err := pgxscan.Select(ctx, s.db, &comments, `
		SELECT c.*,
		       (
		           SELECT COUNT(*)
		           FROM comment child
		           WHERE child.parent_comment_id=c.id AND child.is_active=TRUE
		       ) AS replies_count
		FROM comment c
		WHERE c.post_id=$1 AND c.parent_comment_id IS NULL AND c.is_active=TRUE
		ORDER BY c.created_at DESC, c.id DESC
		LIMIT $2 OFFSET $3
	`, postID, limit, offset)
	logQuery(ctx, "commentStorage.GetTopLevelByPostID", start)
	if err != nil && !pgxscan.NotFound(err) {
		return nil, err
	}
	return comments, nil
}

func (s *commentStorage) GetReplies(ctx context.Context, postID, parentCommentID int64, limit, offset int) ([]model.Comment, error) {
	start := time.Now()
	var comments []model.Comment
	err := pgxscan.Select(ctx, s.db, &comments, `
		SELECT c.*,
		       (
		           SELECT COUNT(*)
		           FROM comment child
		           WHERE child.parent_comment_id=c.id AND child.is_active=TRUE
		       ) AS replies_count
		FROM comment c
		WHERE c.post_id=$1 AND c.parent_comment_id=$2 AND c.is_active=TRUE
		ORDER BY c.created_at ASC, c.id ASC
		LIMIT $3 OFFSET $4
	`, postID, parentCommentID, limit, offset)
	logQuery(ctx, "commentStorage.GetReplies", start)
	if err != nil && !pgxscan.NotFound(err) {
		return nil, err
	}
	return comments, nil
}

func (s *commentStorage) GetRepliesByParentIDs(ctx context.Context, postID int64, parentCommentIDs []int64, limit, offset int) (map[int64][]model.Comment, error) {
	result := make(map[int64][]model.Comment, len(parentCommentIDs))
	for _, parentID := range parentCommentIDs {
		result[parentID] = []model.Comment{}
	}
	if len(parentCommentIDs) == 0 {
		return result, nil
	}

	start := time.Now()
	var comments []model.Comment
	err := pgxscan.Select(ctx, s.db, &comments, `
		WITH ranked AS (
			SELECT c.*,
			       (
			           SELECT COUNT(*)
			           FROM comment child
			           WHERE child.parent_comment_id=c.id AND child.is_active=TRUE
			       ) AS replies_count,
			       ROW_NUMBER() OVER (PARTITION BY c.parent_comment_id ORDER BY c.created_at ASC, c.id ASC) AS rn
			FROM comment c
			WHERE c.post_id=$1 AND c.parent_comment_id=ANY($2) AND c.is_active=TRUE
		)
		SELECT id, uid, comment_text, post_id, parent_comment_id, sticker_id, author_id,
		       is_active, created_at, updated_at, replies_count
		FROM ranked
		WHERE rn > $3 AND rn <= ($3 + $4)
		ORDER BY parent_comment_id ASC, rn ASC
	`, postID, parentCommentIDs, offset, limit)
	logQuery(ctx, "commentStorage.GetRepliesByParentIDs", start)
	if err != nil && !pgxscan.NotFound(err) {
		return nil, err
	}
	for _, comment := range comments {
		if comment.ParentCommentID == nil {
			continue
		}
		result[*comment.ParentCommentID] = append(result[*comment.ParentCommentID], comment)
	}
	return result, nil
}

func (s *commentStorage) Get(ctx context.Context, id int64) (*model.Comment, error) {
	start := time.Now()
	var comment model.Comment
	err := pgxscan.Get(ctx, s.db, &comment, `
		SELECT c.*,
		       (
		           SELECT COUNT(*)
		           FROM comment child
		           WHERE child.parent_comment_id=c.id AND child.is_active=TRUE
		       ) AS replies_count
		FROM comment c
		WHERE c.id=$1 AND c.is_active=TRUE
	`, id)
	logQuery(ctx, "commentStorage.Get", start)
	if err != nil {
		if pgxscan.NotFound(err) {
			return nil, ErrCommentNotFound
		}
		return nil, err
	}
	return &comment, nil
}

func (s *commentStorage) Save(ctx context.Context, comment model.Comment) (int64, error) {
	start := time.Now()
	row := s.db.QueryRow(ctx, `
		INSERT INTO comment (uid, comment_text, post_id, parent_comment_id, sticker_id, author_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, uuid.New(), comment.Text, comment.PostID, comment.ParentCommentID, comment.StickerID, comment.AuthorID)
	logQuery(ctx, "commentStorage.Save", start)

	var id int64
	if err := row.Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *commentStorage) Update(ctx context.Context, comment model.Comment) error {
	start := time.Now()
	tag, err := s.db.Exec(ctx, `UPDATE comment SET comment_text=$1, updated_at=NOW() WHERE id=$2 AND is_active=TRUE`, comment.Text, comment.ID)
	logQuery(ctx, "commentStorage.Update", start)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrCommentNotFound
	}
	return nil
}

func (s *commentStorage) Delete(ctx context.Context, id int64) error {
	start := time.Now()
	tag, err := s.db.Exec(ctx, `UPDATE comment SET is_active=FALSE, updated_at=NOW() WHERE id=$1 AND is_active=TRUE`, id)
	logQuery(ctx, "commentStorage.Delete", start)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrCommentNotFound
	}
	return nil
}

func (s *commentStorage) GetCommentCountsBatch(ctx context.Context, postIDs []int64) (map[int64]int, error) {
	result := make(map[int64]int, len(postIDs))
	if len(postIDs) == 0 {
		return result, nil
	}
	start := time.Now()
	rows, err := s.db.Query(ctx, `
		SELECT post_id, COUNT(*)::int
		FROM comment
		WHERE post_id=ANY($1) AND is_active=TRUE
		GROUP BY post_id
	`, postIDs)
	logQuery(ctx, "commentStorage.GetCommentCountsBatch", start)
	if err != nil {
		return result, nil
	}
	defer rows.Close()
	for rows.Next() {
		var postID int64
		var count int
		if err := rows.Scan(&postID, &count); err != nil {
			continue
		}
		result[postID] = count
	}
	return result, rows.Err()
}

type LikeRepo interface {
	Save(ctx context.Context, like model.Like) (int64, error)
	GetLikeCountOnPost(ctx context.Context, postID int64) int
	GetPostLikeByAuthor(ctx context.Context, postID, authorID int64) (*model.Like, error)
	SetActive(ctx context.Context, likeID int64, active bool) error
	HasActivePostLike(ctx context.Context, postID, authorID int64) bool
	GetCommentLikeByAuthor(ctx context.Context, commentID, authorID int64) (*model.Like, error)
	GetCommentLikeCountBatch(ctx context.Context, commentIDs []int64) (map[int64]int, error)
	GetCommentViewerLikesBatch(ctx context.Context, commentIDs []int64, authorID int64) (map[int64]bool, error)
	GetPostLikeCountsBatch(ctx context.Context, postIDs []int64) (map[int64]int, error)
	GetViewerPostLikesBatch(ctx context.Context, postIDs []int64, viewerProfileID int64) (map[int64]bool, error)
}

type likeStorage struct {
	db DB
}

func NewLikeStorage(db DB) LikeRepo {
	return &likeStorage{db: db}
}

func (s *likeStorage) Save(ctx context.Context, like model.Like) (int64, error) {
	start := time.Now()
	row := s.db.QueryRow(ctx, `
		INSERT INTO like_record (uid, post_id, comment_id, author_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, uuid.New(), like.PostID, like.CommentID, like.AuthorID)
	logQuery(ctx, "likeStorage.Save", start)

	var likeID int64
	if err := row.Scan(&likeID); err != nil {
		return 0, err
	}
	return likeID, nil
}

func (s *likeStorage) GetLikeCountOnPost(ctx context.Context, postID int64) int {
	return countByQuery(ctx, s.db, `SELECT COUNT(*) FROM like_record WHERE post_id=$1 AND is_active=TRUE`, "likeStorage.GetLikeCountOnPost", postID)
}

func (s *likeStorage) GetPostLikeByAuthor(ctx context.Context, postID, authorID int64) (*model.Like, error) {
	var like model.Like
	start := time.Now()
	err := pgxscan.Get(ctx, s.db, &like, `SELECT * FROM like_record WHERE post_id=$1 AND author_id=$2`, postID, authorID)
	logQuery(ctx, "likeStorage.GetPostLikeByAuthor", start)
	if err != nil {
		return nil, err
	}
	return &like, nil
}

func (s *likeStorage) SetActive(ctx context.Context, likeID int64, active bool) error {
	start := time.Now()
	tag, err := s.db.Exec(ctx, `UPDATE like_record SET is_active=$1, updated_at=NOW() WHERE id=$2`, active, likeID)
	logQuery(ctx, "likeStorage.SetActive", start)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("like not found")
	}
	return nil
}

func (s *likeStorage) HasActivePostLike(ctx context.Context, postID, authorID int64) bool {
	like, err := s.GetPostLikeByAuthor(ctx, postID, authorID)
	return err == nil && like.IsActive
}

func (s *likeStorage) GetCommentLikeByAuthor(ctx context.Context, commentID, authorID int64) (*model.Like, error) {
	start := time.Now()
	var like model.Like
	err := pgxscan.Get(ctx, s.db, &like, `SELECT * FROM like_record WHERE comment_id=$1 AND author_id=$2`, commentID, authorID)
	logQuery(ctx, "likeStorage.GetCommentLikeByAuthor", start)
	if err != nil {
		return nil, err
	}
	return &like, nil
}

func (s *likeStorage) GetCommentLikeCountBatch(ctx context.Context, commentIDs []int64) (map[int64]int, error) {
	result := make(map[int64]int, len(commentIDs))
	if len(commentIDs) == 0 {
		return result, nil
	}
	start := time.Now()
	rows, err := s.db.Query(ctx, `
		SELECT comment_id, COUNT(*)::int
		FROM like_record
		WHERE comment_id=ANY($1) AND is_active=TRUE
		GROUP BY comment_id
	`, commentIDs)
	logQuery(ctx, "likeStorage.GetCommentLikeCountBatch", start)
	if err != nil {
		return result, nil
	}
	defer rows.Close()
	for rows.Next() {
		var commentID int64
		var count int
		if err := rows.Scan(&commentID, &count); err != nil {
			return result, nil
		}
		result[commentID] = count
	}
	return result, rows.Err()
}

func (s *likeStorage) GetCommentViewerLikesBatch(ctx context.Context, commentIDs []int64, authorID int64) (map[int64]bool, error) {
	result := make(map[int64]bool, len(commentIDs))
	if len(commentIDs) == 0 || authorID <= 0 {
		return result, nil
	}
	start := time.Now()
	rows, err := s.db.Query(ctx, `
		SELECT comment_id
		FROM like_record
		WHERE comment_id=ANY($1) AND author_id=$2 AND is_active=TRUE
	`, commentIDs, authorID)
	logQuery(ctx, "likeStorage.GetCommentViewerLikesBatch", start)
	if err != nil {
		return result, nil
	}
	defer rows.Close()
	for rows.Next() {
		var commentID int64
		if err := rows.Scan(&commentID); err != nil {
			return result, nil
		}
		result[commentID] = true
	}
	return result, rows.Err()
}

func (s *likeStorage) GetPostLikeCountsBatch(ctx context.Context, postIDs []int64) (map[int64]int, error) {
	result := make(map[int64]int, len(postIDs))
	if len(postIDs) == 0 {
		return result, nil
	}
	start := time.Now()
	rows, err := s.db.Query(ctx, `
		SELECT post_id, COUNT(*)::int
		FROM like_record
		WHERE post_id=ANY($1) AND is_active=TRUE
		GROUP BY post_id
	`, postIDs)
	logQuery(ctx, "likeStorage.GetPostLikeCountsBatch", start)
	if err != nil {
		return result, nil
	}
	defer rows.Close()
	for rows.Next() {
		var postID int64
		var count int
		if err := rows.Scan(&postID, &count); err != nil {
			continue
		}
		result[postID] = count
	}
	return result, rows.Err()
}

func (s *likeStorage) GetViewerPostLikesBatch(ctx context.Context, postIDs []int64, viewerProfileID int64) (map[int64]bool, error) {
	result := make(map[int64]bool, len(postIDs))
	if len(postIDs) == 0 || viewerProfileID <= 0 {
		return result, nil
	}
	start := time.Now()
	rows, err := s.db.Query(ctx, `
		SELECT post_id
		FROM like_record
		WHERE post_id=ANY($1) AND author_id=$2 AND is_active=TRUE
	`, postIDs, viewerProfileID)
	logQuery(ctx, "likeStorage.GetViewerPostLikesBatch", start)
	if err != nil {
		return result, nil
	}
	defer rows.Close()
	for rows.Next() {
		var postID int64
		if err := rows.Scan(&postID); err != nil {
			continue
		}
		result[postID] = true
	}
	return result, rows.Err()
}

type RepostRepo interface {
	GetRepostCount(ctx context.Context, postID int64) int
	GetRepostCountsBatch(ctx context.Context, postIDs []int64) (map[int64]int, error)
}

type repostStorage struct {
	db DB
}

func NewRepostStorage(db DB) RepostRepo {
	return &repostStorage{db: db}
}

func (s *repostStorage) GetRepostCount(ctx context.Context, postID int64) int {
	return countByQuery(ctx, s.db, `SELECT COUNT(*) FROM repost WHERE post_id=$1 AND is_active=TRUE`, "repostStorage.GetRepostCount", postID)
}

func (s *repostStorage) GetRepostCountsBatch(ctx context.Context, postIDs []int64) (map[int64]int, error) {
	result := make(map[int64]int, len(postIDs))
	if len(postIDs) == 0 {
		return result, nil
	}
	start := time.Now()
	rows, err := s.db.Query(ctx, `
		SELECT post_id, COUNT(*)::int
		FROM repost
		WHERE post_id=ANY($1) AND is_active=TRUE
		GROUP BY post_id
	`, postIDs)
	logQuery(ctx, "repostStorage.GetRepostCountsBatch", start)
	if err != nil {
		return result, nil
	}
	defer rows.Close()
	for rows.Next() {
		var postID int64
		var count int
		if err := rows.Scan(&postID, &count); err != nil {
			continue
		}
		result[postID] = count
	}
	return result, rows.Err()
}

func countByQuery(ctx context.Context, db DB, query string, label string, args ...any) int {
	start := time.Now()
	row := db.QueryRow(ctx, query, args...)
	logQuery(ctx, label, start)

	var count int64
	if err := row.Scan(&count); err != nil {
		return 0
	}
	return int(count)
}

func logQuery(ctx context.Context, query string, start time.Time) {
	logg := logger.FromContext(ctx)
	if logg == nil {
		return
	}
	logg.Debug("db query", zap.String("query", query), zap.Duration("duration_ms", time.Since(start)))
}
