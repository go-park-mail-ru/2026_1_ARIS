package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	cache "github.com/go-park-mail-ru/2026_1_ARIS/pkg/tarantool"
	mediapb "github.com/go-park-mail-ru/2026_1_ARIS/proto/media"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/user/internal/model"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/user/internal/repository"
)

var (
	ErrInvalidInput        = errors.New("invalid input")
	ErrUserAccountNotFound = repository.ErrUserAccountNotFound
	ErrProfileNotFound     = repository.ErrProfileNotFound
	ErrUserProfileNotFound = repository.ErrUserProfileNotFound
	ErrUsernameTaken       = errors.New("username already exists")
	ErrNothingToUpdate     = errors.New("no fields provided for update")
	ErrInvalidStatus       = errors.New("unknown status value")
	ErrFriendshipNotFound  = repository.ErrFriendshipNotFound
	ErrAlreadyFriends      = repository.ErrAlreadyExists
	ErrFriendshipNotExists = repository.ErrNoRowsAffected
)

type Service struct {
	store       repository.Store
	mediaClient mediapb.MediaServiceClient
	cache       ProfileCache
}

func New(store repository.Store, mediaClient ...mediapb.MediaServiceClient) *Service {
	var media mediapb.MediaServiceClient
	if len(mediaClient) > 0 {
		media = mediaClient[0]
	}
	return &Service{store: store, mediaClient: media}
}

type ProfileCache interface {
	GetAuthUserByAccount(ctx context.Context, userAccountID int64) (*cache.AuthUser, error)
	SetAuthUserByAccount(ctx context.Context, user cache.AuthUser) error
	DeleteAuthUserByAccount(ctx context.Context, userAccountID int64) error
	GetProfileDetails(ctx context.Context, profileID int64) (*cache.ProfileDetails, error)
	SetProfileDetails(ctx context.Context, details cache.ProfileDetails) error
	DeleteProfileDetails(ctx context.Context, profileID int64) error
	GetProfileSummary(ctx context.Context, profileID int64) (*cache.ProfileSummary, error)
	SetProfileSummary(ctx context.Context, summary cache.ProfileSummary) error
	DeleteProfileSummary(ctx context.Context, profileID int64) error
	GetProfileIDByAccount(ctx context.Context, userAccountID int64) (int64, error)
	SetProfileIDByAccount(ctx context.Context, userAccountID, profileID int64) error
	DeleteProfileIDByAccount(ctx context.Context, userAccountID int64) error
	GetPresence(ctx context.Context, userAccountID int64) (*cache.PresenceStatus, error)
}

func (s *Service) SetCache(cache ProfileCache) {
	s.cache = cache
}

type CreateAuthUserInput struct {
	Username     string
	PasswordHash string
	FirstName    string
	LastName     string
	Birthday     string
	Gender       model.Gender
}

type GetOrCreateOAuthUserInput struct {
	Provider       string
	ProviderUserID string
	Username       string
	Email          *string
	FirstName      string
	LastName       string
	Birthday       string
	Gender         model.Gender
}

type Credentials struct {
	UserAccountID int64
	PasswordHash  string
}

type AuthUser struct {
	UserAccountID int64
	UserProfileID int64
	ProfileID     int64
	Login         string
	Email         *string
	FirstName     string
	LastName      string
	AvatarID      *int64
	CreatedAt     time.Time
	IsOnline      bool
	LastSeenAt    *time.Time
}

type SearchProfileResult struct {
	ProfileID     int64
	UserAccountID int64
	Username      string
	FirstName     string
	LastName      string
	AvatarID      *int64
}

type Education struct {
	Institution *string
	Group       *string
}

type Work struct {
	Company  *string
	JobTitle *string
}

type ProfileDetails struct {
	ProfileID     int64
	UserProfileID int64
	UserAccountID int64
	Username      string
	AvatarID      *int64
	FirstName     string
	LastName      string
	Bio           *string
	ImageLink     *string
	Gender        model.Gender
	BirthdayDate  time.Time
	NativeTown    *string
	Phone         *string
	Email         *string
	Town          *string
	Education     []Education
	Work          []Work
	Interests     *string
	FavMusic      *string
	IsOnline      bool
	LastSeenAt    *time.Time
}

type UserCard struct {
	ID         int64
	FirstName  string
	LastName   string
	Username   string
	AvatarLink string
	IsOnline   bool
	LastSeenAt *time.Time
}

type LatestEvent struct {
	UserCard
	Type int
}

type UpdateFullProfileInput struct {
	Username     *string
	Email        *string
	Phone        *string
	FirstName    *string
	LastName     *string
	Bio          *string
	BirthdayDate *string
	Gender       *model.Gender
	NativeTown   *string
	Town         *string
	Institution  *string
	Group        *string
	Company      *string
	JobTitle     *string
	Interests    *string
	FavMusic     *string
	AvatarID     *int64
	RemoveAvatar *bool
}

