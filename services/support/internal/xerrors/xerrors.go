package xerrors

import "errors"

const (
	InternalServerErrorStr = "Internal server error"
	InvalidRequestBody     = "Invalid request body"
	InvalidRequest         = "Invalid request"
	InvalidID              = "Invlalid ID parameter"
)

var (
	ProfileNotFound        = errors.New("profile not found")
	UserProfileNotFound    = errors.New("user profile not found")
	UserAccountNotFound    = errors.New("user account not found")
	MediaNotFound          = errors.New("media not found")
	SupportTicketNotFound  = errors.New("support ticket not found")
	SupportForbidden       = errors.New("support access forbidden")
	ErrNothingToUpdate     = errors.New("no fields provided for update")
	NoRowsAffected         = errors.New("affected on 0 rows")
	MultipleRowsAffect     = errors.New("Affected not on 1 row")
	UnsupportedContentType = errors.New("unsupported content type")
	AllreadyExists         = errors.New("Resource already exists")
)
