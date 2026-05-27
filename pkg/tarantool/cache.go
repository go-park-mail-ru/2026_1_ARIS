package tarantool

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"time"

	tnt "github.com/tarantool/go-tarantool/v2"
)

const spaceFeedSessions = "feed_sessions"

var (
	ErrCacheMiss   = errors.New("tarantool cache miss")
	ErrUnavailable = errors.New("tarantool unavailable")
)

const (
	spaceProfileDetails     = "profile_details"
	spaceProfileSummaries   = "profile_summaries"
	spaceAuthUsers          = "auth_users"
	spaceProfileIDByAccount = "profile_id_by_account"
	spacePostLikeCounts     = "post_like_counts"
	spacePresence           = "presence"
)

type payloadTuple struct {
	//lint:ignore U1000 msgpack asArray marker
	_msgpack  struct{} `msgpack:",asArray"`
	ID        uint64
	Payload   string
	UpdatedAt float64
}

type profileIDTuple struct {
	//lint:ignore U1000 msgpack asArray marker
	_msgpack      struct{} `msgpack:",asArray"`
	UserAccountID uint64
	ProfileID     uint64
	UpdatedAt     float64
}

type likeCountTuple struct {
	//lint:ignore U1000 msgpack asArray marker
	_msgpack  struct{} `msgpack:",asArray"`
	PostID    uint64
	Count     uint64
	UpdatedAt float64
}

type presenceTuple struct {
	//lint:ignore U1000 msgpack asArray marker
	_msgpack      struct{} `msgpack:",asArray"`
	UserAccountID uint64
	IsOnline      bool
	LastSeenAt    float64
	UpdatedAt     float64
	Connections   uint64
}

func (c *Client) GetProfileDetails(ctx context.Context, profileID int64) (*ProfileDetails, error) {
	var details ProfileDetails
	if err := c.getJSON(ctx, spaceProfileDetails, profileID, &details); err != nil {
		return nil, err
	}
	return &details, nil
}

func (c *Client) SetProfileDetails(ctx context.Context, details ProfileDetails) error {
	if details.ProfileID <= 0 {
		return ErrCacheMiss
	}
	return c.setJSON(ctx, spaceProfileDetails, details.ProfileID, details)
}

func (c *Client) DeleteProfileDetails(ctx context.Context, profileID int64) error {
	return c.delete(ctx, spaceProfileDetails, profileID)
}

func (c *Client) GetProfileSummary(ctx context.Context, profileID int64) (*ProfileSummary, error) {
	var summary ProfileSummary
	if err := c.getJSON(ctx, spaceProfileSummaries, profileID, &summary); err != nil {
		return nil, err
	}
	return &summary, nil
}

func (c *Client) SetProfileSummary(ctx context.Context, summary ProfileSummary) error {
	if summary.ProfileID <= 0 {
		return ErrCacheMiss
	}
	return c.setJSON(ctx, spaceProfileSummaries, summary.ProfileID, summary)
}

func (c *Client) DeleteProfileSummary(ctx context.Context, profileID int64) error {
	return c.delete(ctx, spaceProfileSummaries, profileID)
}