func (s *Service) CheckUsernameAvailable(ctx context.Context, username string) (bool, error) {
	username = normalizeUsername(username)
	if username == "" {
		return false, ErrInvalidInput
	}
	_, err := s.store.Accounts.GetByUsername(ctx, username)
	if err == nil {
		return false, nil
	}
	if errors.Is(err, repository.ErrUserAccountNotFound) {
		return true, nil
	}
	return false, err
}

func (s *Service) CreateAuthUser(ctx context.Context, in CreateAuthUserInput) (*AuthUser, error) {
	username := normalizeUsername(in.Username)
	if username == "" || in.PasswordHash == "" || in.FirstName == "" || in.LastName == "" {
		return nil, ErrInvalidInput
	}

	available, err := s.CheckUsernameAvailable(ctx, username)
	if err != nil {
		return nil, err
	}
	if !available {
		return nil, ErrUsernameTaken
	}

	birthday, err := time.Parse(time.DateOnly, in.Birthday)
	if err != nil {
		return nil, ErrInvalidInput
	}

	accountID, err := s.store.Accounts.Save(ctx, *model.NewUserAccount(username, nil, nil, in.PasswordHash))
	if err != nil {
		return nil, err
	}

	profileID, err := s.store.Profiles.Save(ctx, *model.NewProfile(nil))
	if err != nil {
		return nil, err
	}

	userProfile := model.NewUserProfile(accountID, profileID, in.FirstName, in.LastName, nil, birthday, normalizeGender(in.Gender))
	if _, err := s.store.UserProfiles.Save(ctx, *userProfile); err != nil {
		return nil, err
	}

	return s.GetAuthUserByAccount(ctx, accountID)
}

func (s *Service) GetOrCreateOAuthUser(ctx context.Context, in GetOrCreateOAuthUserInput) (*AuthUser, error) {
	provider := normalizeUsername(in.Provider)
	providerUserID := strings.TrimSpace(in.ProviderUserID)
	if provider == "" || providerUserID == "" || s.store.OAuth == nil {
		return nil, ErrInvalidInput
	}

	accountID, err := s.store.OAuth.GetUserAccountID(ctx, provider, providerUserID)
	if err == nil {
		if err := s.syncOAuthProfile(ctx, accountID, in); err != nil {
			return nil, err
		}
		return s.GetAuthUserByAccount(ctx, accountID)
	}
	if !errors.Is(err, repository.ErrUserAccountNotFound) {
		return nil, err
	}

	username, err := s.availableOAuthUsername(ctx, provider, providerUserID, in.Username)
	if err != nil {
		return nil, err
	}

	birthday := oauthBirthday(in.Birthday)
	firstName := normalizeProfileName(in.FirstName, "VKID")
	lastName := normalizeProfileName(in.LastName, "User")
	email := normalizeOptionalEmail(in.Email)

	accountID, err = s.store.Accounts.Save(ctx, *model.NewUserAccount(username, nil, nil, oauthPasswordHash(provider, providerUserID)))
	if err != nil {
		return nil, err
	}

	profileID, err := s.store.Profiles.Save(ctx, *model.NewProfile(nil))
	if err != nil {
		return nil, err
	}

	userProfile := model.NewUserProfile(accountID, profileID, firstName, lastName, nil, birthday, normalizeGender(in.Gender))
	if _, err := s.store.UserProfiles.Save(ctx, *userProfile); err != nil {
		return nil, err
	}

	if err := s.store.OAuth.Save(ctx, provider, providerUserID, accountID, email); err != nil {
		return nil, err
	}

	return s.GetAuthUserByAccount(ctx, accountID)
}

func (s *Service) syncOAuthProfile(ctx context.Context, accountID int64, in GetOrCreateOAuthUserInput) error {
	userProfile, err := s.store.UserProfiles.GetByUserAccountID(ctx, accountID)
	if err != nil {
		return normalizeUserProfileError(err)
	}

	update := repository.UserProfileUpdate{ID: userProfile.ID}
	if firstName := normalizeProfileName(in.FirstName, ""); firstName != "" && firstName != userProfile.FirstName {
		update.FirstName = &firstName
	}
	if lastName := normalizeProfileName(in.LastName, ""); lastName != "" && lastName != userProfile.LastName {
		update.LastName = &lastName
	}
	gender := normalizeGender(in.Gender)
	if in.Gender != "" && gender != userProfile.Gender {
		update.Gender = &gender
	}

	if !update.HasUpdates() {
		return nil
	}
	if err := s.store.UserProfiles.Update(ctx, update); err != nil {
		return err
	}
	s.invalidateProfileCache(ctx, accountID, userProfile.ProfileID)
	return nil
}

func (s *Service) GetCredentialsByLogin(ctx context.Context, login string) (*Credentials, error) {
	login = normalizeUsername(login)
	if login == "" {
		return nil, ErrInvalidInput
	}
	account, err := s.store.Accounts.GetByUsername(ctx, login)
	if err != nil {
		return nil, normalizeAccountError(err)
	}
	return &Credentials{UserAccountID: account.ID, PasswordHash: account.PasswordHash}, nil
}

