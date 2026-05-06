package userprofile

import (
	"context"
	"testing"
	"time"

	hdto "github.com/go-park-mail-ru/2026_1_ARIS/internal/dto"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/google/uuid"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestUserProfileStorageSave(t *testing.T) {
	t.Parallel()

	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	repo := NewUserProfileStorage(mockPool)
	bio := "bio"
	rows := pgxmock.NewRows([]string{"id"}).AddRow(int64(99))
	mockPool.ExpectQuery("INSERT INTO user_profile").
		WithArgs(
			pgxmock.AnyArg(),
			int64(1),
			int64(2),
			"John",
			"Doe",
			&bio,
			pgxmock.AnyArg(),
			models.Gender("male"),
		).
		WillReturnRows(rows)

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(context.Background(), mockLogger)

	gotID, err := repo.Save(ctx, models.UserProfile{
		UserAccountID: 1,
		ProfileID:     2,
		FirstName:     "John",
		LastName:      "Doe",
		Bio:           &bio,
		BirthdayDate:  time.Now(),
		Gender:        models.Gender("male"),
	})
	require.NoError(t, err)
	require.Equal(t, int64(99), gotID)
	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestUserProfileStorageUpdate(t *testing.T) {
	t.Parallel()

	t.Run("nothing to update", func(t *testing.T) {
		t.Parallel()
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		mockLogger := zap.NewNop()
		ctx := logger.WithLogger(context.Background(), mockLogger)

		repo := NewUserProfileStorage(mockPool)
		err = repo.Update(ctx, hdto.UpdateUserProfileDTO{ID: 1})
		require.NoError(t, err)
		require.NoError(t, mockPool.ExpectationsWereMet())
	})

	t.Run("updated one row", func(t *testing.T) {
		t.Parallel()
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		repo := NewUserProfileStorage(mockPool)
		firstName := "Jane"
		lastName := "Smith"
		bio := "New bio"
		birthdayDate := time.Date(1990, time.January, 1, 0, 0, 0, 0, time.UTC)
		denger := models.Male
		nativeTown := "Saint Petersburg"
		institution := "University"
		studyGroup := "CS-101"
		company := "Tech Corp"
		jobTitle := "Software Engineer"
		interests := "Programming, Music"
		favMusic := "Rock"
		town := "Moscow"
		mockPool.ExpectExec("UPDATE user_profile SET first_name=\\$1, last_name=\\$2, bio=\\$3, birthday_date=\\$4, gender=\\$5, native_town=\\$6, town=\\$7, institution=\\$8, study_group=\\$9, company=\\$10, job_title=\\$11, interests=\\$12, fav_music=\\$13 WHERE id=\\$14").
			WithArgs(firstName, lastName, bio, birthdayDate, denger, nativeTown, town, institution, studyGroup, company, jobTitle, interests, favMusic, int64(3)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		mockLogger := zap.NewNop()
		ctx := logger.WithLogger(context.Background(), mockLogger)

		err = repo.Update(ctx, hdto.UpdateUserProfileDTO{
			ID:           3,
			FirstName:    &firstName,
			LastName:     &lastName,
			Town:         &town,
			Bio:          &bio,
			BirthdayDate: &birthdayDate,
			Gender:       &denger,
			NativeTown:   &nativeTown,
			Institution:  &institution,
			Group:        &studyGroup,
			Company:      &company,
			JobTitle:     &jobTitle,
			Interests:    &interests,
			FavMusic:     &favMusic,
		})
		require.NoError(t, err)
		require.NoError(t, mockPool.ExpectationsWereMet())
	})
}

func TestUserProfileStorageGetByProfileID(t *testing.T) {
	t.Parallel()

	t.Run("not found mapped", func(t *testing.T) {
		t.Parallel()
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		repo := NewUserProfileStorage(mockPool)
		rows := pgxmock.NewRows([]string{"id", "uid", "user_account_id", "profile_id", "first_name", "last_name", "bio", "birthday_date", "gender", "native_town", "town", "institution", "study_group", "company", "job_title", "interests", "fav_music", "is_active", "created_at", "updated_at"})
		mockPool.ExpectQuery("SELECT \\* FROM user_profile WHERE profile_id=\\$1;").
			WithArgs(int64(555)).
			WillReturnRows(rows)

		mockLogger := zap.NewNop()
		ctx := logger.WithLogger(context.Background(), mockLogger)

		_, err = repo.GetByProfileID(ctx, 555)
		require.ErrorIs(t, err, xerrors.UserProfileNotFound)
		require.NoError(t, mockPool.ExpectationsWereMet())
	})
}

func TestUserProfileStorageGetAndGetByUserAccountID(t *testing.T) {
	t.Parallel()

	now := time.Now()
	cols := []string{"id", "uid", "user_account_id", "profile_id", "first_name", "last_name", "bio", "birthday_date", "gender", "native_town", "town", "institution", "study_group", "company", "job_title", "interests", "fav_music", "is_active", "created_at", "updated_at"}

	t.Run("Get", func(t *testing.T) {
		mockPool, _ := pgxmock.NewPool()
		defer mockPool.Close()
		repo := NewUserProfileStorage(mockPool)
		rows := pgxmock.NewRows(cols).AddRow(int64(1), uuid.New(), int64(10), int64(20), "A", "B", nil, now, "male", nil, nil, nil, nil, nil, nil, nil, nil, true, now, now)
		mockPool.ExpectQuery("SELECT \\* FROM user_profile WHERE id=\\$1;").WithArgs(int64(1)).WillReturnRows(rows)

		mockLogger := zap.NewNop()
		ctx := logger.WithLogger(context.Background(), mockLogger)

		got, err := repo.Get(ctx, 1)
		require.NoError(t, err)
		require.Equal(t, int64(1), got.ID)
	})

	t.Run("GetByUserAccountID", func(t *testing.T) {
		mockPool, _ := pgxmock.NewPool()
		defer mockPool.Close()
		repo := NewUserProfileStorage(mockPool)
		rows := pgxmock.NewRows(cols).AddRow(int64(1), uuid.New(), int64(10), int64(20), "A", "B", nil, now, "male", nil, nil, nil, nil, nil, nil, nil, nil, true, now, now)
		mockPool.ExpectQuery("SELECT \\* FROM user_profile WHERE user_account_id=\\$1;").WithArgs(int64(10)).WillReturnRows(rows)

		mockLogger := zap.NewNop()
		ctx := logger.WithLogger(context.Background(), mockLogger)

		got, err := repo.GetByUserAccountID(ctx, 10)
		require.NoError(t, err)
		require.Equal(t, int64(10), got.UserAccountID)
	})
}
