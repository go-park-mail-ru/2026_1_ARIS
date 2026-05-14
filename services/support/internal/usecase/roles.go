package usecase

import (
	"context"
	"errors"

	"github.com/go-park-mail-ru/2026_1_ARIS/services/support/internal/model"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/support/internal/repository"
)

var (
	ErrInvalidInput       = errors.New("invalid input")
	ErrInvalidSupportRole = errors.New("invalid support role")
)

type Service struct {
	roles repository.ProfileRoleRepo
}

func New(roles repository.ProfileRoleRepo) *Service {
	return &Service{roles: roles}
}

func (s *Service) SetProfileRole(ctx context.Context, profileID int64, role model.SupportRole) error {
	if profileID <= 0 {
		return ErrInvalidInput
	}
	if !isElevatedRole(role) {
		return ErrInvalidSupportRole
	}
	return s.roles.SetProfileRole(ctx, profileID, role)
}

func (s *Service) GetProfileRole(ctx context.Context, profileID int64) (model.SupportRole, error) {
	if profileID <= 0 {
		return model.SupportRoleUser, ErrInvalidInput
	}

	role, err := s.roles.GetProfileRole(ctx, profileID)
	if err != nil {
		if errors.Is(err, repository.ErrSupportRoleNotFound) {
			return model.SupportRoleUser, nil
		}
		return model.SupportRoleUser, err
	}
	return role.Role, nil
}

func (s *Service) IsSupportAgent(ctx context.Context, profileID int64) (bool, error) {
	role, err := s.GetProfileRole(ctx, profileID)
	if err != nil {
		return false, err
	}
	return isElevatedRole(role), nil
}

func (s *Service) IsAdmin(ctx context.Context, profileID int64) (bool, error) {
	role, err := s.GetProfileRole(ctx, profileID)
	if err != nil {
		return false, err
	}
	return role == model.SupportRoleAdmin, nil
}

func isElevatedRole(role model.SupportRole) bool {
	return role == model.SupportRoleAdmin || role == model.SupportRoleSupportL1 || role == model.SupportRoleSupportL2
}