func (s *Service) GetAuthUserByAccount(ctx context.Context, userAccountID int64) (*AuthUser, error) {
	if userAccountID <= 0 {
		return nil, ErrInvalidInput
	}
	if cached, ok := s.cachedAuthUser(ctx, userAccountID); ok {
		return cached, nil
	}

	account, err := s.store.Accounts.Get(ctx, userAccountID)
	if err != nil {
		return nil, normalizeAccountError(err)
	}
	userProfile, err := s.store.UserProfiles.GetByUserAccountID(ctx, userAccountID)
	if err != nil {
		return nil, normalizeUserProfileError(err)
	}
	profile, err := s.store.Profiles.GetByUserAccountID(ctx, userAccountID)
	if err != nil {
		return nil, normalizeProfileError(err)
	}

	user := &AuthUser{
		UserAccountID: account.ID,
		UserProfileID: userProfile.ID,
		ProfileID:     userProfile.ProfileID,
		Login:         account.Username,
		Email:         account.Email,
		FirstName:     userProfile.FirstName,
		LastName:      userProfile.LastName,
		AvatarID:      profile.AvatarID,
		CreatedAt:     userProfile.CreatedAt,
	}
	s.applyPresence(ctx, user)
	s.cacheAuthUser(ctx, user)
	s.cacheProfileIDByAccount(ctx, user.UserAccountID, user.ProfileID)
	return user, nil
}

func (s *Service) GetProfileByUserAccount(ctx context.Context, userAccountID int64) (*model.Profile, error) {
	if userAccountID <= 0 {
		return nil, ErrInvalidInput
	}
	if s.cache != nil {
		if profileID, err := s.cache.GetProfileIDByAccount(ctx, userAccountID); err == nil && profileID > 0 {
			return &model.Profile{ID: profileID}, nil
		}
	}
	profile, err := s.store.Profiles.GetByUserAccountID(ctx, userAccountID)
	if err != nil {
		return nil, normalizeProfileError(err)
	}
	s.cacheProfileIDByAccount(ctx, userAccountID, profile.ID)
	return profile, nil
}

func (s *Service) GetUserProfileByUserAccount(ctx context.Context, userAccountID int64) (*model.UserProfile, error) {
	if userAccountID <= 0 {
		return nil, ErrInvalidInput
	}
	userProfile, err := s.store.UserProfiles.GetByUserAccountID(ctx, userAccountID)
	if err != nil {
		return nil, normalizeUserProfileError(err)
	}
	return userProfile, nil
}

func (s *Service) GetProfileSummary(ctx context.Context, profileID int64) (*AuthUser, error) {
	if profileID <= 0 {
		return nil, ErrInvalidInput
	}
	if cached, ok := s.cachedProfileSummary(ctx, profileID); ok {
		return cached, nil
	}
	userProfile, err := s.store.UserProfiles.GetByProfileID(ctx, profileID)
	if err != nil {
		return nil, normalizeUserProfileError(err)
	}
	user, err := s.GetAuthUserByAccount(ctx, userProfile.UserAccountID)
	if err == nil {
		s.cacheProfileSummary(ctx, summaryFromAuthUser(user))
	}
	return user, err
}

