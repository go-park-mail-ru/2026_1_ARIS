package model

type SupportRole string

const (
	SupportRoleUser      SupportRole = "user"
	SupportRoleSupportL1 SupportRole = "support_l1"
	SupportRoleSupportL2 SupportRole = "support_l2"
	SupportRoleAdmin     SupportRole = "admin"
)

type SupportProfileRole struct {
	ProfileID int64       `db:"profile_id"`
	Role      SupportRole `db:"role"`
}
