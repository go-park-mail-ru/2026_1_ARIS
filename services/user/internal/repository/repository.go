package repository

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/user/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

var (
	ErrUserAccountNotFound = errors.New("user account not found")
	ErrProfileNotFound     = errors.New("profile not found")
	ErrUserProfileNotFound = errors.New("user profile not found")
	ErrSettingsNotFound    = errors.New("user settings not found")
	ErrFriendshipNotFound  = errors.New("friendship not found")
	ErrNoRowsAffected      = errors.New("no rows affected")
	ErrAlreadyExists       = errors.New("already exists")
)

type DB interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type Store struct {
	Accounts     AccountRepo
	OAuth        OAuthAccountRepo
	Profiles     ProfileRepo
	UserProfiles UserProfileRepo
	Settings     SettingsRepo
	Friendships  FriendshipRepo
}

func NewStore(db DB) Store {
	return Store{
		Accounts:     NewAccountStorage(db),
		OAuth:        NewOAuthAccountStorage(db),
		Profiles:     NewProfileStorage(db),
		UserProfiles: NewUserProfileStorage(db),
		Settings:     NewSettingsStorage(db),
		Friendships:  NewFriendshipStorage(db),
	}
}

type AccountRepo interface {
	Save(ctx context.Context, account model.UserAccount) (int64, error)
	Get(ctx context.Context, id int64) (*model.UserAccount, error)
	GetByUsername(ctx context.Context, username string) (*model.UserAccount, error)
	Update(ctx context.Context, update AccountUpdate) error
}

type AccountUpdate struct {
	ID           int64
	Username     *string
	Email        *string
	Phone        *string
	PasswordHash *string
}

func (u AccountUpdate) HasUpdates() bool {
	return u.Username != nil || u.Email != nil || u.Phone != nil || u.PasswordHash != nil
}

type accountStorage struct {
	db DB
}

func NewAccountStorage(db DB) AccountRepo {
	return &accountStorage{db: db}
}

type OAuthAccountRepo interface {
	Save(ctx context.Context, provider, providerUserID string, userAccountID int64, email *string) error
	GetUserAccountID(ctx context.Context, provider, providerUserID string) (int64, error)
}

type oauthAccountStorage struct {
	db DB
}

func NewOAuthAccountStorage(db DB) OAuthAccountRepo {
	return &oauthAccountStorage{db: db}
}

func (s *oauthAccountStorage) Save(ctx context.Context, provider, providerUserID string, userAccountID int64, email *string) error {
	start := time.Now()
	_, err := s.db.Exec(ctx, `
		INSERT INTO oauth_account (provider, provider_user_id, user_account_id, email)
		VALUES ($1, $2, $3, $4)
	`, provider, providerUserID, userAccountID, email)
	logQuery(ctx, "oauthAccountStorage.Save", start)
	if err != nil {
		return err
	}
	return nil
}

func (s *oauthAccountStorage) GetUserAccountID(ctx context.Context, provider, providerUserID string) (int64, error) {
	start := time.Now()
	row := s.db.QueryRow(ctx, `
		SELECT user_account_id
		FROM oauth_account
		WHERE provider=$1 AND provider_user_id=$2
	`, provider, providerUserID)
	logQuery(ctx, "oauthAccountStorage.GetUserAccountID", start)

	var userAccountID int64
	if err := row.Scan(&userAccountID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrUserAccountNotFound
		}
		return 0, err
	}
	return userAccountID, nil
}