func (s *Service) SearchProfiles(ctx context.Context, query string, limit int) ([]SearchProfileResult, error) {
	query = normalizeSearchQuery(query)
	if query == "" {
		return nil, ErrInvalidInput
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	profiles, err := s.store.UserProfiles.Search(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	result := make([]SearchProfileResult, 0, len(profiles))
	for _, profile := range profiles {
		result = append(result, SearchProfileResult{
			ProfileID:     profile.ProfileID,
			UserAccountID: profile.UserAccountID,
			Username:      profile.Username,
			FirstName:     profile.FirstName,
			LastName:      profile.LastName,
			AvatarID:      profile.AvatarID,
		})
	}
	return result, nil
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
	if cached, ok := s.cachedProfileDetails(ctx, profileID); ok {
		return cached, nil
	}
	userProfile, err := s.store.UserProfiles.GetByProfileID(ctx, profileID)
	if err != nil {
		return nil, normalizeUserProfileError(err)
	}
	account, err := s.store.Accounts.Get(ctx, userProfile.UserAccountID)
	if err != nil {
		return nil, normalizeAccountError(err)
	}
	profile, err := s.store.Profiles.Get(ctx, profileID)
	if err != nil {
		return nil, normalizeProfileError(err)
	}
	details := s.buildProfileDetails(ctx, account, userProfile, profile)
	s.applyPresence(ctx, details)
	s.cacheProfileDetails(ctx, details)
	s.cacheProfileSummary(ctx, summaryFromProfileDetails(details))
	s.cacheProfileIDByAccount(ctx, details.UserAccountID, details.ProfileID)
	return details, nil
}

func (s *Service) UpdateMe(ctx context.Context, userAccountID int64, update UpdateFullProfileInput) error {
	if userAccountID <= 0 {
		return ErrInvalidInput
	}
	userProfile, err := s.store.UserProfiles.GetByUserAccountID(ctx, userAccountID)
	if err != nil {
		return normalizeUserProfileError(err)
	}
	if update.Username != nil {
		username := normalizeUsername(*update.Username)
		update.Username = &username
	}
	accountUpdate := repository.AccountUpdate{ID: userAccountID, Username: update.Username, Email: update.Email, Phone: update.Phone}
	var birthday *time.Time
	if update.BirthdayDate != nil {
		parsed, err := time.Parse(time.DateOnly, *update.BirthdayDate)
		if err != nil {
			return ErrInvalidInput
		}
		birthday = &parsed
	}
	profileUpdate := repository.UserProfileUpdate{
		ID:           userProfile.ID,
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
		if err := s.store.Profiles.UpdateAvatar(ctx, userProfile.ProfileID, nil); err != nil {
			return err
		}
	}
	if update.AvatarID != nil {
		if err := s.store.Profiles.UpdateAvatar(ctx, userProfile.ProfileID, update.AvatarID); err != nil {
			return err
		}
	}
	s.invalidateProfileCache(ctx, userAccountID, userProfile.ProfileID)
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
	filtered := make([]model.Profile, 0, len(profiles))
	for _, profile := range profiles {
		if profile.ID == currentUserProfile.ProfileID {
			continue
		}
		account, err := s.accountByProfileID(ctx, profile.ID)
		if err != nil || normalizeUsername(account.Username) == "komandaaris" {
			continue
		}
		filtered = append(filtered, profile)
	}
	rand.Shuffle(len(filtered), func(i, j int) { filtered[i], filtered[j] = filtered[j], filtered[i] })
	if len(filtered) > 4 {
		filtered = filtered[:4]
	}
	return s.cardsFromProfiles(ctx, filtered), nil
}

func (s *Service) GetPublicPopularUsers(ctx context.Context) ([]UserCard, error) {
	return s.cardsByUsernames(ctx, []string{"sergeyshulginenko", "annaoparina", "ivankhvostov", "rinatbaikov"})
}

func (s *Service) GetLatestEvents(ctx context.Context) ([]LatestEvent, error) {
	targets := []struct {
		Username string
		Type     int
	}{{"sofiasitnichenko", 1}, {"daniilkhasyanov", 2}, {"konstantingalanin", 3}}
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
		if err == nil {
			events = append(events, LatestEvent{UserCard: card, Type: target.Type})
		}
	}
	return events, nil
}

func (s *Service) GetSettings(ctx context.Context, userAccountID int64) (*model.UserSettings, error) {
	if userAccountID <= 0 {
		return nil, ErrInvalidInput
	}
	settings, err := s.store.Settings.GetByUserID(ctx, userAccountID)
	if errors.Is(err, repository.ErrSettingsNotFound) {
		return defaultSettings(userAccountID), nil
	}
	return settings, err
}

func (s *Service) UpdateSettings(ctx context.Context, userAccountID int64, update repository.SettingsUpdate) (*model.UserSettings, error) {
	if userAccountID <= 0 {
		return nil, ErrInvalidInput
	}
	if update.IsEmpty() {
		return s.GetSettings(ctx, userAccountID)
	}
	settings, err := s.store.Settings.Update(ctx, userAccountID, update)
	if errors.Is(err, repository.ErrSettingsNotFound) {
		return applySettingsUpdate(defaultSettings(userAccountID), update), nil
	}
	return settings, err
}

func (s *Service) GetFriends(ctx context.Context, userAccountID int64, status model.FriendshipStatus) ([]model.Friend, error) {
	profileID, err := s.currentProfileID(ctx, userAccountID)
	if err != nil {
		return nil, err
	}
	if !validFriendshipStatus(string(status)) {
		return nil, ErrInvalidStatus
	}
	return s.store.Friendships.GetFriends(ctx, profileID, status)
}

func (s *Service) GetUsersFriends(ctx context.Context, profileID int64) ([]model.Friend, error) {
	if profileID <= 0 {
		return nil, ErrInvalidInput
	}
	if _, err := s.store.Profiles.Get(ctx, profileID); err != nil {
		return nil, normalizeProfileError(err)
	}
	return s.store.Friendships.GetFriends(ctx, profileID, model.FriendshipAccepted)
}

func (s *Service) GetFriendProfileIDs(ctx context.Context, userAccountID int64) ([]int64, error) {
	profileID, err := s.currentProfileID(ctx, userAccountID)
	if err != nil {
		return nil, err
	}
	friends, err := s.store.Friendships.GetFriends(ctx, profileID, model.FriendshipAccepted)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(friends))
	for _, f := range friends {
		ids = append(ids, f.ProfileID)
	}
	return ids, nil
}

func (s *Service) DeleteFriend(ctx context.Context, userAccountID int64, friendID int64) error {
	profileID, err := s.currentProfileID(ctx, userAccountID)
	if err != nil {
		return err
	}
	if friendID <= 0 {
		return ErrInvalidInput
	}
	return normalizeFriendshipMutationError(s.store.Friendships.DeleteFriend(ctx, profileID, friendID))
}

