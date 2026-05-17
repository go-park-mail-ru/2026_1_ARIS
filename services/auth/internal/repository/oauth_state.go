package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrOAuthStateNotFound = errors.New("oauth state not found")

type OAuthState struct {
	State        string `json:"state"`
	CodeVerifier string `json:"codeVerifier"`
	RedirectURI  string `json:"redirectUri"`
	ReturnTo     string `json:"returnTo"`
}

type OAuthStateRepo interface {
	Save(ctx context.Context, state OAuthState, ttl time.Duration) error
	Pop(ctx context.Context, state string) (*OAuthState, error)
}

type oauthStateRedis struct {
	client *redis.Client
}

func NewOAuthStateStorage(client *redis.Client) OAuthStateRepo {
	return &oauthStateRedis{client: client}
}

func (r *oauthStateRedis) Save(ctx context.Context, state OAuthState, ttl time.Duration) error {
	marshaled, err := json.Marshal(state)
	if err != nil {
		return err
	}

	return r.client.Set(ctx, oauthStateKey(state.State), marshaled, ttl).Err()
}

func (r *oauthStateRedis) Pop(ctx context.Context, state string) (*OAuthState, error) {
	key := oauthStateKey(state)
	res, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrOAuthStateNotFound
		}
		return nil, err
	}

	_ = r.client.Del(ctx, key).Err()

	var oauthState OAuthState
	if err := json.Unmarshal(res, &oauthState); err != nil {
		return nil, err
	}
	return &oauthState, nil
}

func oauthStateKey(state string) string {
	return "oauth:vkid:state:" + state
}