func (c *Client) GetAuthUserByAccount(ctx context.Context, userAccountID int64) (*AuthUser, error) {
	var user AuthUser
	if err := c.getJSON(ctx, spaceAuthUsers, userAccountID, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (c *Client) SetAuthUserByAccount(ctx context.Context, user AuthUser) error {
	if user.UserAccountID <= 0 {
		return ErrCacheMiss
	}
	return c.setJSON(ctx, spaceAuthUsers, user.UserAccountID, user)
}

func (c *Client) DeleteAuthUserByAccount(ctx context.Context, userAccountID int64) error {
	return c.delete(ctx, spaceAuthUsers, userAccountID)
}

func (c *Client) GetProfileIDByAccount(ctx context.Context, userAccountID int64) (int64, error) {
	if err := c.ensure(); err != nil {
		return 0, err
	}
	if userAccountID <= 0 {
		return 0, ErrCacheMiss
	}

	reqCtx, cancel := c.withTimeout(ctx)
	defer cancel()

	var rows []profileIDTuple
	err := c.conn.Do(tnt.NewSelectRequest(spaceProfileIDByAccount).
		Index("primary").
		Limit(1).
		Key(uintKey(userAccountID)).
		Context(reqCtx),
	).GetTyped(&rows)
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, ErrCacheMiss
	}
	return int64(rows[0].ProfileID), nil
}

func (c *Client) SetProfileIDByAccount(ctx context.Context, userAccountID, profileID int64) error {
	if err := c.ensure(); err != nil {
		return err
	}
	if userAccountID <= 0 || profileID <= 0 {
		return ErrCacheMiss
	}

	reqCtx, cancel := c.withTimeout(ctx)
	defer cancel()

	_, err := c.conn.Do(tnt.NewReplaceRequest(spaceProfileIDByAccount).
		Tuple([]interface{}{uint64(userAccountID), uint64(profileID), nowSeconds()}).
		Context(reqCtx),
	).Get()
	return err
}

func (c *Client) DeleteProfileIDByAccount(ctx context.Context, userAccountID int64) error {
	return c.delete(ctx, spaceProfileIDByAccount, userAccountID)
}

func (c *Client) GetPostLikeCount(ctx context.Context, postID int64) (int, error) {
	if err := c.ensure(); err != nil {
		return 0, err
	}
	if postID <= 0 {
		return 0, ErrCacheMiss
	}

	reqCtx, cancel := c.withTimeout(ctx)
	defer cancel()

	var rows []likeCountTuple
	err := c.conn.Do(tnt.NewSelectRequest(spacePostLikeCounts).
		Index("primary").
		Limit(1).
		Key(uintKey(postID)).
		Context(reqCtx),
	).GetTyped(&rows)
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, ErrCacheMiss
	}
	return int(rows[0].Count), nil
}

func (c *Client) SetPostLikeCount(ctx context.Context, postID int64, count int) error {
	if err := c.ensure(); err != nil {
		return err
	}
	if postID <= 0 {
		return ErrCacheMiss
	}
	if count < 0 {
		count = 0
	}

	reqCtx, cancel := c.withTimeout(ctx)
	defer cancel()

	_, err := c.conn.Do(tnt.NewReplaceRequest(spacePostLikeCounts).
		Tuple([]interface{}{uint64(postID), uint64(count), nowSeconds()}).
		Context(reqCtx),
	).Get()
	return err
}

func (c *Client) DeletePostLikeCount(ctx context.Context, postID int64) error {
	return c.delete(ctx, spacePostLikeCounts, postID)
}

func (c *Client) SetOnline(ctx context.Context, userAccountID int64) error {
	return c.call(ctx, "presence_online", userAccountID)
}

func (c *Client) SetOffline(ctx context.Context, userAccountID int64) error {
	return c.call(ctx, "presence_offline", userAccountID)
}

func (c *Client) ForceOffline(ctx context.Context, userAccountID int64) error {
	for i := 0; i < 32; i++ {
		if err := c.SetOffline(ctx, userAccountID); err != nil {
			return err
		}
		status, err := c.GetPresence(ctx, userAccountID)
		if err != nil || status == nil || !status.IsOnline || status.Connections == 0 {
			return nil
		}
	}
	return nil
}

func (c *Client) Heartbeat(ctx context.Context, userAccountID int64) error {
	return c.call(ctx, "presence_heartbeat", userAccountID)
}

func (c *Client) GetPresence(ctx context.Context, userAccountID int64) (*PresenceStatus, error) {
	if err := c.ensure(); err != nil {
		return nil, err
	}
	if userAccountID <= 0 {
		return nil, ErrCacheMiss
	}

	reqCtx, cancel := c.withTimeout(ctx)
	defer cancel()

	var rows []presenceTuple
	err := c.conn.Do(tnt.NewSelectRequest(spacePresence).
		Index("primary").
		Limit(1).
		Key(uintKey(userAccountID)).
		Context(reqCtx),
	).GetTyped(&rows)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrCacheMiss
	}
	return rows[0].toStatus(), nil
}

type feedSessionTuple struct {
	//lint:ignore U1000 msgpack asArray marker
	_msgpack  struct{} `msgpack:",asArray"`
	SessionID string
	PostIDs   string
	ExpiresAt float64
}

func (c *Client) SaveFeedSession(ctx context.Context, sessionID string, postIDs []int64, ttl time.Duration) error {
	if err := c.ensure(); err != nil {
		return err
	}
	payload, err := json.Marshal(postIDs)
	if err != nil {
		return err
	}
	expiresAt := float64(time.Now().Add(ttl).UnixNano()) / float64(time.Second)

	reqCtx, cancel := c.withTimeout(ctx)
	defer cancel()

	_, err = c.conn.Do(tnt.NewReplaceRequest(spaceFeedSessions).
		Tuple([]interface{}{sessionID, string(payload), expiresAt}).
		Context(reqCtx),
	).Get()
	return err
}