func (s *Service) RequestFriendship(ctx context.Context, userAccountID int64, friendID int64) error {
	profileID, err := s.currentProfileID(ctx, userAccountID)
	if err != nil {
		return err
	}
	if friendID <= 0 || friendID == profileID {
		return ErrInvalidInput
	}
	if _, err := s.store.Profiles.Get(ctx, friendID); err != nil {
		return normalizeProfileError(err)
	}
	if exists, status, err := s.checkFriendshipBy(ctx, friendID, profileID); err != nil {
		return err
	} else if exists {
		if status == model.FriendshipPending {
			return normalizeFriendshipMutationError(s.store.Friendships.AcceptFriendship(ctx, friendID, profileID))
		}
		return ErrAlreadyFriends
	}
	if exists, _, err := s.checkFriendshipBy(ctx, profileID, friendID); err != nil {
		return err
	} else if exists {
		return ErrAlreadyFriends
	}
	return normalizeFriendshipMutationError(s.store.Friendships.Create(ctx, profileID, friendID, string(model.FriendshipPending)))
}

func (s *Service) GetIncomingFriendRequests(ctx context.Context, userAccountID int64, status string) ([]model.Friend, error) {
	profileID, err := s.currentProfileID(ctx, userAccountID)
	if err != nil {
		return nil, err
	}
	if !validFriendshipStatus(status) {
		return nil, ErrInvalidStatus
	}
	return s.store.Friendships.GetIncomingFriends(ctx, profileID, status)
}

func (s *Service) GetOutgoingFriendRequests(ctx context.Context, userAccountID int64, status string) ([]model.Friend, error) {
	profileID, err := s.currentProfileID(ctx, userAccountID)
	if err != nil {
		return nil, err
	}
	if !validFriendshipStatus(status) {
		return nil, ErrInvalidStatus
	}
	return s.store.Friendships.GetOutgoingFriends(ctx, profileID, status)
}

func (s *Service) AcceptFriendRequest(ctx context.Context, userAccountID int64, requesterID int64) error {
	profileID, err := s.currentProfileID(ctx, userAccountID)
	if err != nil {
		return err
	}
	if requesterID <= 0 || requesterID == profileID {
		return ErrInvalidInput
	}
	if exists, status, err := s.checkFriendshipBy(ctx, requesterID, profileID); err != nil {
		return err
	} else if !exists || status != model.FriendshipPending {
		return ErrFriendshipNotExists
	}
	return normalizeFriendshipMutationError(s.store.Friendships.AcceptFriendship(ctx, requesterID, profileID))
}

func (s *Service) DeclineFriendRequest(ctx context.Context, userAccountID int64, requesterID int64) error {
	profileID, err := s.currentProfileID(ctx, userAccountID)
	if err != nil {
		return err
	}
	if requesterID <= 0 || requesterID == profileID {
		return ErrInvalidInput
	}
	if exists, status, err := s.checkFriendshipBy(ctx, requesterID, profileID); err != nil {
		return err
	} else if !exists || status != model.FriendshipPending {
		return ErrFriendshipNotExists
	}
	return normalizeFriendshipMutationError(s.store.Friendships.DeclineFriendship(ctx, requesterID, profileID))
}

func (s *Service) RevokeFriendRequest(ctx context.Context, userAccountID int64, addresseeID int64) error {
	profileID, err := s.currentProfileID(ctx, userAccountID)
	if err != nil {
		return err
	}
	if addresseeID <= 0 || addresseeID == profileID {
		return ErrInvalidInput
	}
	return normalizeFriendshipMutationError(s.store.Friendships.RevokeFriendRequest(ctx, profileID, addresseeID))
}

func normalizeUsername(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func (s *Service) availableOAuthUsername(ctx context.Context, provider, providerUserID, preferred string) (string, error) {
	base := normalizeOAuthUsername(preferred)
	if base == "" {
		base = normalizeOAuthUsername(provider + providerUserID)
	}
	if base == "" {
		base = "oauthuser"
	}

	for attempt := 0; attempt < 10; attempt++ {
		candidate := oauthUsernameCandidate(base, provider, providerUserID, attempt)
		available, err := s.CheckUsernameAvailable(ctx, candidate)
		if err != nil {
			return "", err
		}
		if available {
			return candidate, nil
		}
	}

	return "", ErrUsernameTaken
}

func oauthUsernameCandidate(base, provider, providerUserID string, attempt int) string {
	if attempt == 0 && len(base) <= 20 {
		return base
	}

	hash := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%d", provider, providerUserID, attempt)))
	suffix := hex.EncodeToString(hash[:])[:6]
	maxBaseLen := 20 - len(suffix)
	if len(base) > maxBaseLen {
		base = base[:maxBaseLen]
	}
	return base + suffix
}

func normalizeOAuthUsername(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			builder.WriteRune(r)
		}
	}
	result := builder.String()
	if len(result) > 20 {
		result = result[:20]
	}
	if len(result) >= 3 {
		return result
	}
	return ""
}

