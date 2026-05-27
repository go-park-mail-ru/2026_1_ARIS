package http

//go:generate mockgen -source=service.go -destination=mocks/service_mock.go -package=mocks

import (
	"context"

	"github.com/go-park-mail-ru/2026_1_ARIS/services/community/internal/model"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/community/internal/usecase"
)

type CommunityService interface {
	Create(context.Context, int64, usecase.CreateInput) (*usecase.Details, error)
	List(context.Context, int, int, *int64) ([]usecase.Details, error)
	GetDetails(context.Context, int64, *int64) (*usecase.Details, error)
	GetDetailsByProfileID(context.Context, int64, *int64) (*usecase.Details, error)
	Update(context.Context, int64, int64, usecase.UpdateInput) (*usecase.Details, error)
	CheckExists(context.Context, usecase.CheckExistsInput) (*usecase.CheckExistsResult, error)
	Delete(context.Context, int64, int64) error
	ListMembers(context.Context, int64, *int64, bool) ([]usecase.MemberDetails, error)
	Join(context.Context, int64, int64) (*usecase.MemberDetails, error)
	Leave(context.Context, int64, int64) error
	RemoveMember(context.Context, int64, int64, int64) error
	ChangeMemberRole(context.Context, int64, int64, int64, model.CommunityMemberRole) (*usecase.MemberDetails, error)
}