func (c *Client) GetFeedSession(ctx context.Context, sessionID string) ([]int64, error) {
	if err := c.ensure(); err != nil {
		return nil, err
	}

	reqCtx, cancel := c.withTimeout(ctx)
	defer cancel()

	var rows []feedSessionTuple
	err := c.conn.Do(tnt.NewSelectRequest(spaceFeedSessions).
		Index("primary").
		Limit(1).
		Key(tnt.StringKey{S: sessionID}).
		Context(reqCtx),
	).GetTyped(&rows)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrCacheMiss
	}
	row := rows[0]
	expiresAt := unixFloat(row.ExpiresAt)
	if time.Now().After(expiresAt) {
		_ = c.DeleteFeedSession(ctx, sessionID)
		return nil, ErrCacheMiss
	}
	var postIDs []int64
	if err := json.Unmarshal([]byte(row.PostIDs), &postIDs); err != nil {
		return nil, err
	}
	return postIDs, nil
}

func (c *Client) DeleteFeedSession(ctx context.Context, sessionID string) error {
	if err := c.ensure(); err != nil {
		return err
	}
	reqCtx, cancel := c.withTimeout(ctx)
	defer cancel()

	_, err := c.conn.Do(tnt.NewDeleteRequest(spaceFeedSessions).
		Index("primary").
		Key(tnt.StringKey{S: sessionID}).
		Context(reqCtx),
	).Get()
	return err
}

func (c *Client) getJSON(ctx context.Context, space string, id int64, out interface{}) error {
	if err := c.ensure(); err != nil {
		return err
	}
	if id <= 0 {
		return ErrCacheMiss
	}

	reqCtx, cancel := c.withTimeout(ctx)
	defer cancel()

	var rows []payloadTuple
	err := c.conn.Do(tnt.NewSelectRequest(space).
		Index("primary").
		Limit(1).
		Key(uintKey(id)).
		Context(reqCtx),
	).GetTyped(&rows)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return ErrCacheMiss
	}
	return json.Unmarshal([]byte(rows[0].Payload), out)
}

func (c *Client) setJSON(ctx context.Context, space string, id int64, value interface{}) error {
	if err := c.ensure(); err != nil {
		return err
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}

	reqCtx, cancel := c.withTimeout(ctx)
	defer cancel()

	_, err = c.conn.Do(tnt.NewReplaceRequest(space).
		Tuple([]interface{}{uint64(id), string(payload), nowSeconds()}).
		Context(reqCtx),
	).Get()
	return err
}

func (c *Client) delete(ctx context.Context, space string, id int64) error {
	if err := c.ensure(); err != nil {
		return err
	}
	if id <= 0 {
		return nil
	}

	reqCtx, cancel := c.withTimeout(ctx)
	defer cancel()

	_, err := c.conn.Do(tnt.NewDeleteRequest(space).
		Index("primary").
		Key(uintKey(id)).
		Context(reqCtx),
	).Get()
	return err
}

func (c *Client) call(ctx context.Context, function string, userAccountID int64) error {
	if err := c.ensure(); err != nil {
		return err
	}
	if userAccountID <= 0 {
		return ErrCacheMiss
	}

	reqCtx, cancel := c.withTimeout(ctx)
	defer cancel()

	_, err := c.conn.Do(tnt.NewCallRequest(function).
		Args([]interface{}{uint64(userAccountID)}).
		Context(reqCtx),
	).Get()
	return err
}

func (p presenceTuple) toStatus() *PresenceStatus {
	return &PresenceStatus{
		UserAccountID: int64(p.UserAccountID),
		IsOnline:      p.IsOnline,
		LastSeenAt:    unixFloat(p.LastSeenAt),
		UpdatedAt:     unixFloat(p.UpdatedAt),
		Connections:   p.Connections,
	}
}

func uintKey(id int64) tnt.UintKey {
	return tnt.UintKey{I: uint(id)}
}

func nowSeconds() float64 {
	return float64(time.Now().UnixNano()) / float64(time.Second)
}

func unixFloat(value float64) time.Time {
	sec, frac := math.Modf(value)
	return time.Unix(int64(sec), int64(frac*float64(time.Second))).UTC()
}