func normalizeProfileName(value, fallback string) string {
	value = strings.TrimSpace(value)
	var builder strings.Builder
	for _, r := range value {
		if isProfileNameRune(r) {
			builder.WriteRune(r)
		}
		if builder.Len() >= 255 {
			break
		}
	}
	if builder.Len() == 0 {
		return fallback
	}
	return builder.String()
}

func isProfileNameRune(r rune) bool {
	return r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= 'А' && r <= 'я' || r == 'Ё' || r == 'ё'
}

func normalizeOptionalEmail(value *string) *string {
	if value == nil {
		return nil
	}
	email := strings.TrimSpace(*value)
	if email == "" {
		return nil
	}
	return &email
}

func oauthBirthday(value string) time.Time {
	if parsed, err := time.Parse(time.DateOnly, strings.TrimSpace(value)); err == nil {
		return parsed
	}
	return time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
}

func oauthPasswordHash(provider, providerUserID string) string {
	hash := sha256.Sum256([]byte(provider + ":" + providerUserID))
	return "oauth:" + hex.EncodeToString(hash[:])
}

func (s *Service) buildProfileDetails(ctx context.Context, account *model.UserAccount, userProfile *model.UserProfile, profile *model.Profile) *ProfileDetails {
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
		Education:     []Education{{Institution: userProfile.Institution, Group: userProfile.Group}},
		Work:          []Work{{Company: userProfile.Company, JobTitle: userProfile.JobTitle}},
		Interests:     userProfile.Interests,
		FavMusic:      userProfile.FavMusic,
	}
}

func (s *Service) cachedAuthUser(ctx context.Context, userAccountID int64) (*AuthUser, bool) {
	if s.cache == nil {
		return nil, false
	}
	cached, err := s.cache.GetAuthUserByAccount(ctx, userAccountID)
	if err != nil {
		return nil, false
	}
	user := authUserFromCache(cached)
	s.applyPresence(ctx, user)
	return user, true
}

func (s *Service) cacheAuthUser(ctx context.Context, user *AuthUser) {
	if s.cache == nil || user == nil {
		return
	}
	_ = s.cache.SetAuthUserByAccount(ctx, authUserToCache(user))
}

func (s *Service) cachedProfileDetails(ctx context.Context, profileID int64) (*ProfileDetails, bool) {
	if s.cache == nil {
		return nil, false
	}
	cached, err := s.cache.GetProfileDetails(ctx, profileID)
	if err != nil {
		return nil, false
	}
	details := profileDetailsFromCache(cached)
	s.applyPresence(ctx, details)
	return details, true
}

func (s *Service) cacheProfileDetails(ctx context.Context, details *ProfileDetails) {
	if s.cache == nil || details == nil {
		return
	}
	_ = s.cache.SetProfileDetails(ctx, profileDetailsToCache(details))
}

func (s *Service) cachedProfileSummary(ctx context.Context, profileID int64) (*AuthUser, bool) {
	if s.cache == nil {
		return nil, false
	}
	cached, err := s.cache.GetProfileSummary(ctx, profileID)
	if err != nil {
		return nil, false
	}
	user := authUserFromSummary(cached)
	s.applyPresence(ctx, user)
	return user, true
}

func (s *Service) cacheProfileSummary(ctx context.Context, summary cache.ProfileSummary) {
	if s.cache == nil || summary.ProfileID <= 0 {
		return
	}
	_ = s.cache.SetProfileSummary(ctx, summary)
}

func (s *Service) cacheProfileIDByAccount(ctx context.Context, userAccountID, profileID int64) {
	if s.cache == nil || userAccountID <= 0 || profileID <= 0 {
		return
	}
	_ = s.cache.SetProfileIDByAccount(ctx, userAccountID, profileID)
}

func (s *Service) invalidateProfileCache(ctx context.Context, userAccountID, profileID int64) {
	if s.cache == nil {
		return
	}
	_ = s.cache.DeleteAuthUserByAccount(ctx, userAccountID)
	_ = s.cache.DeleteProfileIDByAccount(ctx, userAccountID)
	_ = s.cache.DeleteProfileDetails(ctx, profileID)
	_ = s.cache.DeleteProfileSummary(ctx, profileID)
}

func (s *Service) applyPresence(ctx context.Context, target interface{}) {
	var userAccountID int64
	switch value := target.(type) {
	case *AuthUser:
		userAccountID = value.UserAccountID
	case *ProfileDetails:
		userAccountID = value.UserAccountID
	default:
		return
	}

	presence := s.presence(ctx, userAccountID)
	if presence == nil {
		return
	}
	lastSeenAt := presence.LastSeenAt
	switch value := target.(type) {
	case *AuthUser:
		value.IsOnline = presence.IsOnline
		value.LastSeenAt = &lastSeenAt
	case *ProfileDetails:
		value.IsOnline = presence.IsOnline
		value.LastSeenAt = &lastSeenAt
	}
}

func (s *Service) presence(ctx context.Context, userAccountID int64) *cache.PresenceStatus {
	if s.cache == nil || userAccountID <= 0 {
		return nil
	}
	presence, err := s.cache.GetPresence(ctx, userAccountID)
	if err != nil {
		return nil
	}
	return presence
}

