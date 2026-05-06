package repository

import (
	legacyfriend "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/friend"
	legacyprofile "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/profile"
	legacysettings "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/settings"
	legacyaccount "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/user_account"
	legacyuserprofile "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/user_profile"
)

type Store struct {
	Accounts     legacyaccount.UserAccountRepo
	Profiles     legacyprofile.ProfileRepo
	UserProfiles legacyuserprofile.UserProfileRepo
	Settings     legacysettings.UserSettingsRepository
	Friendships  legacyfriend.FriendshipRepo
}

func NewStore(
	accounts legacyaccount.UserAccountRepo,
	profiles legacyprofile.ProfileRepo,
	userProfiles legacyuserprofile.UserProfileRepo,
	settings legacysettings.UserSettingsRepository,
	friendships legacyfriend.FriendshipRepo,
) Store {
	return Store{
		Accounts:     accounts,
		Profiles:     profiles,
		UserProfiles: userProfiles,
		Settings:     settings,
		Friendships:  friendships,
	}
}
