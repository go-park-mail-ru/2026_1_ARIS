package user

//go:generate mockgen -destination=./../mocks/user_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_ARIS/internal/service/user UserService
import (
	"context"
	"math/rand"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/profile"
	useraccount "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/user_account"
	userprofile "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/user_profile"
	"github.com/google/uuid"
)

const maxSuggestedUsers = 4

type userService struct {
	UserAccountRepo useraccount.UserAccountRepo
	ProfileRepo     profile.ProfileRepo
	UserProfileRepo userprofile.UserProfileRepo
}

type UserService interface {
	CreateRealUserProfile(ctx context.Context, email, phone *string, passwordHash, username, firstName, lastName string, birthdayDate time.Time, gender models.Gender, avatarID *int64) (*models.Profile, error)
	GetUserList(ctx context.Context, offset, limit int) []models.UserAccount
	GetUserProfileByProfileID(ctx context.Context, profileID int64) (*models.UserProfile, error)
	GetUserProfileByUserProfileID(ctx context.Context, userProfileID int64) (*models.UserProfile, error)
	GetUserProfileByUserID(ctx context.Context, userID int64) (*models.UserProfile, error)
	GetUserProfileByUserAccountUid(ctx context.Context, userAccountUid uuid.UUID) (*models.UserProfile, error)
	GetUserAccountByUserProfileID(ctx context.Context, userProfileID int64) (*models.UserAccount, error)
	GetUserAccountByProfileID(ctx context.Context, profileID int64) (*models.UserAccount, error)
	GetSuggestedUsers(ctx context.Context, currentUserID int64) ([]models.Profile, error)
	GetPublicPopularUsers(ctx context.Context) ([]models.Profile, error)
	GetLatestEvents(ctx context.Context) ([]LatestEvent, error)
	GetUserAccountByUserAccountUid(ctx context.Context, userAccountUid uuid.UUID) (*models.UserAccount, error)
}

type LatestEvent struct {
	Profile models.Profile
	Type    int
}

func NewUserService(userAccountRepo useraccount.UserAccountRepo, profileRepo profile.ProfileRepo, userProfileRepo userprofile.UserProfileRepo) UserService {
	return &userService{
		UserAccountRepo: userAccountRepo,
		ProfileRepo:     profileRepo,
		UserProfileRepo: userProfileRepo,
	}
}

func (s *userService) CreateRealUserProfile(ctx context.Context, email, phone *string, passwordHash, username, firstName, lastName string, birthdayDate time.Time, gender models.Gender, avatarID *int64) (*models.Profile, error) {
	userAccount := models.NewUserAccount(username, email, phone, passwordHash)
	profile := models.NewProfile(avatarID)
	userProfile := models.NewUserProfile(userAccount.ID, profile.ID, firstName, lastName, nil, birthdayDate, gender)

	_, err := s.UserAccountRepo.Save(ctx, *userAccount)
	if err != nil {
		return nil, err
	}

	_, err = s.ProfileRepo.Save(ctx, *profile)
	if err != nil {
		return nil, err
	}

	_, err = s.UserProfileRepo.Save(ctx, *userProfile)
	if err != nil {
		return nil, err
	}

	return profile, nil
}

func (s *userService) GetUserList(ctx context.Context, offset, limit int) []models.UserAccount {
	return s.UserAccountRepo.List(ctx, offset, limit)
}

func (s *userService) GetUserProfileByProfileID(ctx context.Context, profileID int64) (*models.UserProfile, error) {
	return s.UserProfileRepo.GetByProfileID(ctx, profileID)
}

func (s *userService) GetUserProfileByUserProfileID(ctx context.Context, userProfileID int64) (*models.UserProfile, error) {
	return s.UserProfileRepo.Get(ctx, userProfileID)
}

func (s *userService) GetUserProfileByUserID(ctx context.Context, userID int64) (*models.UserProfile, error) {
	return s.UserProfileRepo.GetByUserAccountID(ctx, userID)
}

func (s *userService) GetUserAccountByUserProfileID(ctx context.Context, userProfileID int64) (*models.UserAccount, error) {
	userProfile, err := s.UserProfileRepo.Get(ctx, userProfileID)
	if err != nil {
		return nil, err
	}

	userAccount, err := s.UserAccountRepo.Get(ctx, userProfile.UserAccountID)
	if err != nil {
		return nil, err
	}

	return userAccount, nil
}