func authUserToCache(user *AuthUser) cache.AuthUser {
	if user == nil {
		return cache.AuthUser{}
	}
	return cache.AuthUser{
		UserAccountID: user.UserAccountID,
		UserProfileID: user.UserProfileID,
		ProfileID:     user.ProfileID,
		Login:         user.Login,
		Email:         user.Email,
		FirstName:     user.FirstName,
		LastName:      user.LastName,
		AvatarID:      user.AvatarID,
		CreatedAt:     user.CreatedAt,
		IsOnline:      user.IsOnline,
		LastSeenAt:    user.LastSeenAt,
	}
}

func authUserFromCache(user *cache.AuthUser) *AuthUser {
	if user == nil {
		return nil
	}
	return &AuthUser{
		UserAccountID: user.UserAccountID,
		UserProfileID: user.UserProfileID,
		ProfileID:     user.ProfileID,
		Login:         user.Login,
		Email:         user.Email,
		FirstName:     user.FirstName,
		LastName:      user.LastName,
		AvatarID:      user.AvatarID,
		CreatedAt:     user.CreatedAt,
		IsOnline:      user.IsOnline,
		LastSeenAt:    user.LastSeenAt,
	}
}

func authUserFromSummary(summary *cache.ProfileSummary) *AuthUser {
	if summary == nil {
		return nil
	}
	return &AuthUser{
		UserAccountID: summary.UserAccountID,
		ProfileID:     summary.ProfileID,
		Login:         summary.Username,
		FirstName:     summary.FirstName,
		LastName:      summary.LastName,
		AvatarID:      summary.AvatarID,
		IsOnline:      summary.IsOnline,
		LastSeenAt:    summary.LastSeenAt,
	}
}

func summaryFromAuthUser(user *AuthUser) cache.ProfileSummary {
	if user == nil {
		return cache.ProfileSummary{}
	}
	return cache.ProfileSummary{
		ProfileID:     user.ProfileID,
		UserAccountID: user.UserAccountID,
		FirstName:     user.FirstName,
		LastName:      user.LastName,
		Username:      user.Login,
		AvatarID:      user.AvatarID,
		IsOnline:      user.IsOnline,
		LastSeenAt:    user.LastSeenAt,
	}
}

func summaryFromProfileDetails(details *ProfileDetails) cache.ProfileSummary {
	if details == nil {
		return cache.ProfileSummary{}
	}
	return cache.ProfileSummary{
		ProfileID:     details.ProfileID,
		UserAccountID: details.UserAccountID,
		FirstName:     details.FirstName,
		LastName:      details.LastName,
		Username:      details.Username,
		AvatarID:      details.AvatarID,
		IsOnline:      details.IsOnline,
		LastSeenAt:    details.LastSeenAt,
	}
}

func profileDetailsToCache(details *ProfileDetails) cache.ProfileDetails {
	if details == nil {
		return cache.ProfileDetails{}
	}
	education := make([]cache.Education, 0, len(details.Education))
	for _, item := range details.Education {
		education = append(education, cache.Education{Institution: item.Institution, Group: item.Group})
	}
	work := make([]cache.Work, 0, len(details.Work))
	for _, item := range details.Work {
		work = append(work, cache.Work{Company: item.Company, JobTitle: item.JobTitle})
	}
	return cache.ProfileDetails{
		ProfileID:     details.ProfileID,
		UserProfileID: details.UserProfileID,
		UserAccountID: details.UserAccountID,
		Username:      details.Username,
		AvatarID:      details.AvatarID,
		FirstName:     details.FirstName,
		LastName:      details.LastName,
		Bio:           details.Bio,
		ImageLink:     details.ImageLink,
		Gender:        string(details.Gender),
		BirthdayDate:  details.BirthdayDate,
		NativeTown:    details.NativeTown,
		Phone:         details.Phone,
		Email:         details.Email,
		Town:          details.Town,
		Education:     education,
		Work:          work,
		Interests:     details.Interests,
		FavMusic:      details.FavMusic,
		IsOnline:      details.IsOnline,
		LastSeenAt:    details.LastSeenAt,
	}
}

func profileDetailsFromCache(details *cache.ProfileDetails) *ProfileDetails {
	if details == nil {
		return nil
	}
	education := make([]Education, 0, len(details.Education))
	for _, item := range details.Education {
		education = append(education, Education{Institution: item.Institution, Group: item.Group})
	}
	work := make([]Work, 0, len(details.Work))
	for _, item := range details.Work {
		work = append(work, Work{Company: item.Company, JobTitle: item.JobTitle})
	}
	return &ProfileDetails{
		ProfileID:     details.ProfileID,
		UserProfileID: details.UserProfileID,
		UserAccountID: details.UserAccountID,
		Username:      details.Username,
		AvatarID:      details.AvatarID,
		FirstName:     details.FirstName,
		LastName:      details.LastName,
		Bio:           details.Bio,
		ImageLink:     details.ImageLink,
		Gender:        model.Gender(details.Gender),
		BirthdayDate:  details.BirthdayDate,
		NativeTown:    details.NativeTown,
		Phone:         details.Phone,
		Email:         details.Email,
		Town:          details.Town,
		Education:     education,
		Work:          work,
		Interests:     details.Interests,
		FavMusic:      details.FavMusic,
		IsOnline:      details.IsOnline,
		LastSeenAt:    details.LastSeenAt,
	}
}

