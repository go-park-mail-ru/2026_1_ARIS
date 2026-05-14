package service

import (
	"context"
	"errors"

	servicedto "github.com/go-park-mail-ru/2026_1_ARIS/internal/dto"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
)

var (
	ErrInvalidStatus       = errors.New("unknown status value")
	ErrFriendshipNotFound  = xerrors.FriendshipNotFound
	ErrAlreadyFriends      = xerrors.AllreadyExists
	ErrFriendshipNotExists = xerrors.NoRowsAffected
)

func (s *Service) GetFriends(ctx context.Context, userAccountID int64, status models.FriendshipStatus) ([]servicedto.FriendDTO, error) {
	profileID, err := s.currentProfileID(ctx, userAccountID)
	if err != nil {
		return nil, err
	}
	if !validFriendshipStatus(string(status)) {
		return nil, ErrInvalidStatus
	}
	return s.store.Friendships.GetFriends(ctx, profileID, status)
}

func (s *Service) GetUsersFriends(ctx context.Context, profileID int64) ([]servicedto.FriendDTO, error) {
	if profileID <= 0 {
		return nil, ErrInvalidInput
	}
	if _, err := s.store.Profiles.Get(ctx, profileID); err != nil {
		return nil, normalizeProfileError(err)
	}
	return s.store.Friendships.GetFriends(ctx, profileID, models.FriendshipAccepted)
}

func (s *Service) DeleteFriend(ctx context.Context, userAccountID int64, friendID int64) error {
	profileID, err := s.currentProfileID(ctx, userAccountID)
	if err != nil {
		return err
	}
	if friendID <= 0 {
		return ErrInvalidInput
	}
	return normalizeFriendshipMutationError(s.store.Friendships.DeleteFriend(ctx, profileID, friendID))
}

func (s *Service) RequestFriendship(ctx context.Context, userAccountID int64, friendID int64) error {
	profileID, err := s.currentProfileID(ctx, userAccountID)
	if err != nil {
		return err
	}
	if friendID <= 0 || friendID == profileID {
		return ErrInvalidInput
	}
	if _, err := s.store.Profiles.Get(ctx, friendID); err != nil {
		return normalizeProfileError(err)
	}

	if exists, status, err := s.checkFriendshipBy(ctx, friendID, profileID); err != nil {
		return err
	} else if exists {
		if status == models.FriendshipPending {
			return normalizeFriendshipMutationError(s.store.Friendships.AcceptFriendship(ctx, friendID, profileID))
		}
		return ErrAlreadyFriends
	}

	if exists, _, err := s.checkFriendshipBy(ctx, profileID, friendID); err != nil {
		return err
	} else if exists {
		return ErrAlreadyFriends
	}

	return normalizeFriendshipMutationError(s.store.Friendships.Create(ctx, profileID, friendID, string(models.FriendshipPending)))
}

func (s *Service) GetIncomingFriendRequests(ctx context.Context, userAccountID int64, status string) ([]servicedto.FriendDTO, error) {
	profileID, err := s.currentProfileID(ctx, userAccountID)
	if err != nil {
		return nil, err
	}
	if !validFriendshipStatus(status) {
		return nil, ErrInvalidStatus
	}
	return s.store.Friendships.GetIncomingFriends(ctx, profileID, status)
}

func (s *Service) GetOutgoingFriendRequests(ctx context.Context, userAccountID int64, status string) ([]servicedto.FriendDTO, error) {
	profileID, err := s.currentProfileID(ctx, userAccountID)
	if err != nil {
		return nil, err
	}
	if !validFriendshipStatus(status) {
		return nil, ErrInvalidStatus
	}
	return s.store.Friendships.GetOutgoingFriends(ctx, profileID, status)
}

func (s *Service) AcceptFriendRequest(ctx context.Context, userAccountID int64, requesterID int64) error {
	profileID, err := s.currentProfileID(ctx, userAccountID)
	if err != nil {
		return err
	}
	if requesterID <= 0 || requesterID == profileID {
		return ErrInvalidInput
	}
	if exists, status, err := s.checkFriendshipBy(ctx, requesterID, profileID); err != nil {
		return err
	} else if !exists || status != models.FriendshipPending {
		return ErrFriendshipNotExists
	}
	return normalizeFriendshipMutationError(s.store.Friendships.AcceptFriendship(ctx, requesterID, profileID))
}

func (s *Service) DeclineFriendRequest(ctx context.Context, userAccountID int64, requesterID int64) error {
	profileID, err := s.currentProfileID(ctx, userAccountID)
	if err != nil {
		return err
	}
	if requesterID <= 0 || requesterID == profileID {
		return ErrInvalidInput
	}
	if exists, status, err := s.checkFriendshipBy(ctx, requesterID, profileID); err != nil {
		return err
	} else if !exists || status != models.FriendshipPending {
		return ErrFriendshipNotExists
	}
	return normalizeFriendshipMutationError(s.store.Friendships.DeclineFriendship(ctx, requesterID, profileID))
}

func (s *Service) RevokeFriendRequest(ctx context.Context, userAccountID int64, addresseeID int64) error {
	profileID, err := s.currentProfileID(ctx, userAccountID)
	if err != nil {
		return err
	}
	if addresseeID <= 0 || addresseeID == profileID {
		return ErrInvalidInput
	}
	return normalizeFriendshipMutationError(s.store.Friendships.RevokeFriendRequest(ctx, profileID, addresseeID))
}

func (s *Service) currentProfileID(ctx context.Context, userAccountID int64) (int64, error) {
	if userAccountID <= 0 {
		return 0, ErrInvalidInput
	}
	profile, err := s.store.Profiles.GetByUserAccountID(ctx, userAccountID)
	if err != nil {
		return 0, normalizeProfileError(err)
	}
	return profile.ID, nil
}

func (s *Service) checkFriendshipBy(ctx context.Context, profileID, friendID int64) (bool, models.FriendshipStatus, error) {
	status, err := s.store.Friendships.GetFriendshipStatusBy(ctx, profileID, friendID)
	if err != nil {
		if errors.Is(err, xerrors.FriendshipNotFound) {
			return false, "", nil
		}
		return false, "", err
	}
	if status == "" {
		return false, "", nil
	}
	return true, models.FriendshipStatus(status), nil
}

func validFriendshipStatus(status string) bool {
	return status == string(models.FriendshipPending) || status == string(models.FriendshipAccepted)
}

func normalizeFriendshipMutationError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, xerrors.NoRowsAffected), errors.Is(err, xerrors.FriendshipNotFound):
		return ErrFriendshipNotExists
	case errors.Is(err, xerrors.AllreadyExists):
		return ErrAlreadyFriends
	default:
		return err
	}
}
