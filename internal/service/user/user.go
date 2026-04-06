package user

//go:generate mockgen -destination=./../mocks/user_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_ARIS/internal/service/user UserService
import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/dto"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/profile"
	useraccount "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/user_account"
	userprofile "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/user_profile"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const maxSuggestedUsers = 4

func normalizeUsername(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

type userService struct {
	UserAccountRepo useraccount.UserAccountRepo
	ProfileRepo     profile.ProfileRepo
	UserProfileRepo userprofile.UserProfileRepo
}

type UserService interface {
	CreateRealUserProfile(ctx context.Context, email, phone *string, password, username, firstName, lastName string, birthdayDate time.Time, gender models.Gender, avatarID *int64) (*models.Profile, error)
	GetUserList(ctx context.Context, offset, limit int) []models.UserAccount
	GetUserProfileByProfileID(ctx context.Context, profileID int64) (*models.UserProfile, error)
	GetUserProfileByUserProfileID(ctx context.Context, userProfileID int64) (*models.UserProfile, error)
	GetUserProfileByUserID(ctx context.Context, userID int64) (*models.UserProfile, error)
	GetUserProfileByUserAccountUid(ctx context.Context, userAccountUid uuid.UUID) (*models.UserProfile, error)
	GetUserProfileByUserAccountID(ctx context.Context, userAccountID int64) (*models.UserProfile, error)
	GetUserAccountByUserProfileID(ctx context.Context, userProfileID int64) (*models.UserAccount, error)
	GetUserAccountByProfileID(ctx context.Context, profileID int64) (*models.UserAccount, error)
	GetSuggestedUsers(ctx context.Context, currentUserID int64) ([]models.Profile, error)
	GetPublicPopularUsers(ctx context.Context) ([]models.Profile, error)
	GetLatestEvents(ctx context.Context) ([]LatestEvent, error)
	GetUserAccountByUserAccountUid(ctx context.Context, userAccountUid uuid.UUID) (*models.UserAccount, error)
	GetProfileByUserProfileID(ctx context.Context, userProfileID int64) (*models.Profile, error)
	GetProfileByProfileID(ctx context.Context, profileID int64) (*models.Profile, error)
	GetProfileByUserAccountID(ctx context.Context, userAccountID int64) (*models.Profile, error)
	UpdateMe(ctx context.Context, updateDTO dto.UpdateFullProfileRequestDTO) error
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

func (s *userService) UpdateMe(ctx context.Context, updateDTO dto.UpdateFullProfileRequestDTO) error {

	if updateDTO.Username != nil {
		login := strings.ToLower(*updateDTO.Username)
		updateDTO.Username = &login
	}

	userAccountDTO := dto.UpdateUserAccountDTO{
		ID:       updateDTO.UserAccountID,
		Username: updateDTO.Username,
		Email:    updateDTO.Email,
		Phone:    updateDTO.Phone,
	}

	var date *time.Time
	if updateDTO.BirthdayDate != nil {
		d, err := time.Parse("2006-01-02", *updateDTO.BirthdayDate)
		if err == nil {
			date = &d
		}
	}

	userProfileDTO := dto.UpdateUserProfileDTO{
		ID:           updateDTO.UserProfileID,
		FirstName:    updateDTO.FirstName,
		LastName:     updateDTO.LastName,
		Bio:          updateDTO.Bio,
		BirthdayDate: date,
		Gender:       updateDTO.Gender,
		NativeTown:   updateDTO.NativeTown,
		Town:         updateDTO.Town,
		Institution:  updateDTO.Institution,
		Group:        updateDTO.Group,
		Company:      updateDTO.Company,
		JobTitle:     updateDTO.JobTitle,
		Interests:    updateDTO.Interests,
		FavMusic:     updateDTO.FavMusic,
	}

	// Если данные изменились - есть смысл сохранять
	// email phone username - unique
	if userAccountDTO.HasUpdates() {
		fmt.Println("Изменили userAccount")
		err := s.UserAccountRepo.Update(ctx, userAccountDTO)
		if err != nil {
			return err
		}
	}

	// Если данные изменились - есть смысл сохранять
	if userProfileDTO.HasUpdates() {
		fmt.Println("Изменили userProfile")
		err := s.UserProfileRepo.Update(ctx, userProfileDTO)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *userService) CreateRealUserProfile(ctx context.Context, email, phone *string, password, username, firstName, lastName string, birthdayDate time.Time, gender models.Gender, avatarID *int64) (*models.Profile, error) {
	username = strings.ToLower(strings.TrimSpace(username))

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("ошибка при обработке пароля")
	}

	userAccount := models.NewUserAccount(username, email, phone, string(hashedPassword))
	profile := models.NewProfile(avatarID)

	userAccountID, err := s.UserAccountRepo.Save(ctx, *userAccount)
	if err != nil {
		return nil, err
	}

	profileID, err := s.ProfileRepo.Save(ctx, *profile)
	if err != nil {
		return nil, err
	}

	userProfile := models.NewUserProfile(userAccountID, profileID, firstName, lastName, nil, birthdayDate, gender)

	_, err = s.UserProfileRepo.Save(ctx, *userProfile)
	if err != nil {
		return nil, err
	}

	profileWithID, err := s.ProfileRepo.Get(ctx, profileID)
	if err != nil {
		return nil, err
	}

	return profileWithID, nil
}

func (s *userService) GetUserList(ctx context.Context, offset, limit int) []models.UserAccount {
	list, _ := s.UserAccountRepo.List(ctx, offset, limit)
	return list
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

		if normalizeUsername(userAccount.Username) == "komandaaris" {
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
		"sergeyshulginenko",
		"annaoparina",
		"ivankhvostov",
		"rinatbaikov",
	}

	profilesByUsername := make(map[string]models.Profile)
	for _, profile := range allProfiles {
		userAccount, err := s.GetUserAccountByProfileID(ctx, profile.ID)
		if err != nil {
			continue
		}
		profilesByUsername[normalizeUsername(userAccount.Username)] = profile
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
		{Username: "sofiasitnichenko", Type: 1},
		{Username: "daniilkhasyanov", Type: 2},
		{Username: "konstantingalanin", Type: 3},
	}

	profilesByUsername := make(map[string]models.Profile)
	for _, profile := range allProfiles {
		userAccount, err := s.GetUserAccountByProfileID(ctx, profile.ID)
		if err != nil {
			continue
		}
		profilesByUsername[normalizeUsername(userAccount.Username)] = profile
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

func (s *userService) GetUserProfileByUserAccountID(ctx context.Context, userAccountID int64) (*models.UserProfile, error) {
	// userAccount, err := s.UserAccountRepo.Get(ctx, userAccountID)
	// if err != nil {
	// 	return nil, err
	// }

	userProfile, err := s.UserProfileRepo.GetByUserAccountID(ctx, userAccountID)
	if err != nil {
		return nil, err
	}

	return userProfile, nil
}

func (s *userService) GetProfileByUserProfileID(ctx context.Context, userProfileID int64) (*models.Profile, error) {
	userProfile, err := s.UserProfileRepo.Get(ctx, userProfileID)
	if err != nil {
		return nil, err
	}

	return s.ProfileRepo.Get(ctx, userProfile.ProfileID)
}

func (s *userService) GetProfileByProfileID(ctx context.Context, profileID int64) (*models.Profile, error) {
	return s.ProfileRepo.Get(ctx, profileID)
}

func (s *userService) GetProfileByUserAccountID(ctx context.Context, userAccountID int64) (*models.Profile, error) {
	return s.ProfileRepo.GetByUserAccountID(ctx, userAccountID)
}