func (s *Service) cardsByUsernames(ctx context.Context, usernames []string) ([]UserCard, error) {
	profilesByUsername, err := s.profilesByUsername(ctx)
	if err != nil {
		return nil, err
	}
	profiles := make([]model.Profile, 0, len(usernames))
	for _, username := range usernames {
		if profile, ok := profilesByUsername[username]; ok {
			profiles = append(profiles, profile)
		}
	}
	return s.cardsFromProfiles(ctx, profiles), nil
}

func (s *Service) cardsFromProfiles(ctx context.Context, profiles []model.Profile) []UserCard {
	cards := make([]UserCard, 0, len(profiles))
	for _, profile := range profiles {
		card, err := s.cardFromProfile(ctx, profile)
		if err == nil {
			cards = append(cards, card)
		}
	}
	return cards
}

func (s *Service) cardFromProfile(ctx context.Context, profile model.Profile) (UserCard, error) {
	userProfile, err := s.store.UserProfiles.GetByProfileID(ctx, profile.ID)
	if err != nil {
		return UserCard{}, err
	}
	account, err := s.store.Accounts.Get(ctx, userProfile.UserAccountID)
	if err != nil {
		return UserCard{}, err
	}
	card := UserCard{
		ID:         profile.ID,
		FirstName:  userProfile.FirstName,
		LastName:   userProfile.LastName,
		Username:   account.Username,
		AvatarLink: derefString(s.avatarURL(ctx, profile.AvatarID)),
	}
	if presence := s.presence(ctx, account.ID); presence != nil {
		card.IsOnline = presence.IsOnline
		card.LastSeenAt = &presence.LastSeenAt
	}
	return card, nil
}

func (s *Service) profilesByUsername(ctx context.Context) (map[string]model.Profile, error) {
	profiles, err := s.store.Profiles.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	result := make(map[string]model.Profile, len(profiles))
	for _, profile := range profiles {
		account, err := s.accountByProfileID(ctx, profile.ID)
		if err == nil {
			result[normalizeUsername(account.Username)] = profile
		}
	}
	return result, nil
}

func (s *Service) accountByProfileID(ctx context.Context, profileID int64) (*model.UserAccount, error) {
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

func defaultSettings(userAccountID int64) *model.UserSettings {
	return &model.UserSettings{UserAccountID: userAccountID, Language: model.LanguageRU, Theme: model.ThemeLight}
}

func applySettingsUpdate(settings *model.UserSettings, update repository.SettingsUpdate) *model.UserSettings {
	if update.Language != nil {
		settings.Language = *update.Language
	}
	if update.Theme != nil {
		settings.Theme = *update.Theme
	}
	return settings
}

func (s *Service) currentProfileID(ctx context.Context, userAccountID int64) (int64, error) {
	if userAccountID <= 0 {
		return 0, ErrInvalidInput
	}
	profile, err := s.store.Profiles.GetByUserAccountID(ctx, userAccountID)
	if err != nil {
		return 0, normalizeProfileError(err)
	}
	return profile.ID, nil
}

func (s *Service) checkFriendshipBy(ctx context.Context, profileID, friendID int64) (bool, model.FriendshipStatus, error) {
	status, err := s.store.Friendships.GetFriendshipStatusBy(ctx, profileID, friendID)
	if err != nil {
		if errors.Is(err, repository.ErrFriendshipNotFound) {
			return false, "", nil
		}
		return false, "", err
	}
	if status == "" {
		return false, "", nil
	}
	return true, model.FriendshipStatus(status), nil
}

func validFriendshipStatus(status string) bool {
	return status == string(model.FriendshipPending) || status == string(model.FriendshipAccepted)
}

func normalizeFriendshipMutationError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, repository.ErrNoRowsAffected), errors.Is(err, repository.ErrFriendshipNotFound):
		return ErrFriendshipNotExists
	case errors.Is(err, repository.ErrAlreadyExists):
		return ErrAlreadyFriends
	default:
		return err
	}
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func normalizeSearchQuery(value string) string {
	return strings.TrimSpace(value)
}

func normalizeGender(value model.Gender) model.Gender {
	if value == model.Male {
		return model.Male
	}
	return model.Female
}

func normalizeAccountError(err error) error {
	if errors.Is(err, repository.ErrUserAccountNotFound) {
		return ErrUserAccountNotFound
	}
	return err
}

func normalizeProfileError(err error) error {
	if errors.Is(err, repository.ErrProfileNotFound) {
		return ErrProfileNotFound
	}
	return err
}

func normalizeUserProfileError(err error) error {
	if errors.Is(err, repository.ErrUserProfileNotFound) {
		return ErrUserProfileNotFound
	}
	return err
}
