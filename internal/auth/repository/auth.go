package repository

import (
	legacyprofile "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/profile"
	legacysession "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/session"
	legacyaccount "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/user_account"
	legacyuserprofile "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/user_profile"
)

type Store struct {
	Accounts     legacyaccount.UserAccountRepo
	Profiles     legacyprofile.ProfileRepo
	UserProfiles legacyuserprofile.UserProfileRepo
	Sessions     legacysession.SessionRepo
}

func NewStore(
	accounts legacyaccount.UserAccountRepo,
	profiles legacyprofile.ProfileRepo,
	userProfiles legacyuserprofile.UserProfileRepo,
	sessions legacysession.SessionRepo,
) Store {
	return Store{
		Accounts:     accounts,
		Profiles:     profiles,
		UserProfiles: userProfiles,
		Sessions:     sessions,
	}
}
