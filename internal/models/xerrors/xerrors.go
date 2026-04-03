package xerrors

import "errors"

var ProfileNotFound = errors.New("profile not found")
var UserProfileNotFound = errors.New("user profile not found")
var UserAccountNotFound = errors.New("user account not found")
var SessionNotFound = errors.New("session not found")
var FriendshipNotFound = errors.New("friendship not found")

var NoRowsAffected = errors.New("affected on 0 rows")

var NothingDeleted = errors.New("nothing deleted")
var NothingInserted = errors.New("nothing inserted")
var NothingUpdated = errors.New("nothing updated")
