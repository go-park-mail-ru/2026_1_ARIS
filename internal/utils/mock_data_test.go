package utils

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/dto"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	mediarepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/media"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

type seedStore struct {
	mu       sync.Mutex
	nextID   int64
	profiles map[int64]models.Profile
}

func newSeedStore() *seedStore {
	return &seedStore{nextID: 1, profiles: make(map[int64]models.Profile)}
}

func (s *seedStore) id() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.nextID
	s.nextID++
	return id
}

type fakeSeedUserAccounts struct{ store *seedStore }

func (f fakeSeedUserAccounts) Save(_ context.Context, user models.UserAccount) (int64, error) {
	id := f.store.id()
	user.ID = id
	return id, nil
}
func (f fakeSeedUserAccounts) Delete(context.Context, int64) error { return nil }
func (f fakeSeedUserAccounts) Update(context.Context, dto.UpdateUserAccountDTO) error {
	return nil
}
func (f fakeSeedUserAccounts) Get(context.Context, int64) (*models.UserAccount, error) {
	return &models.UserAccount{}, nil
}
func (f fakeSeedUserAccounts) GetByEmail(context.Context, string) (*models.UserAccount, error) {
	return nil, errors.New("not found")
}
func (f fakeSeedUserAccounts) GetByPhone(context.Context, string) (*models.UserAccount, error) {
	return nil, errors.New("not found")
}
func (f fakeSeedUserAccounts) GetByUsername(context.Context, string) (*models.UserAccount, error) {
	return nil, errors.New("not found")
}
func (f fakeSeedUserAccounts) GetByUid(context.Context, uuid.UUID) (*models.UserAccount, error) {
	return nil, errors.New("not found")
}
func (f fakeSeedUserAccounts) List(context.Context, int, int) ([]models.UserAccount, error) {
	return nil, nil
}

type fakeSeedProfiles struct{ store *seedStore }

func (f fakeSeedProfiles) Save(_ context.Context, profile models.Profile) (int64, error) {
	id := f.store.id()
	profile.ID = id
	f.store.profiles[id] = profile
	return id, nil
}
func (f fakeSeedProfiles) Get(_ context.Context, profileID int64) (*models.Profile, error) {
	profile, ok := f.store.profiles[profileID]
	if !ok {
		return nil, errors.New("not found")
	}
	return &profile, nil
}
func (f fakeSeedProfiles) GetAll(context.Context) ([]models.Profile, error) { return nil, nil }
func (f fakeSeedProfiles) GetByUserAccountID(context.Context, int64) (*models.Profile, error) {
	return &models.Profile{}, nil
}
func (f fakeSeedProfiles) UpdateAvatar(context.Context, int64, *int64) error { return nil }

type fakeSeedUserProfiles struct{ store *seedStore }

func (f fakeSeedUserProfiles) Save(_ context.Context, profile models.UserProfile) (int64, error) {
	id := f.store.id()
	profile.ID = id
	return id, nil
}
func (f fakeSeedUserProfiles) Get(context.Context, int64) (*models.UserProfile, error) {
	return nil, errors.New("not found")
}
func (f fakeSeedUserProfiles) GetByProfileID(context.Context, int64) (*models.UserProfile, error) {
	return nil, errors.New("not found")
}
func (f fakeSeedUserProfiles) GetByUserAccountID(context.Context, int64) (*models.UserProfile, error) {
	return nil, errors.New("not found")
}
func (f fakeSeedUserProfiles) Update(context.Context, dto.UpdateUserProfileDTO) error { return nil }

type fakeSeedMedia struct{ store *seedStore }

func (f fakeSeedMedia) Get(context.Context, int64) (*models.Media, error) { return nil, nil }
func (f fakeSeedMedia) Save(_ context.Context, media models.Media) (int64, error) {
	id := f.store.id()
	media.ID = id
	return id, nil
}
func (f fakeSeedMedia) GetLink(context.Context, int64) (string, error)  { return "", nil }
func (f fakeSeedMedia) UpdateLink(context.Context, int64, string) error { return nil }

type fakeSeedPosts struct{ store *seedStore }

func (f fakeSeedPosts) Save(_ context.Context, post models.Post) (int64, error) {
	id := f.store.id()
	post.ID = id
	return id, nil
}
func (f fakeSeedPosts) Delete(context.Context, int64) error       { return nil }
func (f fakeSeedPosts) Update(context.Context, models.Post) error { return nil }
func (f fakeSeedPosts) GetByAuthorID(context.Context, int64) ([]models.Post, error) {
	return nil, nil
}
func (f fakeSeedPosts) GetByCommunityID(context.Context, int64) ([]models.Post, error) {
	return nil, nil
}
func (f fakeSeedPosts) List(context.Context, int, int) ([]models.Post, error) { return nil, nil }
func (f fakeSeedPosts) Get(context.Context, int64) (*models.Post, error)      { return nil, nil }
func (f fakeSeedPosts) GetAll(context.Context) ([]models.Post, error)         { return nil, nil }

