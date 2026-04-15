package auth

//go:generate mockgen -destination=./../mocks/auth_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_ARIS/internal/service/auth AuthService
import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/profile"
	useraccount "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/user_account"
	userprofile "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/user_profile"
	"golang.org/x/crypto/bcrypt"
)

type authService struct {
	userAccountRepo useraccount.UserAccountRepo
	profileRepo     profile.ProfileRepo
	userProfileRepo userprofile.UserProfileRepo
}

type AuthService interface {
	CreateRealUserProfile(ctx context.Context, password_hash, username, firstName, lastName string, email, phone *string, isActive bool, birthdayDate time.Time, gender models.Gender, avatarID *int64) (*models.Profile, error)
	Register(ctx context.Context, firstName, lastName, login, password1, birthday string, gender models.Gender) (*models.Profile, error)
	ValidateRegisterStepOne(ctx context.Context, login, password1, password2 string) (map[string]string, error)
	Login(ctx context.Context, login, password string) (*models.UserAccount, error)
}

func NewAuthService(userAccountRepo useraccount.UserAccountRepo, profileRepo profile.ProfileRepo, userProfileRepo userprofile.UserProfileRepo) AuthService {
	return &authService{
		userAccountRepo: userAccountRepo,
		profileRepo:     profileRepo,
		userProfileRepo: userProfileRepo,
	}
}

func (s *authService) Register(ctx context.Context, firstName, lastName, login, password1, birthday string, gender models.Gender) (*models.Profile, error) {
	login = strings.ToLower(strings.TrimSpace(login))

	if _, err := s.userAccountRepo.GetByUsername(ctx, login); err == nil {
		return nil, errors.New("пользователь с таким login уже существует")
	}

	birthdayDate, err := time.Parse("02/01/2006", birthday)
	if err != nil {
		return nil, errors.New("invalid birthday date")
	}
	if birthdayDate.AddDate(12, 0, 0).After(time.Now()) {
		return nil, errors.New("you are too young, buddy")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password1), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("ошибка при обработке пароля")
	}

	return s.CreateRealUserProfile(ctx, string(hashedPassword), login, firstName, lastName, nil, nil, true, birthdayDate, gender, nil)

	// по хорошему, надо возвращать DTO
	//return profile, nil
}

func (s *authService) Login(ctx context.Context, login, password string) (*models.UserAccount, error) {
	login = strings.ToLower(strings.TrimSpace(login))

	user, err := s.userAccountRepo.GetByUsername(ctx, login)
	if err != nil {
		return nil, errors.New("недействительные учётные данные")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, errors.New("недействительные учётные данные")
	}

	// по хорошему, надо возвращать DTO
	return user, nil
}

func (s *authService) ValidateRegisterStepOne(ctx context.Context, login, password1, password2 string) (map[string]string, error) {
	errorsMap := make(map[string]string)

	login = strings.ToLower(strings.TrimSpace(login))

	if _, err := s.userAccountRepo.GetByUsername(ctx, login); err == nil {
		errorsMap["login"] = "Такой логин уже существует"
	}

	return errorsMap, nil
}

func (s *authService) CreateRealUserProfile(ctx context.Context, password_hash, username, firstName, lastName string, email, phone *string, isActive bool, birthdayDate time.Time, gender models.Gender, avatarID *int64) (*models.Profile, error) {
	userAccount := models.NewUserAccount(username, email, phone, password_hash)
	profile := models.NewProfile(avatarID)

	userAccountID, err := s.userAccountRepo.Save(ctx, *userAccount)
	if err != nil {
		return nil, err
	}

	profileID, err := s.profileRepo.Save(ctx, *profile)
	if err != nil {
		return nil, err
	}

	userProfile := models.NewUserProfile(userAccountID, profileID, firstName, lastName, nil, birthdayDate, gender)

	_, err = s.userProfileRepo.Save(ctx, *userProfile)
	if err != nil {
		return nil, err
	}

	profileWithID, err := s.profileRepo.Get(ctx, profileID)
	if err != nil {
		return nil, err
	}

	return profileWithID, nil
}
