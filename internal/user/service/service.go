package service

import (
	"context"
	"errors"
	"math/rand"
	"strings"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/dto"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/user/repository"
	mediapb "github.com/go-park-mail-ru/2026_1_ARIS/proto/media"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const maxSuggestedUsers = 4

var (
	ErrInvalidInput        = errors.New("invalid input")
	ErrProfileNotFound     = xerrors.ProfileNotFound
	ErrUserProfileNotFound = xerrors.UserProfileNotFound
	ErrUserAccountNotFound = xerrors.UserAccountNotFound
	ErrNothingToUpdate     = xerrors.ErrNothingToUpdate
)

type Service struct {
	store       repository.Store
	mediaClient mediapb.MediaServiceClient
}

func New(store repository.Store, mediaClient mediapb.MediaServiceClient) *Service {
	return &Service{store: store, mediaClient: mediaClient}
}

func (s *Service) GetProfileByUserAccount(ctx context.Context, userAccountID int64) (*models.Profile, error) {
	if userAccountID <= 0 {
		return nil, ErrInvalidInput
	}
	return s.store.Profiles.GetByUserAccountID(ctx, userAccountID)
}

func (s *Service) GetProfileByUserAccountID(ctx context.Context, userAccountID int64) (*models.Profile, error) {
	return s.GetProfileByUserAccount(ctx, userAccountID)
}

func (s *Service) GetUserProfileByUserAccount(ctx context.Context, userAccountID int64) (*models.UserProfile, error) {
	if userAccountID <= 0 {
		return nil, ErrInvalidInput
	}

	userProfile, err := s.store.UserProfiles.GetByUserAccountID(ctx, userAccountID)
	if err != nil {
		return nil, normalizeUserProfileError(err)
	}
	return userProfile, nil
}

func (s *Service) GetUserProfileByProfileID(ctx context.Context, profileID int64) (*models.UserProfile, error) {
	if profileID <= 0 {
		return nil, ErrInvalidInput
	}

	userProfile, err := s.store.UserProfiles.GetByProfileID(ctx, profileID)
	if err != nil {
		return nil, normalizeUserProfileError(err)
	}
	return userProfile, nil
}

func (s *Service) GetUserAccountByProfileID(ctx context.Context, profileID int64) (*models.UserAccount, error) {
	if profileID <= 0 {
		return nil, ErrInvalidInput
	}

	return s.accountByProfileID(ctx, profileID)
}

func (s *Service) GetProfileMe(ctx context.Context, userAccountID int64) (*ProfileDetails, error) {
	if userAccountID <= 0 {
		return nil, ErrInvalidInput
	}

	profile, err := s.store.Profiles.GetByUserAccountID(ctx, userAccountID)
	if err != nil {
		return nil, normalizeProfileError(err)
	}

	return s.GetProfileByID(ctx, profile.ID)
}

func (s *Service) GetProfileByID(ctx context.Context, profileID int64) (*ProfileDetails, error) {
	if profileID <= 0 {
		return nil, ErrInvalidInput
	}

	userProfile, err := s.store.UserProfiles.GetByProfileID(ctx, profileID)
	if err != nil {
		return nil, normalizeUserProfileError(err)
	}

	userAccount, err := s.store.Accounts.Get(ctx, userProfile.UserAccountID)
	if err != nil {
		return nil, normalizeUserAccountError(err)
	}

	profile, err := s.store.Profiles.Get(ctx, profileID)
	if err != nil {
		return nil, normalizeProfileError(err)
	}

	return s.buildProfileDetails(ctx, userAccount, userProfile, profile), nil
}

func (s *Service) GetProfileSummary(ctx context.Context, profileID int64) (*ProfileDetails, error) {
	return s.GetProfileByID(ctx, profileID)
}

func (s *Service) UpdateMe(ctx context.Context, userAccountID int64, update dto.UpdateFullProfileRequestDTO) error {
	if userAccountID <= 0 {
		return ErrInvalidInput
	}

	userProfile, err := s.store.UserProfiles.GetByUserAccountID(ctx, userAccountID)
	if err != nil {
		return normalizeUserProfileError(err)
	}

	update.UserAccountID = userAccountID
	update.UserProfileID = userProfile.ID
	update.ProfileID = userProfile.ProfileID

	if update.Username != nil {
		username := normalizeUsername(*update.Username)
		update.Username = &username
	}

	accountUpdate := dto.UpdateUserAccountDTO{
		ID:       update.UserAccountID,
		Username: update.Username,
		Email:    update.Email,
		Phone:    update.Phone,
	}

	var birthday *time.Time
	if update.BirthdayDate != nil {
		parsed, err := time.Parse(time.DateOnly, *update.BirthdayDate)
		if err != nil {
			return ErrInvalidInput
		}
		birthday = &parsed
	}

	profileUpdate := dto.UpdateUserProfileDTO{
		ID:           update.UserProfileID,
		FirstName:    update.FirstName,
		LastName:     update.LastName,
		Bio:          update.Bio,
		BirthdayDate: birthday,
		Gender:       update.Gender,
		NativeTown:   update.NativeTown,
		Town:         update.Town,
		Institution:  update.Institution,
		Group:        update.Group,
		Company:      update.Company,
		JobTitle:     update.JobTitle,
		Interests:    update.Interests,
		FavMusic:     update.FavMusic,
	}

	if !accountUpdate.HasUpdates() && !profileUpdate.HasUpdates() && update.AvatarID == nil && (update.RemoveAvatar == nil || !*update.RemoveAvatar) {
		return ErrNothingToUpdate
	}

	if accountUpdate.HasUpdates() {
		if err := s.store.Accounts.Update(ctx, accountUpdate); err != nil {
			return err
		}
	}

	if profileUpdate.HasUpdates() {
		if err := s.store.UserProfiles.Update(ctx, profileUpdate); err != nil {
			return err
		}
	}

	if update.RemoveAvatar != nil && *update.RemoveAvatar {
		if err := s.store.Profiles.UpdateAvatar(ctx, update.ProfileID, nil); err != nil {
			return err
		}
	}

	if update.AvatarID != nil {
		if err := s.store.Profiles.UpdateAvatar(ctx, update.ProfileID, update.AvatarID); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) GetSuggestedUsers(ctx context.Context, currentUserAccountID int64) ([]UserCard, error) {
	if currentUserAccountID <= 0 {
		return nil, ErrInvalidInput
	}

	currentUserProfile, err := s.store.UserProfiles.GetByUserAccountID(ctx, currentUserAccountID)
	if err != nil {
		return nil, normalizeUserProfileError(err)
	}

	profiles, err := s.store.Profiles.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	filtered := make([]models.Profile, 0, len(profiles))
	for _, profile := range profiles {
		if profile.ID == currentUserProfile.ProfileID {
			continue
		}
		userAccount, err := s.accountByProfileID(ctx, profile.ID)
		if err != nil || normalizeUsername(userAccount.Username) == "komandaaris" {
			continue
		}
		filtered = append(filtered, profile)
	}

	rand.Shuffle(len(filtered), func(i, j int) {
		filtered[i], filtered[j] = filtered[j], filtered[i]
	})
	if len(filtered) > maxSuggestedUsers {
		filtered = filtered[:maxSuggestedUsers]
	}

	return s.cardsFromProfiles(ctx, filtered), nil
}

func (s *Service) GetPublicPopularUsers(ctx context.Context) ([]UserCard, error) {
	targetUsernames := []string{
		"sergeyshulginenko",
		"annaoparina",
		"ivankhvostov",
		"rinatbaikov",
	}

	profilesByUsername, err := s.profilesByUsername(ctx)
	if err != nil {
		return nil, err
	}

	profiles := make([]models.Profile, 0, len(targetUsernames))
	for _, username := range targetUsernames {
		if profile, ok := profilesByUsername[username]; ok {
			profiles = append(profiles, profile)
		}
	}

	return s.cardsFromProfiles(ctx, profiles), nil
}

func (s *Service) GetLatestEvents(ctx context.Context) ([]LatestEvent, error) {
	targets := []struct {
		Username string
		Type     int
	}{
		{Username: "sofiasitnichenko", Type: 1},
		{Username: "daniilkhasyanov", Type: 2},
		{Username: "konstantingalanin", Type: 3},
	}

	profilesByUsername, err := s.profilesByUsername(ctx)
	if err != nil {
		return nil, err
	}

	events := make([]LatestEvent, 0, len(targets))
	for _, target := range targets {
		profile, ok := profilesByUsername[target.Username]
		if !ok {
			continue
		}
		card, err := s.cardFromProfile(ctx, profile)
		if err != nil {
			continue
		}
		events = append(events, LatestEvent{UserCard: card, Type: target.Type})
	}

	return events, nil
}

func (s *Service) GetSettings(ctx context.Context, userAccountID int64) (*models.UserSettings, error) {
	if userAccountID <= 0 {
		return nil, ErrInvalidInput
	}

	settings, err := s.store.Settings.GetByUserID(ctx, userAccountID)
	if errors.Is(err, xerrors.ErrUserSettingsNotFound) {
		return defaultSettings(userAccountID), nil
	}
	if err != nil {
		return nil, err
	}

	return settings, nil
}

func (s *Service) UpdateSettings(ctx context.Context, userAccountID int64, update dto.UserSettingsUpdate) (*models.UserSettings, error) {
	if userAccountID <= 0 {
		return nil, ErrInvalidInput
	}
	if update.IsEmpty() {
		return s.GetSettings(ctx, userAccountID)
	}

	settings, err := s.store.Settings.Update(ctx, userAccountID, update)
	if errors.Is(err, xerrors.ErrUserSettingsNotFound) {
		return applySettingsUpdate(defaultSettings(userAccountID), update), nil
	}
	if err != nil {
		return nil, err
	}

	return settings, nil
}

func (s *Service) buildProfileDetails(ctx context.Context, account *models.UserAccount, userProfile *models.UserProfile, profile *models.Profile) *ProfileDetails {
	return &ProfileDetails{
		ProfileID:     profile.ID,
		UserProfileID: userProfile.ID,
		UserAccountID: account.ID,
		Username:      account.Username,
		AvatarID:      profile.AvatarID,
		FirstName:     userProfile.FirstName,
		LastName:      userProfile.LastName,
		Bio:           userProfile.Bio,
		ImageLink:     s.avatarURL(ctx, profile.AvatarID),
		Gender:        userProfile.Gender,
		BirthdayDate:  userProfile.BirthdayDate,
		NativeTown:    userProfile.NativeTown,
		Phone:         account.Phone,
		Email:         account.Email,
		Town:          userProfile.Town,
		Interests:     userProfile.Interests,
		FavMusic:      userProfile.FavMusic,
		Education: []Education{{
			Institution: userProfile.Institution,
			Group:       userProfile.Group,
		}},
		Work: []Work{{
			Company:  userProfile.Company,
			JobTitle: userProfile.JobTitle,
		}},
	}
}

func (s *Service) cardsFromProfiles(ctx context.Context, profiles []models.Profile) []UserCard {
	cards := make([]UserCard, 0, len(profiles))
	for _, profile := range profiles {
		card, err := s.cardFromProfile(ctx, profile)
		if err == nil {
			cards = append(cards, card)
		}
	}
	return cards
}

func (s *Service) cardFromProfile(ctx context.Context, profile models.Profile) (UserCard, error) {
	userProfile, err := s.store.UserProfiles.GetByProfileID(ctx, profile.ID)
	if err != nil {
		return UserCard{}, err
	}

	userAccount, err := s.store.Accounts.Get(ctx, userProfile.UserAccountID)
	if err != nil {
		return UserCard{}, err
	}

	return UserCard{
		ID:         profile.ID,
		FirstName:  userProfile.FirstName,
		LastName:   userProfile.LastName,
		Username:   userAccount.Username,
		AvatarLink: derefString(s.avatarURL(ctx, profile.AvatarID)),
	}, nil
}

func (s *Service) profilesByUsername(ctx context.Context) (map[string]models.Profile, error) {
	profiles, err := s.store.Profiles.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	result := make(map[string]models.Profile, len(profiles))
	for _, profile := range profiles {
		account, err := s.accountByProfileID(ctx, profile.ID)
		if err != nil {
			continue
		}
		result[normalizeUsername(account.Username)] = profile
	}

	return result, nil
}

func (s *Service) accountByProfileID(ctx context.Context, profileID int64) (*models.UserAccount, error) {
	userProfile, err := s.store.UserProfiles.GetByProfileID(ctx, profileID)
	if err != nil {
		return nil, err
	}
	return s.store.Accounts.Get(ctx, userProfile.UserAccountID)
}

func (s *Service) avatarURL(ctx context.Context, avatarID *int64) *string {
	if avatarID == nil || *avatarID <= 0 || s.mediaClient == nil {
		return nil
	}

	resp, err := s.mediaClient.GetMediaURL(ctx, &mediapb.GetMediaURLRequest{MediaId: *avatarID})
	if err != nil || resp == nil || strings.TrimSpace(resp.GetUrl()) == "" {
		return nil
	}

	url := resp.GetUrl()
	return &url
}

func defaultSettings(userAccountID int64) *models.UserSettings {
	return &models.UserSettings{
		UserAccountID: userAccountID,
		Language:      models.LanguageRU,
		Theme:         models.ThemeLight,
	}
}

func applySettingsUpdate(settings *models.UserSettings, update dto.UserSettingsUpdate) *models.UserSettings {
	if update.Language != nil {
		settings.Language = *update.Language
	}
	if update.Theme != nil {
		settings.Theme = *update.Theme
	}
	return settings
}

func normalizeUsername(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeProfileError(err error) error {
	if errors.Is(err, xerrors.ProfileNotFound) || pgxscan.NotFound(err) {
		return ErrProfileNotFound
	}
	return err
}

func normalizeUserProfileError(err error) error {
	if errors.Is(err, xerrors.UserProfileNotFound) || pgxscan.NotFound(err) {
		return ErrUserProfileNotFound
	}
	return err
}

func normalizeUserAccountError(err error) error {
	if errors.Is(err, xerrors.UserAccountNotFound) || pgxscan.NotFound(err) {
		return ErrUserAccountNotFound
	}
	return err
}

func ToStatus(err error) error {
	switch {
	case errors.Is(err, ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, ErrProfileNotFound), errors.Is(err, ErrUserProfileNotFound), errors.Is(err, ErrUserAccountNotFound):
		return status.Error(codes.NotFound, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
