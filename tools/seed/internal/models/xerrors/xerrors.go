package xerrors

import "errors"

// 404
var ProfileNotFound = errors.New("profile not found")
var UserProfileNotFound = errors.New("user profile not found")
var UserAccountNotFound = errors.New("user account not found")
var SessionNotFound = errors.New("session not found")
var FriendshipNotFound = errors.New("friendship not found")
var MediaNotFound = errors.New("media not found")
var PostNotFound = errors.New("post not found")
var SupportTicketNotFound = errors.New("support ticket not found")
var ErrUserSettingsNotFound = errors.New("user settings not found")

var ErrNothingToUpdate = errors.New("no fields provided for update")

var NoRowsAffected = errors.New("affected on 0 rows")

var SupportForbidden = errors.New("support access forbidden")

var AllreadyExists = errors.New("Resource already exists")

var InternalServerError = errors.New(InternalServerErrorStr)

var MultipleRowsAffect = errors.New("Affected not on 1 row")

var UnsupportedContentType = errors.New("unsupported content type")
