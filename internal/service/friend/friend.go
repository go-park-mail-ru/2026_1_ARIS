package friend

import (
	"context"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/friend"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/dto"
)

type friendshipService struct {
	friendshipRepo friend.FriendshipRepo
}

type FriendshipService interface {
	GetFriends(ctx context.Context, profileID int64, status models.FriendshipStatus) ([]dto.FriendDTO, error)
	CheckFriendship(ctx context.Context, profileID, friendID int64) (bool, models.FriendshipStatus)
	CheckFriendshipBy(ctx context.Context, profileID, friendID int64) (bool, models.FriendshipStatus)
	DeleteFriend(ctx context.Context, profileID, friendID int64) error
	GetOutgoingFriends(ctx context.Context, profileID int64, status string) ([]dto.FriendDTO, error)
	GetIncomingFriends(ctx context.Context, profileID int64, status string) ([]dto.FriendDTO, error)
	MakeFriends(ctx context.Context, profileID, friendID int64, status models.FriendshipStatus) error
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

func (s *friendshipService) CheckFriendship(ctx context.Context, profileID1, profileID2 int64) (bool, models.FriendshipStatus) {
	status, err := s.friendshipRepo.GetFriendshipStatus(ctx, profileID1, profileID2)
	if err != nil {
		return false, ""
	}

	if status != "" {
		return true, models.FriendshipStatus(status)
	}

	return false, ""
}

func (s *friendshipService) CheckFriendshipBy(ctx context.Context, profileID, friendID int64) (bool, models.FriendshipStatus) {
	status, err := s.friendshipRepo.GetFriendshipStatusBy(ctx, profileID, friendID)
	if err != nil {
		return false, ""
	}

	if status != "" {
		return true, models.FriendshipStatus(status)
	}

	return false, ""
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

func (s *friendshipService) MakeFriends(ctx context.Context, profileID, friendID int64, status models.FriendshipStatus) error {
	return s.friendshipRepo.Create(ctx, profileID, friendID, string(status))
}

func (s *friendshipService) AcceptFriendship(ctx context.Context, profileID1, profileID2 int64) error {
	return s.friendshipRepo.AcceptFriendship(ctx, profileID1, profileID2)
}

func (s *friendshipService) DeclineFriendship(ctx context.Context, profileID1, profileID2 int64) error {
	return s.friendshipRepo.DeclineFriendship(ctx, profileID1, profileID2)
}

func (s *friendshipService) RevokeFriendRequest(ctx context.Context, profileID, friendID int64) error {
	return s.friendshipRepo.RevokeFriendRequest(ctx, profileID, friendID)
}
