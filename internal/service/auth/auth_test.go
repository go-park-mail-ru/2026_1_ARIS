package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/mocks"
	repomocks "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/mocks"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestAuthServiceCreateRealUserProfile(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := repomocks.NewMockUserAccountRepo(ctrl)
	profRepo := repomocks.NewMockProfileRepo(ctrl)
	upRepo := repomocks.NewMockUserProfileRepo(ctrl)

	svc := NewAuthService(userRepo, profRepo, upRepo)

	userRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(int64(10), nil)
	profRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(int64(20), nil)
	upRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(int64(30), nil)
	want := &models.Profile{ID: 20}
	profRepo.EXPECT().Get(gomock.Any(), int64(20)).Return(want, nil)

	got, err := svc.CreateRealUserProfile(context.Background(), "hash", "login", "A", "B", nil, nil, true, time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC), models.Male, nil)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestAuthServiceRegister(t *testing.T) {
	t.Parallel()

	t.Run("duplicate login", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		userRepo := repomocks.NewMockUserAccountRepo(ctrl)
		svc := NewAuthService(userRepo, repomocks.NewMockProfileRepo(ctrl), repomocks.NewMockUserProfileRepo(ctrl))
		userRepo.EXPECT().GetByUsername(gomock.Any(), "ivan").Return(&models.UserAccount{}, nil)
		_, err := svc.Register(context.Background(), "I", "P", "IVAN", "pass1", "01/01/1990", models.Male)
		require.ErrorContains(t, err, "login")
	})

	t.Run("invalid birthday", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		userRepo := repomocks.NewMockUserAccountRepo(ctrl)
		svc := NewAuthService(userRepo, repomocks.NewMockProfileRepo(ctrl), repomocks.NewMockUserProfileRepo(ctrl))
		userRepo.EXPECT().GetByUsername(gomock.Any(), "ivan").Return(nil, errors.New("not found"))
		_, err := svc.Register(context.Background(), "I", "P", "ivan", "pass1", "bad-date", models.Male)
		require.ErrorContains(t, err, "birthday")
	})

	t.Run("too young", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		userRepo := repomocks.NewMockUserAccountRepo(ctrl)
		svc := NewAuthService(userRepo, repomocks.NewMockProfileRepo(ctrl), repomocks.NewMockUserProfileRepo(ctrl))
		userRepo.EXPECT().GetByUsername(gomock.Any(), "kid").Return(nil, errors.New("not found"))
		bday := time.Now().Format("02/01/2006")
		_, err := svc.Register(context.Background(), "I", "P", "kid", "pass1", bday, models.Male)
		require.ErrorContains(t, err, "young")
	})

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		userRepo := repomocks.NewMockUserAccountRepo(ctrl)
		profRepo := repomocks.NewMockProfileRepo(ctrl)
		upRepo := repomocks.NewMockUserProfileRepo(ctrl)
		svc := NewAuthService(userRepo, profRepo, upRepo)
		userRepo.EXPECT().GetByUsername(gomock.Any(), "newuser").Return(nil, errors.New("not found"))
		userRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(int64(1), nil)
		profRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(int64(2), nil)
		upRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(int64(3), nil)
		want := &models.Profile{ID: 2}
		profRepo.EXPECT().Get(gomock.Any(), int64(2)).Return(want, nil)
		got, err := svc.Register(context.Background(), "A", "B", "newuser", "password1", "01/01/1990", models.Male)
		require.NoError(t, err)
		require.Equal(t, want, got)
	})
}