func (s *userService) GetUserAccountByProfileID(ctx context.Context, profileID int64) (*models.UserAccount, error) {
	userProfile, err := s.UserProfileRepo.GetByProfileID(ctx, profileID)
	if err != nil {
		return nil, err
	}

	userAccount, err := s.UserAccountRepo.Get(ctx, userProfile.UserAccountID)
	if err != nil {
		return nil, err
	}

	return userAccount, nil
}

// возвращает []models.Profile рекомендованных пользователей
func (s *userService) GetSuggestedUsers(ctx context.Context, currentUserAccountID int64) ([]models.Profile, error) {
	profiles, err := s.ProfileRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	currentUserProfile, err := s.UserProfileRepo.GetByUserAccountID(ctx, currentUserAccountID)
	if err != nil {
		return nil, err
	}

	currentProfileID := currentUserProfile.ProfileID
	filtered := make([]models.Profile, 0, len(profiles))

	for _, profile := range profiles {
		if profile.ID == currentProfileID {
			continue
		}
		userAccount, err := s.GetUserAccountByProfileID(ctx, profile.ID)
		if err != nil {
			continue
		}

		if userAccount.Username == "KomandaARIS" {
			continue
		}

		filtered = append(filtered, profile)
	}

	if len(filtered) <= maxSuggestedUsers {
		rand.Shuffle(len(filtered), func(i, j int) {
			filtered[i], filtered[j] = filtered[j], filtered[i]
		})
		return filtered, nil
	}

	rand.Shuffle(len(filtered), func(i, j int) {
		filtered[i], filtered[j] = filtered[j], filtered[i]
	})

	return filtered[:maxSuggestedUsers], nil
}

// возвращает []models.Profile по заранее подготовленным username'ам, если такие пользователи существуют в базе
func (s *userService) GetPublicPopularUsers(ctx context.Context) ([]models.Profile, error) {
	allProfiles, err := s.ProfileRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	targetUsernames := []string{
		"SergeyShulginenko",
		"AnnaOparina",
		"IvanKhvostov",
		"RinatBaikov",
	}

	profilesByUsername := make(map[string]models.Profile)
	for _, profile := range allProfiles {
		userAccount, err := s.GetUserAccountByProfileID(ctx, profile.ID)
		if err != nil {
			continue
		}
		profilesByUsername[userAccount.Username] = profile
	}

	result := make([]models.Profile, 0, len(targetUsernames))
	for _, username := range targetUsernames {
		profile, ok := profilesByUsername[username]
		if ok {
			result = append(result, profile)
		}
	}

	return result, nil
}

// возвращает "события" по заранее подготовленным сценариям по username'ам, если такие пользователи существуют в базе
func (s *userService) GetLatestEvents(ctx context.Context) ([]LatestEvent, error) {
	allProfiles, err := s.ProfileRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	targets := []struct {
		Username string
		Type     int
	}{
		{Username: "SofiaSitnichenko", Type: 1},
		{Username: "DaniilKhasyanov", Type: 2},
		{Username: "KonstantinGalanin", Type: 3},
	}

	profilesByUsername := make(map[string]models.Profile)
	for _, profile := range allProfiles {
		userAccount, err := s.GetUserAccountByProfileID(ctx, profile.ID)
		if err != nil {
			continue
		}
		profilesByUsername[userAccount.Username] = profile
	}

	result := make([]LatestEvent, 0, len(targets))
	for _, target := range targets {
		profile, ok := profilesByUsername[target.Username]
		if ok {
			result = append(result, LatestEvent{
				Profile: profile,
				Type:    target.Type,
			})
		}
	}

	return result, nil
}

func (s *userService) GetUserProfileByUserAccountUid(ctx context.Context, userAccountUid uuid.UUID) (*models.UserProfile, error) {
	userAccount, err := s.UserAccountRepo.GetByUid(ctx, userAccountUid)
	if err != nil {
		return nil, err
	}

	userProfile, err := s.UserProfileRepo.GetByUserAccountID(ctx, userAccount.ID)
	if err != nil {
		return nil, err
	}

	return userProfile, nil
}

func (s *userService) GetUserAccountByUserAccountUid(ctx context.Context, userAccountUid uuid.UUID) (*models.UserAccount, error) {
	return s.UserAccountRepo.GetByUid(ctx, userAccountUid)
}
