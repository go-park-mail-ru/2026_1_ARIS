package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	mediapb "github.com/go-park-mail-ru/2026_1_ARIS/proto/media"
	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/auth/internal/model"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/auth/internal/repository"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const SessionTTL = 24 * time.Hour

var (
	ErrInvalidInput       = errors.New("invalid input")
	ErrInvalidBirthday    = errors.New("invalid birthday")
	ErrTooYoung           = errors.New("user must be at least 12 years old")
	ErrLoginAlreadyExists = errors.New("login already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrSessionNotFound    = errors.New("session not found")
	ErrOAuthUnavailable   = errors.New("oauth provider unavailable")
	ErrOAuthProvider      = errors.New("oauth provider error")
	ErrPasswordMismatch   = errors.New("passwords do not match")
	ErrPasswordReuse      = errors.New("new password must differ from old password")
	ErrWeakPassword       = errors.New("password must be 7-20 characters")
)

type VKIDClient interface {
	ExchangeCode(ctx context.Context, in VKIDCallbackInput) (*VKIDUser, error)
}

type VKIDUser struct {
	ID        string
	FirstName string
	LastName  string
	Email     *string
	Gender    userpb.Gender
}

type Option func(*Service)

type Service struct {
	sessions    repository.SessionRepo
	users       userpb.UserServiceClient
	mediaClient mediapb.MediaServiceClient
	vkid        VKIDClient
	now         func() time.Time
}

func New(sessions repository.SessionRepo, users userpb.UserServiceClient, mediaClient mediapb.MediaServiceClient, opts ...Option) *Service {
	service := &Service{sessions: sessions, users: users, mediaClient: mediaClient, now: time.Now}
	for _, opt := range opts {
		opt(service)
	}
	return service
}

func WithVKIDClient(client VKIDClient) Option {
	return func(s *Service) {
		s.vkid = client
	}
}

func (s *Service) RegisterStepOne(ctx context.Context, in RegisterStepOneInput) error {
	available, err := s.usernameAvailable(ctx, normalizeLogin(in.Login))
	if err != nil {
		return err
	}
	if !available {
		return ErrLoginAlreadyExists
	}
	return nil
}

func (s *Service) Register(ctx context.Context, in RegisterInput) (*AuthResult, error) {
	login := normalizeLogin(in.Login)
	if in.FirstName == "" || in.LastName == "" || login == "" || in.Password1 == "" {
		return nil, ErrInvalidInput
	}
	if in.Password1 != in.Password2 {
		return nil, ErrInvalidInput
	}

	available, err := s.usernameAvailable(ctx, login)
	if err != nil {
		return nil, err
	}
	if !available {
		return nil, ErrLoginAlreadyExists
	}

	birthday, err := parseBirthday(in.Birthday)
	if err != nil {
		return nil, ErrInvalidBirthday
	}
	if birthday.AddDate(12, 0, 0).After(s.now()) {
		return nil, ErrTooYoung
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password1), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user, err := s.users.CreateAuthUser(ctx, &userpb.CreateAuthUserRequest{
		Username:     login,
		PasswordHash: string(hash),
		FirstName:    in.FirstName,
		LastName:     in.LastName,
		Birthday:     birthday.Format(time.DateOnly),
		Gender:       toProtoGender(in.Gender),
	})
	if err != nil {
		return nil, normalizeUserError(err)
	}

	return s.issueAuthResult(ctx, user.GetUserAccountId())
}

func (s *Service) Login(ctx context.Context, in LoginInput) (*AuthResult, error) {
	credentials, err := s.users.GetCredentialsByLogin(ctx, &userpb.GetCredentialsByLoginRequest{
		Login: normalizeLogin(in.Login),
	})
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	if bcrypt.CompareHashAndPassword([]byte(credentials.GetPasswordHash()), []byte(in.Password)) != nil {
		return nil, ErrInvalidCredentials
	}

	return s.issueAuthResult(ctx, credentials.GetUserAccountId())
}

func (s *Service) ChangePassword(ctx context.Context, sessionID string, in ChangePasswordInput) error {
	session, err := s.ValidateSession(ctx, sessionID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(in.OldPassword) == "" || in.NewPassword1 == "" || in.NewPassword2 == "" {
		return ErrInvalidInput
	}
	if in.NewPassword1 != in.NewPassword2 {
		return ErrPasswordMismatch
	}
	if len(in.NewPassword1) < 7 || len(in.NewPassword1) > 20 {
		return ErrWeakPassword
	}

	user, err := s.userByAccountID(ctx, session.UserID)
	if err != nil {
		return err
	}
	credentials, err := s.users.GetCredentialsByLogin(ctx, &userpb.GetCredentialsByLoginRequest{
		Login: normalizeLogin(user.Login),
	})
	if err != nil || credentials.GetUserAccountId() != session.UserID {
		return ErrInvalidCredentials
	}
	currentHash := []byte(credentials.GetPasswordHash())
	if bcrypt.CompareHashAndPassword(currentHash, []byte(in.OldPassword)) != nil {
		return ErrInvalidCredentials
	}
	if bcrypt.CompareHashAndPassword(currentHash, []byte(in.NewPassword1)) == nil {
		return ErrPasswordReuse
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.NewPassword1), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if _, err := s.users.UpdatePasswordHash(ctx, &userpb.UpdatePasswordHashRequest{
		UserAccountId: session.UserID,
		PasswordHash:  string(hash),
	}); err != nil {
		return normalizeUserError(err)
	}
	return s.Logout(ctx, sessionID)
}

func (s *Service) LoginWithVKID(ctx context.Context, in VKIDCallbackInput) (*AuthResult, error) {
	if s.vkid == nil {
		return nil, ErrOAuthUnavailable
	}
	if strings.TrimSpace(in.Code) == "" || strings.TrimSpace(in.CodeVerifier) == "" || strings.TrimSpace(in.RedirectURI) == "" {
		return nil, ErrInvalidInput
	}

	vkUser, err := s.vkid.ExchangeCode(ctx, in)
	if err != nil {
		return nil, err
	}
	if vkUser == nil || strings.TrimSpace(vkUser.ID) == "" {
		return nil, ErrOAuthProvider
	}

	user, err := s.users.GetOrCreateOAuthUser(ctx, &userpb.GetOrCreateOAuthUserRequest{
		Provider:       "vkid",
		ProviderUserId: vkUser.ID,
		Username:       "vk" + vkUser.ID,
		Email:          vkUser.Email,
		FirstName:      vkUser.FirstName,
		LastName:       vkUser.LastName,
		Birthday:       "1970-01-01",
		Gender:         vkUser.Gender,
	})
	if err != nil {
		return nil, normalizeUserError(err)
	}

	return s.issueAuthResult(ctx, user.GetUserAccountId())
}

func (s *Service) ValidateSession(ctx context.Context, sessionID string) (*model.Session, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, ErrSessionNotFound
	}

	session, err := s.sessions.GetByID(ctx, model.SessionID(sessionID))
	if err != nil {
		return nil, ErrSessionNotFound
	}
	if session.ExpiredAt.Before(s.now()) {
		_ = s.sessions.Delete(ctx, session.SessionID)
		return nil, ErrSessionNotFound
	}
	return session, nil
}

func (s *Service) GetMe(ctx context.Context, sessionID string) (*User, error) {
	session, err := s.ValidateSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	return s.userByAccountID(ctx, session.UserID)
}

func (s *Service) Logout(ctx context.Context, sessionID string) error {
	if strings.TrimSpace(sessionID) == "" {
		return nil
	}
	return s.sessions.Delete(ctx, model.SessionID(sessionID))
}

func (s *Service) issueAuthResult(ctx context.Context, accountID int64) (*AuthResult, error) {
	session := model.Session{
		SessionID: model.SessionID(uuid.NewString()),
		UserID:    accountID,
		CreatedAt: s.now(),
		ExpiredAt: s.now().Add(SessionTTL),
	}
	if err := s.sessions.Save(ctx, session); err != nil {
		return nil, err
	}

	user, err := s.userByAccountID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return &AuthResult{User: *user, Session: session}, nil
}

func (s *Service) userByAccountID(ctx context.Context, accountID int64) (*User, error) {
	resp, err := s.users.GetAuthUserByAccount(ctx, &userpb.GetAuthUserByAccountRequest{UserAccountId: accountID})
	if err != nil {
		return nil, normalizeUserError(err)
	}

	return &User{
		UserAccountID: resp.GetUserAccountId(),
		UserProfileID: resp.GetUserProfileId(),
		ProfileID:     resp.GetProfileId(),
		Login:         resp.GetLogin(),
		Email:         resp.Email,
		FirstName:     resp.GetFirstName(),
		LastName:      resp.GetLastName(),
		AvatarURL:     s.avatarURL(ctx, resp.AvatarId),
		CreatedAt:     parseRFC3339(resp.GetCreatedAt()),
	}, nil
}

func (s *Service) avatarURL(ctx context.Context, avatarID *int64) *string {
	if avatarID == nil || *avatarID <= 0 || s.mediaClient == nil {
		return nil
	}

	callCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	resp, err := s.mediaClient.GetMediaURL(callCtx, &mediapb.GetMediaURLRequest{MediaId: *avatarID})
	if err != nil || resp == nil || strings.TrimSpace(resp.GetUrl()) == "" {
		return nil
	}
	url := resp.GetUrl()
	return &url
}

func (s *Service) usernameAvailable(ctx context.Context, login string) (bool, error) {
	if login == "" || s.users == nil {
		return false, ErrInvalidInput
	}
	resp, err := s.users.CheckUsernameAvailable(ctx, &userpb.CheckUsernameAvailableRequest{Username: login})
	if err != nil {
		return false, normalizeUserError(err)
	}
	return resp.GetAvailable(), nil
}

func normalizeLogin(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeGender(value model.Gender) model.Gender {
	if value == model.Male {
		return model.Male
	}
	return model.Female
}

func toProtoGender(value model.Gender) userpb.Gender {
	if normalizeGender(value) == model.Male {
		return userpb.Gender_GENDER_MALE
	}
	return userpb.Gender_GENDER_FEMALE
}

func parseBirthday(value string) (time.Time, error) {
	if parsed, err := time.Parse(time.DateOnly, value); err == nil {
		return parsed, nil
	}
	return time.Parse("02/01/2006", value)
}

func parseRFC3339(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func normalizeUserError(err error) error {
	switch status.Code(err) {
	case codes.AlreadyExists:
		return ErrLoginAlreadyExists
	case codes.InvalidArgument:
		return ErrInvalidInput
	case codes.NotFound, codes.Unauthenticated:
		return ErrInvalidCredentials
	default:
		return err
	}
}
