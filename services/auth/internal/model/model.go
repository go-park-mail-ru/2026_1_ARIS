package model

import "time"

type Gender string

const (
	Male   Gender = "male"
	Female Gender = "female"
)

type SupportRole string

const (
	SupportRoleUser      SupportRole = "user"
	SupportRoleSupportL1 SupportRole = "support_l1"
	SupportRoleSupportL2 SupportRole = "support_l2"
	SupportRoleAdmin     SupportRole = "admin"
)

type SessionID string

type Session struct {
	SessionID SessionID `json:"id"`
	UserID    int64     `json:"user"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiredAt time.Time `json:"expiredAt"`
}