func TestAuthServiceLogin(t *testing.T) {
	t.Parallel()
	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	require.NoError(t, err)

	t.Run("invalid credentials user missing", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		userRepo := repomocks.NewMockUserAccountRepo(ctrl)
		svc := NewAuthService(userRepo, repomocks.NewMockProfileRepo(ctrl), repomocks.NewMockUserProfileRepo(ctrl))
		userRepo.EXPECT().GetByUsername(gomock.Any(), "nobody").Return(nil, errors.New("not found"))
		_, err := svc.Login(context.Background(), "nobody", "secret")
		require.ErrorContains(t, err, "недействительные")
	})

	t.Run("invalid credentials bad password", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		userRepo := repomocks.NewMockUserAccountRepo(ctrl)
		svc := NewAuthService(userRepo, repomocks.NewMockProfileRepo(ctrl), repomocks.NewMockUserProfileRepo(ctrl))
		userRepo.EXPECT().GetByUsername(gomock.Any(), "alex").Return(&models.UserAccount{PasswordHash: string(hash)}, nil)
		_, err := svc.Login(context.Background(), "alex", "wrong")
		require.ErrorContains(t, err, "недействительные")
	})

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		userRepo := repomocks.NewMockUserAccountRepo(ctrl)
		svc := NewAuthService(userRepo, repomocks.NewMockProfileRepo(ctrl), repomocks.NewMockUserProfileRepo(ctrl))
		acc := &models.UserAccount{Username: "alex", PasswordHash: string(hash)}
		userRepo.EXPECT().GetByUsername(gomock.Any(), "alex").Return(acc, nil)
		got, err := svc.Login(context.Background(), "alex", "secret")
		require.NoError(t, err)
		require.Equal(t, acc, got)
	})
}

func TestAuthServiceValidateRegisterStepOne(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	userRepo := repomocks.NewMockUserAccountRepo(ctrl)
	svc := NewAuthService(userRepo, repomocks.NewMockProfileRepo(ctrl), repomocks.NewMockUserProfileRepo(ctrl))

	userRepo.EXPECT().GetByUsername(gomock.Any(), "taken").Return(&models.UserAccount{}, nil)
	m, err := svc.ValidateRegisterStepOne(context.Background(), "taken", "a", "a")
	require.NoError(t, err)
	require.Contains(t, m["login"], "логин")
}

func TestCreateRealUserProfile_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserAccountRepo := mocks.NewMockUserAccountRepo(ctrl)
	mockProfileRepo := mocks.NewMockProfileRepo(ctrl)
	mockUserProfileRepo := mocks.NewMockUserProfileRepo(ctrl)

	service := NewAuthService(mockUserAccountRepo, mockProfileRepo, mockUserProfileRepo) // предположим, что конструктор сервиса такой

	ctx := context.Background()
	passwordHash := "hashedpassword"
	username := "testuser"
	firstName := "John"
	lastName := "Doe"
	var email *string = nil
	var phone *string = nil
	isActive := true
	birthdayDate := time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)
	gender := models.Male
	var avatarID *int64 = nil

	// Настройка моков
	savedUserAccountID := int64(1)
	savedProfileID := int64(2)

	mockUserAccountRepo.EXPECT().
		Save(gomock.Any(), gomock.Any()).
		Return(savedUserAccountID, nil)

	mockProfileRepo.EXPECT().
		Save(gomock.Any(), gomock.Any()).
		Return(savedProfileID, nil)

	mockUserProfileRepo.EXPECT().
		Save(gomock.Any(), gomock.Any()).
		Return(int64(3), nil)

	expectedProfile := &models.Profile{
		ID:        savedProfileID,
		Uid:       uuid.New(),
		AvatarID:  avatarID,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mockProfileRepo.EXPECT().
		Get(gomock.Any(), savedProfileID).
		Return(expectedProfile, nil)

	profile, err := service.CreateRealUserProfile(ctx, passwordHash, username, firstName, lastName, email, phone, isActive, birthdayDate, gender, avatarID)

	require.NoError(t, err)
	assert.Equal(t, expectedProfile, profile)
}