type fakeSeedPostWithMedia struct{}

func (fakeSeedPostWithMedia) GetMediaByPostID(context.Context, int64) []int64  { return nil }
func (fakeSeedPostWithMedia) Save(context.Context, models.PostWithMedia) error { return nil }
func (fakeSeedPostWithMedia) DeleteByPostID(context.Context, int64) error      { return nil }

type fakeSeedComments struct{ store *seedStore }

func (f fakeSeedComments) GetCommentCount(context.Context, int64) int { return 0 }
func (f fakeSeedComments) Save(context.Context, models.Comment) (int64, error) {
	return f.store.id(), nil
}

type fakeSeedReposts struct{ store *seedStore }

func (f fakeSeedReposts) Save(context.Context, models.Repost) (int64, error) {
	return f.store.id(), nil
}
func (f fakeSeedReposts) GetRepostCount(context.Context, int64) int { return 0 }

type fakeSeedChats struct{ store *seedStore }

func (f fakeSeedChats) Save(_ context.Context, chat *models.Chat) error {
	chat.ID = f.store.id()
	return nil
}
func (f fakeSeedChats) GetByID(context.Context, int64) (*models.Chat, error) { return nil, nil }
func (f fakeSeedChats) Delete(context.Context, int64) error                  { return nil }

type fakeSeedLikes struct{ store *seedStore }

func (f fakeSeedLikes) Get(context.Context, int64) (*models.Like, error) { return nil, nil }
func (f fakeSeedLikes) Save(context.Context, models.Like) (int64, error) { return f.store.id(), nil }
func (f fakeSeedLikes) GetLikeCountOnPost(context.Context, int64) int    { return 0 }
func (f fakeSeedLikes) GetPostLikeByAuthor(context.Context, int64, int64) (*models.Like, error) {
	return nil, errors.New("not found")
}
func (f fakeSeedLikes) SetActive(context.Context, int64, bool) error         { return nil }
func (f fakeSeedLikes) HasActivePostLike(context.Context, int64, int64) bool { return false }

type fakeSeedS3 struct{}

func (fakeSeedS3) Save(_ context.Context, bucketName string, reader io.Reader, mediaUUID uuid.UUID, size int64, extension string, opts minio.PutObjectOptions) (string, error) {
	if bucketName == "" || mediaUUID == uuid.Nil || size <= 0 || extension == "" || opts.ContentType == "" {
		return "", errors.New("invalid upload")
	}
	if _, err := io.Copy(io.Discard, reader); err != nil {
		return "", err
	}
	return "/bucket/" + mediaUUID.String() + extension, nil
}

func TestMakeMockSeedsDemoDataWithoutNetwork(t *testing.T) {
	png, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/p9sAAAAASUVORK5CYII=")
	require.NoError(t, err)

	oldTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     http.Header{"Content-Type": []string{"image/png"}},
			Body:       io.NopCloser(strings.NewReader(string(png))),
			Request:    r,
		}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = oldTransport })

	store := newSeedStore()
	MakeMock(
		fakeSeedMedia{store: store},
		fakeSeedUserAccounts{store: store},
		fakeSeedProfiles{store: store},
		fakeSeedUserProfiles{store: store},
		fakeSeedPosts{store: store},
		fakeSeedPostWithMedia{},
		fakeSeedComments{store: store},
		fakeSeedReposts{store: store},
		fakeSeedChats{store: store},
		fakeSeedLikes{store: store},
		fakeSeedS3{},
		"bucket",
	)

	require.Greater(t, store.nextID, int64(70))
}

func TestSeedMediaDownloadValidation(t *testing.T) {
	_, err := newSeedMediaFromURL(context.Background(), nil, "bucket", "name", nil, "http://example.test/a.png", 1)
	require.ErrorContains(t, err, "s3 repo is nil")

	_, err = newSeedMediaFromURL(context.Background(), fakeSeedS3{}, "", "name", nil, "http://example.test/a.png", 1)
	require.ErrorContains(t, err, "minio bucket name is empty")

	_, _, _, err = downloadSeedImage(context.Background(), "://bad")
	require.Error(t, err)

	oldTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Status:     "500 Internal Server Error",
			Body:       io.NopCloser(strings.NewReader("nope")),
			Request:    r,
		}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = oldTransport })

	_, _, _, err = downloadSeedImage(context.Background(), "http://example.test/a.png")
	require.ErrorContains(t, err, "unexpected status")
}

var _ mediarepo.S3Repo = fakeSeedS3{}
