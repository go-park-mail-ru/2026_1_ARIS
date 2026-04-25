package authservice

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/profile"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/session"
	useraccount "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/user_account"
	userprofile "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/user_profile"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	Login(ctx context.Context, dto LoginServiceRequestDTO) (*LoginServiceResultDTO, error)
}

type authService struct {
	userAccountRepo useraccount.UserAccountRepo
	profileRepo     profile.ProfileRepo
	userProfileRepo userprofile.UserProfileRepo
	sessionRepo     session.SessionRepo
}

func NewAuthService(userAccountRepo useraccount.UserAccountRepo, profileRepo profile.ProfileRepo, userProfileRepo userprofile.UserProfileRepo, sessionRepo session.SessionRepo) AuthService {
	return &authService{
		userAccountRepo: userAccountRepo,
		profileRepo:     profileRepo,
		userProfileRepo: userProfileRepo,
		sessionRepo:     sessionRepo,
	}
}

func (s *authService) Login(ctx context.Context, dto LoginServiceRequestDTO) (*LoginServiceResultDTO, error) {
	dto.Login = strings.ToLower(strings.TrimSpace(dto.Login))

	userAccount, err := s.userAccountRepo.GetByUsername(ctx, dto.Login)
	if err != nil {
		return nil, errors.New("недействительные учётные данные")
	}

	err = bcrypt.CompareHashAndPassword([]byte(userAccount.PasswordHash), []byte(dto.Password))
	if err != nil {
		return nil, errors.New("недействительные учётные данные")
	}

	userProfile, err := s.userProfileRepo.GetByUserAccountID(ctx, userAccount.ID)
	if err != nil {
		return nil, err
	}

	profile, err := s.profileRepo.Get(ctx, userProfile.ProfileID)
	if err != nil {
		return nil, err
	}

	expiredAt := time.Now().Add(sessionTTL)

	session := models.Session{
		SessionID: models.SessionID(uuid.New().String()),
		UserID:    userAccount.ID,
		CreatedAt: time.Now(),
		ExpiredAt: expiredAt,
	}

	if err := s.sessionRepo.Save(ctx, session); err != nil {
		return nil, err
	}
	profile.

	return &LoginServiceResultDTO{
		UserAccountID: userAccount.ID,
		FirstName:     userProfile.FirstName,
		LastName:      userProfile.LastName,
		SessionID:     string(session.SessionID),
		ExpiresAt:     expiredAt,
		ProfileID:     profile.ID,
		AvatarLink:    "/////////////////////////////////////////////////////////////",
	}, nil
}
