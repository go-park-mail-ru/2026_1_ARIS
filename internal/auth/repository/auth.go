package repository

import (
	legacyprofile "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/profile"
	legacysession "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/session"
	legacyaccount "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/user_account"
	legacyuserprofile "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/user_profile"
)

type AuthRepo struct {
	Accounts     legacyaccount.UserAccountRepo
	Profiles     legacyprofile.ProfileRepo
	UserProfiles legacyuserprofile.UserProfileRepo
	Sessions     legacysession.SessionRepo
}

type Store = AuthRepo

func NewAuthRepo(
	accounts legacyaccount.UserAccountRepo,
	profiles legacyprofile.ProfileRepo,
	userProfiles legacyuserprofile.UserProfileRepo,
	sessions legacysession.SessionRepo,
) AuthRepo {
	return AuthRepo{
		Accounts:     accounts,
		Profiles:     profiles,
		UserProfiles: userProfiles,
		Sessions:     sessions,
	}
}

func NewStore(
	accounts legacyaccount.UserAccountRepo,
	profiles legacyprofile.ProfileRepo,
	userProfiles legacyuserprofile.UserProfileRepo,
	sessions legacysession.SessionRepo,
) Store {
	return NewAuthRepo(accounts, profiles, userProfiles, sessions)
}
