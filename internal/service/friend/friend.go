package friend

import (
	"context"
	"errors"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/friend"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/dto"
)

type friendshipService struct {
	friendshipRepo friend.FriendshipRepo
}

type FriendshipService interface {
	GetFriends(ctx context.Context, profileID int64, status models.FriendshipStatus) ([]dto.FriendDTO, error)
	CheckFriendship(ctx context.Context, profileID, friendID int64) (bool, models.FriendshipStatus, error)
	CheckFriendshipBy(ctx context.Context, profileID, friendID int64) (bool, models.FriendshipStatus, error)
	DeleteFriend(ctx context.Context, profileID, friendID int64) error
	GetOutgoingFriends(ctx context.Context, profileID int64, status string) ([]dto.FriendDTO, error)
	GetIncomingFriends(ctx context.Context, profileID int64, status string) ([]dto.FriendDTO, error)
	MakeFriends(ctx context.Context, profileID, friendID int64) error
	AcceptFriendship(ctx context.Context, profileID1, profileID2 int64) error
	DeclineFriendship(ctx context.Context, profileID1, profileID2 int64) error
	RevokeFriendRequest(ctx context.Context, profileID, friendID int64) error
}

func NewFriendshipService(friendshipRepo friend.FriendshipRepo) FriendshipService {
	return &friendshipService{
		friendshipRepo: friendshipRepo,
	}
}

func (s *friendshipService) GetFriends(ctx context.Context, profileID int64, status models.FriendshipStatus) ([]dto.FriendDTO, error) {
	return s.friendshipRepo.GetFriends(ctx, profileID, status)
}

func (s *friendshipService) CheckFriendship(ctx context.Context, profileID1, profileID2 int64) (bool, models.FriendshipStatus, error) {
	status, err := s.friendshipRepo.GetFriendshipStatus(ctx, profileID1, profileID2)
	if err != nil {
		if errors.Is(err, xerrors.FriendshipNotFound) {
			return false, "", nil
		}
		return false, "", err
	}

	if status != "" {
		return true, models.FriendshipStatus(status), nil
	}

	return false, "", nil
}

func (s *friendshipService) CheckFriendshipBy(ctx context.Context, profileID, friendID int64) (bool, models.FriendshipStatus, error) {
	status, err := s.friendshipRepo.GetFriendshipStatusBy(ctx, profileID, friendID)
	if err != nil {
		if errors.Is(err, xerrors.FriendshipNotFound) {
			return false, "", nil
		}
		return false, "", err
	}

	if status != "" {
		return true, models.FriendshipStatus(status), nil
	}

	return false, "", nil
}

func (s *friendshipService) DeleteFriend(ctx context.Context, profileID, friendID int64) error {
	return s.friendshipRepo.DeleteFriend(ctx, profileID, friendID)
}

func (s *friendshipService) GetIncomingFriends(ctx context.Context, profileID int64, status string) ([]dto.FriendDTO, error) {
	return s.friendshipRepo.GetIncomingFriends(ctx, profileID, status)
}

func (s *friendshipService) GetOutgoingFriends(ctx context.Context, profileID int64, status string) ([]dto.FriendDTO, error) {
	return s.friendshipRepo.GetOutgoingFriends(ctx, profileID, status)
}

func (s *friendshipService) MakeFriends(ctx context.Context, profileID, friendID int64) error {

	// Ранее никто из них не был друзьями
	if areFriends, _, err := s.CheckFriendship(ctx, profileID, friendID); !areFriends {
		if err != nil {
			return err
		}
		err := s.friendshipRepo.Create(ctx, profileID, friendID, string(models.FriendshipPending))
		if err != nil {
			return err
		}
		return nil
	}

	// Если есть встречный запрос в друзья - автоматически принимается
	if areFriends, status, err := s.CheckFriendshipBy(ctx, friendID, profileID); areFriends && status == models.FriendshipPending {
		if err != nil {
			return err
		}
		err := s.AcceptFriendship(ctx, profileID, friendID)
		if err != nil {
			return err
		}
		return nil
	}

	// Если этот пользователь уже отправлял заявку
	if areFriends, _, err := s.CheckFriendshipBy(ctx, profileID, friendID); areFriends {
		if err != nil {
			return err
		}
		return xerrors.AllreadyExists
	}

	return xerrors.InternalServerError
}

func (s *friendshipService) AcceptFriendship(ctx context.Context, profileID1, profileID2 int64) error {

	areFriends, status, err := s.CheckFriendshipBy(ctx, profileID1, profileID2)
	if err != nil {
		return err
	}

	if areFriends && status == models.FriendshipPending {
		err := s.friendshipRepo.AcceptFriendship(ctx, profileID1, profileID2)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *friendshipService) DeclineFriendship(ctx context.Context, profileID1, profileID2 int64) error {

	areFriends, status, err := s.CheckFriendshipBy(ctx, profileID1, profileID2)
	if err != nil {
		return err
	}

	if areFriends && status == models.FriendshipPending {
		err := s.friendshipRepo.DeclineFriendship(ctx, profileID1, profileID2)
		if err != nil {
			return err
		}
		return nil
	}

	return xerrors.FriendshipNotFound
}

func (s *friendshipService) RevokeFriendRequest(ctx context.Context, profileID, friendID int64) error {
	return s.friendshipRepo.RevokeFriendRequest(ctx, profileID, friendID)
}