func TestCreateRealUserProfile_UserAccountSaveError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserAccountRepo := mocks.NewMockUserAccountRepo(ctrl)
	mockProfileRepo := mocks.NewMockProfileRepo(ctrl)
	mockUserProfileRepo := mocks.NewMockUserProfileRepo(ctrl)

	service := NewAuthService(mockUserAccountRepo, mockProfileRepo, mockUserProfileRepo)

	ctx := context.Background()
	dbErr := errors.New("db connection error")

	mockUserAccountRepo.EXPECT().
		Save(gomock.Any(), gomock.Any()).
		Return(int64(0), dbErr)

	profile, err := service.CreateRealUserProfile(ctx, "hash", "user", "John", "Doe", nil, nil, true, time.Now(), models.Male, nil)

	assert.Error(t, err)
	assert.EqualError(t, err, dbErr.Error())
	assert.Nil(t, profile)
}

func TestCreateRealUserProfile_ProfileSaveError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserAccountRepo := mocks.NewMockUserAccountRepo(ctrl)
	mockProfileRepo := mocks.NewMockProfileRepo(ctrl)
	mockUserProfileRepo := mocks.NewMockUserProfileRepo(ctrl)

	service := NewAuthService(mockUserAccountRepo, mockProfileRepo, mockUserProfileRepo)

	ctx := context.Background()
	dbErr := errors.New("profile save failed")

	mockUserAccountRepo.EXPECT().
		Save(gomock.Any(), gomock.Any()).
		Return(int64(1), nil)

	mockProfileRepo.EXPECT().
		Save(gomock.Any(), gomock.Any()).
		Return(int64(0), dbErr)

	profile, err := service.CreateRealUserProfile(ctx, "hash", "user", "John", "Doe", nil, nil, true, time.Now(), models.Male, nil)

	assert.Error(t, err)
	assert.EqualError(t, err, dbErr.Error())
	assert.Nil(t, profile)
}

func TestCreateRealUserProfile_UserProfileSaveError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserAccountRepo := mocks.NewMockUserAccountRepo(ctrl)
	mockProfileRepo := mocks.NewMockProfileRepo(ctrl)
	mockUserProfileRepo := mocks.NewMockUserProfileRepo(ctrl)

	service := NewAuthService(mockUserAccountRepo, mockProfileRepo, mockUserProfileRepo)

	ctx := context.Background()
	dbErr := errors.New("user profile save failed")

	mockUserAccountRepo.EXPECT().
		Save(gomock.Any(), gomock.Any()).
		Return(int64(1), nil)

	mockProfileRepo.EXPECT().
		Save(gomock.Any(), gomock.Any()).
		Return(int64(2), nil)

	mockUserProfileRepo.EXPECT().
		Save(gomock.Any(), gomock.Any()).
		Return(int64(0), dbErr)

	profile, err := service.CreateRealUserProfile(ctx, "hash", "user", "John", "Doe", nil, nil, true, time.Now(), models.Male, nil)

	assert.Error(t, err)
	assert.EqualError(t, err, dbErr.Error())
	assert.Nil(t, profile)
}

func TestCreateRealUserProfile_ProfileGetError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserAccountRepo := mocks.NewMockUserAccountRepo(ctrl)
	mockProfileRepo := mocks.NewMockProfileRepo(ctrl)
	mockUserProfileRepo := mocks.NewMockUserProfileRepo(ctrl)

	service := NewAuthService(mockUserAccountRepo, mockProfileRepo, mockUserProfileRepo)

	ctx := context.Background()
	dbErr := errors.New("profile not found after save")

	mockUserAccountRepo.EXPECT().
		Save(gomock.Any(), gomock.Any()).
		Return(int64(1), nil)

	mockProfileRepo.EXPECT().
		Save(gomock.Any(), gomock.Any()).
		Return(int64(2), nil)

	mockUserProfileRepo.EXPECT().
		Save(gomock.Any(), gomock.Any()).
		Return(int64(3), nil)

	mockProfileRepo.EXPECT().
		Get(gomock.Any(), int64(2)).
		Return(nil, dbErr)

	profile, err := service.CreateRealUserProfile(ctx, "hash", "user", "John", "Doe", nil, nil, true, time.Now(), models.Male, nil)

	assert.Error(t, err)
	assert.EqualError(t, err, dbErr.Error())
	assert.Nil(t, profile)
}