func (s *accountStorage) Save(ctx context.Context, account model.UserAccount) (int64, error) {
	start := time.Now()
	row := s.db.QueryRow(ctx,
		`INSERT INTO user_account (uid, email, phone, password_hash, username) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		uuid.New(), account.Email, account.Phone, account.PasswordHash, account.Username,
	)
	logQuery(ctx, "accountStorage.Save", start)

	var id int64
	if err := row.Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *accountStorage) Get(ctx context.Context, id int64) (*model.UserAccount, error) {
	var account model.UserAccount
	start := time.Now()
	err := pgxscan.Get(ctx, s.db, &account, `SELECT * FROM user_account WHERE id=$1`, id)
	logQuery(ctx, "accountStorage.Get", start)
	if err != nil {
		if pgxscan.NotFound(err) {
			return nil, ErrUserAccountNotFound
		}
		return nil, err
	}
	return &account, nil
}

func (s *accountStorage) GetByUsername(ctx context.Context, username string) (*model.UserAccount, error) {
	var account model.UserAccount
	start := time.Now()
	err := pgxscan.Get(ctx, s.db, &account, `SELECT * FROM user_account WHERE username=$1`, username)
	logQuery(ctx, "accountStorage.GetByUsername", start)
	if err != nil {
		if pgxscan.NotFound(err) {
			return nil, ErrUserAccountNotFound
		}
		return nil, err
	}
	return &account, nil
}

func (s *accountStorage) Update(ctx context.Context, update AccountUpdate) error {
	args := []any{}
	set := []string{}
	if update.Username != nil {
		args = append(args, update.Username)
		set = append(set, "username=$"+strconv.Itoa(len(args)))
	}
	if update.Email != nil {
		args = append(args, update.Email)
		set = append(set, "email=$"+strconv.Itoa(len(args)))
	}
	if update.Phone != nil {
		args = append(args, update.Phone)
		set = append(set, "phone=$"+strconv.Itoa(len(args)))
	}
	if update.PasswordHash != nil {
		args = append(args, *update.PasswordHash)
		set = append(set, "password_hash=$"+strconv.Itoa(len(args)))
	}
	if len(set) == 0 {
		return nil
	}
	args = append(args, update.ID)
	query := `UPDATE user_account SET ` + strings.Join(set, ", ") + `, updated_at=NOW() WHERE id=$` + strconv.Itoa(len(args))
	start := time.Now()
	tag, err := s.db.Exec(ctx, query, args...)
	logQuery(ctx, "accountStorage.Update", start)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrUserAccountNotFound
	}
	return nil
}

type ProfileRepo interface {
	Save(ctx context.Context, profile model.Profile) (int64, error)
	Get(ctx context.Context, profileID int64) (*model.Profile, error)
	GetAll(ctx context.Context) ([]model.Profile, error)
	GetByUserAccountID(ctx context.Context, userAccountID int64) (*model.Profile, error)
	UpdateAvatar(ctx context.Context, profileID int64, avatarID *int64) error
}

type profileStorage struct {
	db DB
}

func NewProfileStorage(db DB) ProfileRepo {
	return &profileStorage{db: db}
}

func (s *profileStorage) Save(ctx context.Context, profile model.Profile) (int64, error) {
	start := time.Now()
	row := s.db.QueryRow(ctx, `INSERT INTO profile (uid, avatar_id) VALUES ($1, $2) RETURNING id`, uuid.New(), profile.AvatarID)
	logQuery(ctx, "profileStorage.Save", start)

	var id int64
	if err := row.Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *profileStorage) Get(ctx context.Context, profileID int64) (*model.Profile, error) {
	var profile model.Profile
	start := time.Now()
	err := pgxscan.Get(ctx, s.db, &profile, `SELECT * FROM profile WHERE id=$1`, profileID)
	logQuery(ctx, "profileStorage.Get", start)
	if err != nil {
		if pgxscan.NotFound(err) {
			return nil, ErrProfileNotFound
		}
		return nil, err
	}
	return &profile, nil
}

func (s *profileStorage) GetAll(ctx context.Context) ([]model.Profile, error) {
	start := time.Now()
	rows, err := s.db.Query(ctx, `SELECT * FROM profile WHERE is_active=TRUE ORDER BY id ASC`)
	logQuery(ctx, "profileStorage.GetAll", start)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return pgx.CollectRows(rows, pgx.RowToStructByName[model.Profile])
}

func (s *profileStorage) GetByUserAccountID(ctx context.Context, userAccountID int64) (*model.Profile, error) {
	var profile model.Profile
	start := time.Now()
	err := pgxscan.Get(ctx, s.db, &profile, `
		SELECT p.id, p.uid, p.avatar_id, p.is_active, p.created_at, p.updated_at
		FROM user_account ua
		JOIN user_profile up ON up.user_account_id=ua.id
		JOIN profile p ON up.profile_id=p.id
		WHERE ua.id=$1
	`, userAccountID)
	logQuery(ctx, "profileStorage.GetByUserAccountID", start)
	if err != nil {
		if pgxscan.NotFound(err) {
			return nil, ErrProfileNotFound
		}
		return nil, err
	}
	return &profile, nil
}

func (s *profileStorage) UpdateAvatar(ctx context.Context, profileID int64, avatarID *int64) error {
	start := time.Now()
	tag, err := s.db.Exec(ctx, `UPDATE profile SET avatar_id=$1, updated_at=NOW() WHERE id=$2`, avatarID, profileID)
	logQuery(ctx, "profileStorage.UpdateAvatar", start)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrProfileNotFound
	}
	return nil
}

type UserProfileRepo interface {
	Save(ctx context.Context, userProfile model.UserProfile) (int64, error)
	GetByProfileID(ctx context.Context, profileID int64) (*model.UserProfile, error)
	GetByUserAccountID(ctx context.Context, userAccountID int64) (*model.UserProfile, error)
	Update(ctx context.Context, update UserProfileUpdate) error
	Search(ctx context.Context, query string, limit int) ([]SearchProfileResult, error)
}

type UserProfileUpdate struct {
	ID           int64
	FirstName    *string
	LastName     *string
	Bio          *string
	BirthdayDate *time.Time
	Gender       *model.Gender
	NativeTown   *string
	Town         *string
	Institution  *string
	Group        *string
	Company      *string
	JobTitle     *string
	Interests    *string
	FavMusic     *string
}

func (u UserProfileUpdate) HasUpdates() bool {
	return u.FirstName != nil || u.LastName != nil || u.Bio != nil || u.BirthdayDate != nil || u.Gender != nil ||
		u.NativeTown != nil || u.Town != nil || u.Institution != nil || u.Group != nil || u.Company != nil ||
		u.JobTitle != nil || u.Interests != nil || u.FavMusic != nil
}

type SearchProfileResult struct {
	ProfileID     int64  `db:"profile_id"`
	UserAccountID int64  `db:"user_account_id"`
	Username      string `db:"username"`
	FirstName     string `db:"first_name"`
	LastName      string `db:"last_name"`
	AvatarID      *int64 `db:"avatar_id"`
}

type userProfileStorage struct {
	db DB
}

func NewUserProfileStorage(db DB) UserProfileRepo {
	return &userProfileStorage{db: db}
}

func (s *userProfileStorage) Save(ctx context.Context, userProfile model.UserProfile) (int64, error) {
	start := time.Now()
	row := s.db.QueryRow(ctx, `
		INSERT INTO user_profile (uid, user_account_id, profile_id, first_name, last_name, bio, birthday_date, gender)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`, uuid.New(), userProfile.UserAccountID, userProfile.ProfileID, userProfile.FirstName, userProfile.LastName, userProfile.Bio, userProfile.BirthdayDate, userProfile.Gender)
	logQuery(ctx, "userProfileStorage.Save", start)

	var id int64
	if err := row.Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *userProfileStorage) GetByProfileID(ctx context.Context, profileID int64) (*model.UserProfile, error) {
	var userProfile model.UserProfile
	start := time.Now()
	err := pgxscan.Get(ctx, s.db, &userProfile, `SELECT * FROM user_profile WHERE profile_id=$1`, profileID)
	logQuery(ctx, "userProfileStorage.GetByProfileID", start)
	if err != nil {
		if pgxscan.NotFound(err) {
			return nil, ErrUserProfileNotFound
		}
		return nil, err
	}
	return &userProfile, nil
}

func (s *userProfileStorage) GetByUserAccountID(ctx context.Context, userAccountID int64) (*model.UserProfile, error) {
	var userProfile model.UserProfile
	start := time.Now()
	err := pgxscan.Get(ctx, s.db, &userProfile, `SELECT * FROM user_profile WHERE user_account_id=$1`, userAccountID)
	logQuery(ctx, "userProfileStorage.GetByUserAccountID", start)
	if err != nil {
		if pgxscan.NotFound(err) {
			return nil, ErrUserProfileNotFound
		}
		return nil, err
	}
	return &userProfile, nil
}

func (s *userProfileStorage) Update(ctx context.Context, update UserProfileUpdate) error {
	args := []any{}
	set := []string{}
	add := func(column string, value any) {
		args = append(args, value)
		set = append(set, column+"=$"+strconv.Itoa(len(args)))
	}
	if update.FirstName != nil {
		add("first_name", update.FirstName)
	}
	if update.LastName != nil {
		add("last_name", update.LastName)
	}
	if update.Bio != nil {
		add("bio", update.Bio)
	}
	if update.BirthdayDate != nil {
		add("birthday_date", update.BirthdayDate)
	}
	if update.Gender != nil {
		add("gender", update.Gender)
	}
	if update.NativeTown != nil {
		add("native_town", update.NativeTown)
	}
	if update.Town != nil {
		add("town", update.Town)
	}
	if update.Institution != nil {
		add("institution", update.Institution)
	}
	if update.Group != nil {
		add("study_group", update.Group)
	}
	if update.Company != nil {
		add("company", update.Company)
	}
	if update.JobTitle != nil {
		add("job_title", update.JobTitle)
	}
	if update.Interests != nil {
		add("interests", update.Interests)
	}
	if update.FavMusic != nil {
		add("fav_music", update.FavMusic)
	}
	if len(set) == 0 {
		return nil
	}
	args = append(args, update.ID)
	query := `UPDATE user_profile SET ` + strings.Join(set, ", ") + `, updated_at=NOW() WHERE id=$` + strconv.Itoa(len(args))
	start := time.Now()
	tag, err := s.db.Exec(ctx, query, args...)
	logQuery(ctx, "userProfileStorage.Update", start)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrUserProfileNotFound
	}
	return nil
}

func (s *userProfileStorage) Search(ctx context.Context, query string, limit int) ([]SearchProfileResult, error) {
	pattern := likePattern(query)
	prefixPattern := likePrefixPattern(query)
	sql := `
		SELECT p.id AS profile_id, ua.id AS user_account_id, ua.username,
		       up.first_name, up.last_name, p.avatar_id
		FROM user_profile up
		JOIN user_account ua ON ua.id = up.user_account_id
		JOIN profile p ON p.id = up.profile_id
		WHERE up.is_active = TRUE
		  AND ua.is_active = TRUE
		  AND p.is_active = TRUE
		  AND (
			ua.username ILIKE $1 ESCAPE E'\\'
			OR up.first_name ILIKE $1 ESCAPE E'\\'
			OR up.last_name ILIKE $1 ESCAPE E'\\'
			OR (up.first_name || ' ' || up.last_name) ILIKE $1 ESCAPE E'\\'
		  )
		ORDER BY
		  CASE
			WHEN ua.username ILIKE $2 ESCAPE E'\\' THEN 0
			WHEN up.first_name ILIKE $2 ESCAPE E'\\' THEN 1
			WHEN up.last_name ILIKE $2 ESCAPE E'\\' THEN 2
			ELSE 3
		  END,
		  up.first_name ASC,
		  up.last_name ASC,
		  p.id ASC
		LIMIT $3`

	start := time.Now()
	rows, err := s.db.Query(ctx, sql, pattern, prefixPattern, limit)
	logQuery(ctx, "userProfileStorage.Search", start)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return pgx.CollectRows(rows, pgx.RowToStructByName[SearchProfileResult])
}

func likePattern(query string) string {
	return "%" + escapeLike(strings.TrimSpace(query)) + "%"
}

func likePrefixPattern(query string) string {
	return escapeLike(strings.TrimSpace(query)) + "%"
}

func escapeLike(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(value)
}

type SettingsUpdate struct {
	Language *model.LanguageSetting
	Theme    *model.ThemeSetting
}

func (u SettingsUpdate) IsEmpty() bool {
	return u.Language == nil && u.Theme == nil
}

type SettingsRepo interface {
	GetByUserID(ctx context.Context, userID int64) (*model.UserSettings, error)
	Update(ctx context.Context, userID int64, update SettingsUpdate) (*model.UserSettings, error)
}

type settingsStorage struct {
	db DB
}

func NewSettingsStorage(db DB) SettingsRepo {
	return &settingsStorage{db: db}
}

func (s *settingsStorage) GetByUserID(ctx context.Context, userID int64) (*model.UserSettings, error) {
	start := time.Now()
	rows, err := s.db.Query(ctx, `
		SELECT user_account_id, lang, theme
		FROM user_settings
		WHERE user_account_id=$1
	`, userID)
	logQuery(ctx, "settingsStorage.GetByUserID", start)
	if err != nil {
		return nil, err
	}
	settings, err := pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByName[model.UserSettings])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSettingsNotFound
	}
	return settings, err
}

func (s *settingsStorage) Update(ctx context.Context, userID int64, update SettingsUpdate) (*model.UserSettings, error) {
	if update.IsEmpty() {
		return s.GetByUserID(ctx, userID)
	}
	args := []any{userID}
	set := []string{}
	if update.Language != nil {
		args = append(args, strings.ToUpper(string(*update.Language)))
		set = append(set, "lang=$"+strconv.Itoa(len(args)))
	}
	if update.Theme != nil {
		args = append(args, strings.ToLower(string(*update.Theme)))
		set = append(set, "theme=$"+strconv.Itoa(len(args)))
	}
	query := `
		UPDATE user_settings
		SET ` + strings.Join(set, ", ") + `
		WHERE user_account_id=$1
		RETURNING user_account_id, lang, theme`
	start := time.Now()
	rows, err := s.db.Query(ctx, query, args...)
	logQuery(ctx, "settingsStorage.Update", start)
	if err != nil {
		return nil, err
	}
	settings, err := pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByName[model.UserSettings])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSettingsNotFound
	}
	return settings, err
}

type FriendshipRepo interface {
	GetFriends(ctx context.Context, profileID int64, status model.FriendshipStatus) ([]model.Friend, error)
	GetFriendshipStatusBy(ctx context.Context, profileID, friendID int64) (string, error)
	GetOutgoingFriends(ctx context.Context, profileID int64, status string) ([]model.Friend, error)
	GetIncomingFriends(ctx context.Context, profileID int64, status string) ([]model.Friend, error)
	Create(ctx context.Context, profileID, friendID int64, status string) error
	AcceptFriendship(ctx context.Context, profileID1, profileID2 int64) error
	DeclineFriendship(ctx context.Context, profileID1, profileID2 int64) error
	RevokeFriendRequest(ctx context.Context, profileID, friendID int64) error
	DeleteFriend(ctx context.Context, profileID, friendID int64) error
}

type friendshipStorage struct {
	db DB
}

func NewFriendshipStorage(db DB) FriendshipRepo {
	return &friendshipStorage{db: db}
}

func (s *friendshipStorage) GetFriends(ctx context.Context, profileID int64, status model.FriendshipStatus) ([]model.Friend, error) {
	return s.collectFriends(ctx, "friendshipStorage.GetFriends", `
		SELECT p.avatar_id, p.id, up.first_name, up.last_name, ua.username, m.link, status, f.created_at, f.updated_at
		FROM (
			SELECT f.created_at, f.updated_at, status,
			       CASE WHEN requester_id=$1 THEN addressee_id WHEN addressee_id=$1 THEN requester_id END AS friend
			FROM friendship f
			WHERE $1 IN (requester_id, addressee_id) AND status=$2
		) AS f
		JOIN profile p ON p.id=friend
		JOIN user_profile up ON up.profile_id=friend
		JOIN user_account ua ON up.user_account_id=ua.id
		LEFT JOIN media m ON p.avatar_id=m.id AND (m.mime_type LIKE 'image/%' OR m.mime_type='image')
		ORDER BY p.id ASC
	`, profileID, string(status))
}

func (s *friendshipStorage) GetFriendshipStatusBy(ctx context.Context, profileID, friendID int64) (string, error) {
	start := time.Now()
	row := s.db.QueryRow(ctx, `SELECT status FROM friendship WHERE requester_id=$1 AND addressee_id=$2`, profileID, friendID)
	logQuery(ctx, "friendshipStorage.GetFriendshipStatusBy", start)
	var status string
	if err := row.Scan(&status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrFriendshipNotFound
		}
		return "", err
	}
	return status, nil
}

func (s *friendshipStorage) GetOutgoingFriends(ctx context.Context, profileID int64, status string) ([]model.Friend, error) {
	return s.collectFriends(ctx, "friendshipStorage.GetOutgoingFriends", `
		SELECT f.addressee_id AS id, p.avatar_id, up.first_name, up.last_name, ua.username, m.link, f.status, f.created_at, f.updated_at
		FROM friendship f
		JOIN profile p ON p.id=f.addressee_id
		JOIN user_profile up ON up.profile_id=p.id
		JOIN user_account ua ON ua.id=up.user_account_id
		LEFT JOIN media m ON m.id=p.avatar_id
		WHERE f.requester_id=$1 AND f.status=$2
	`, profileID, status)
}

func (s *friendshipStorage) GetIncomingFriends(ctx context.Context, profileID int64, status string) ([]model.Friend, error) {
	return s.collectFriends(ctx, "friendshipStorage.GetIncomingFriends", `
		SELECT f.requester_id AS id, p.avatar_id, up.first_name, up.last_name, ua.username, m.link, f.status, f.created_at, f.updated_at
		FROM friendship f
		JOIN profile p ON p.id=f.requester_id
		JOIN user_profile up ON up.profile_id=p.id
		JOIN user_account ua ON ua.id=up.user_account_id
		LEFT JOIN media m ON m.id=p.avatar_id
		WHERE f.addressee_id=$1 AND f.status=$2
	`, profileID, status)
}

func (s *friendshipStorage) collectFriends(ctx context.Context, label, query string, args ...any) ([]model.Friend, error) {
	start := time.Now()
	rows, err := s.db.Query(ctx, query, args...)
	logQuery(ctx, label, start)
	if err != nil {
		return []model.Friend{}, err
	}
	defer rows.Close()
	friends, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.Friend])
	if err != nil {
		return nil, err
	}
	return friends, nil
}

func (s *friendshipStorage) Create(ctx context.Context, profileID, friendID int64, status string) error {
	return rowsAffectedOne(ctx, s.db, "friendshipStorage.Create", `INSERT INTO friendship (requester_id, addressee_id, status) VALUES ($1, $2, $3)`, profileID, friendID, status)
}

func (s *friendshipStorage) AcceptFriendship(ctx context.Context, profileID1, profileID2 int64) error {
	return rowsAffectedOne(ctx, s.db, "friendshipStorage.AcceptFriendship", `
		UPDATE friendship f SET status='accepted'
		WHERE LEAST(requester_id, addressee_id)=LEAST($1::bigint, $2::bigint)
		  AND GREATEST(requester_id, addressee_id)=GREATEST($1::bigint, $2::bigint)
		  AND status='pending'::friendship_status
	`, profileID1, profileID2)
}

func (s *friendshipStorage) DeclineFriendship(ctx context.Context, profileID1, profileID2 int64) error {
	return rowsAffectedOne(ctx, s.db, "friendshipStorage.DeclineFriendship", `
		DELETE FROM friendship
		WHERE LEAST(requester_id, addressee_id)=LEAST($1::bigint, $2::bigint)
		  AND GREATEST(requester_id, addressee_id)=GREATEST($1::bigint, $2::bigint)
		  AND status='pending'
	`, profileID1, profileID2)
}

func (s *friendshipStorage) RevokeFriendRequest(ctx context.Context, profileID, friendID int64) error {
	return rowsAffectedOne(ctx, s.db, "friendshipStorage.RevokeFriendRequest", `DELETE FROM friendship WHERE requester_id=$1 AND addressee_id=$2 AND status='pending'`, profileID, friendID)
}

func (s *friendshipStorage) DeleteFriend(ctx context.Context, profileID, friendID int64) error {
	return rowsAffectedOne(ctx, s.db, "friendshipStorage.DeleteFriend", `
		DELETE FROM friendship
		WHERE LEAST(requester_id, addressee_id)=LEAST($1::bigint, $2::bigint)
		  AND GREATEST(requester_id, addressee_id)=GREATEST($1::bigint, $2::bigint)
		  AND status='accepted'::friendship_status
	`, profileID, friendID)
}

func rowsAffectedOne(ctx context.Context, db DB, label string, query string, args ...any) error {
	start := time.Now()
	tag, err := db.Exec(ctx, query, args...)
	logQuery(ctx, label, start)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrAlreadyExists
		}
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNoRowsAffected
	}
	return nil
}

func logQuery(ctx context.Context, query string, start time.Time) {
	logg := logger.FromContext(ctx)
	if logg == nil {
		return
	}
	logg.Debug("db query", zap.String("query", query), zap.Duration("duration_ms", time.Since(start)))
}
