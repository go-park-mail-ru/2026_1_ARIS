package xerrors

import "errors"

// 404
var ProfileNotFound = errors.New("profile not found")
var UserProfileNotFound = errors.New("user profile not found")
var UserAccountNotFound = errors.New("user account not found")
var SessionNotFound = errors.New("session not found")
var FriendshipNotFound = errors.New("friendship not found")

var NoRowsAffected = errors.New("affected on 0 rows")

var AllreadyExists = errors.New("Resource already exists")

var InternalServerError = errors.New(InternalServerErrorStr)

var MultipleRowsAffect = errors.New("Affected not on 1 row")
