package usecase

import (
	"context"
	"testing"

	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/game/internal/model"
	"github.com/golang/mock/gomock"
)

func TestRoomMutationMethods(t *testing.T) {
	ctx := context.Background()
	const (
		userAccountID = int64(5)
		profileID     = int64(50)
		roomID        = int64(7)
	)

	t.Run("set ready", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		svc, rooms, members, _, _, _, _, users := newGameService(ctrl)
		users.EXPECT().GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: userAccountID}).
			Return(&userpb.GetProfileByUserAccountResponse{ProfileId: profileID}, nil)
		rooms.EXPECT().GetForUpdate(gomock.Any(), roomID).Return(&model.Room{ID: roomID, Status: model.RoomStatusWaiting}, nil)
		members.EXPECT().SetReady(gomock.Any(), roomID, profileID, true).Return(nil)

		if err := svc.SetReady(ctx, userAccountID, roomID, true); err != nil {
			t.Fatalf("SetReady() error = %v", err)
		}
	})

	t.Run("kick player", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		svc, rooms, members, _, _, _, _, users := newGameService(ctrl)
		users.EXPECT().GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: userAccountID}).
			Return(&userpb.GetProfileByUserAccountResponse{ProfileId: profileID}, nil)
		rooms.EXPECT().GetForUpdate(gomock.Any(), roomID).Return(&model.Room{ID: roomID, Status: model.RoomStatusWaiting, CreatedByProfileID: profileID}, nil)
		members.EXPECT().Deactivate(gomock.Any(), roomID, int64(99)).Return(nil)

		if err := svc.KickPlayer(ctx, userAccountID, roomID, 99); err != nil {
			t.Fatalf("KickPlayer() error = %v", err)
		}
	})

	t.Run("update room title", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		svc, rooms, _, _, _, _, _, users := newGameService(ctrl)
		users.EXPECT().GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: userAccountID}).
			Return(&userpb.GetProfileByUserAccountResponse{ProfileId: profileID}, nil)
		rooms.EXPECT().GetForUpdate(gomock.Any(), roomID).Return(&model.Room{ID: roomID, Title: "Old", Status: model.RoomStatusWaiting, CreatedByProfileID: profileID}, nil)
		rooms.EXPECT().UpdateTitle(gomock.Any(), roomID, "New title").Return(nil)

		if err := svc.UpdateRoomTitle(ctx, userAccountID, roomID, "  New title  "); err != nil {
			t.Fatalf("UpdateRoomTitle() error = %v", err)
		}
	})

	t.Run("update ranked clears ready", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		svc, rooms, members, _, _, _, _, users := newGameService(ctrl)
		users.EXPECT().GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: userAccountID}).
			Return(&userpb.GetProfileByUserAccountResponse{ProfileId: profileID}, nil)
		rooms.EXPECT().GetForUpdate(gomock.Any(), roomID).Return(&model.Room{ID: roomID, Status: model.RoomStatusWaiting, CreatedByProfileID: profileID}, nil)
		rooms.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
		members.EXPECT().ClearReady(gomock.Any(), roomID).Return(nil)

		if err := svc.UpdateRoomRanked(ctx, userAccountID, roomID, true); err != nil {
			t.Fatalf("UpdateRoomRanked() error = %v", err)
		}
	})

	t.Run("assign admin", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		svc, rooms, members, _, _, _, _, users := newGameService(ctrl)
		users.EXPECT().GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: userAccountID}).
			Return(&userpb.GetProfileByUserAccountResponse{ProfileId: profileID}, nil)
		rooms.EXPECT().GetForUpdate(gomock.Any(), roomID).Return(&model.Room{ID: roomID, Status: model.RoomStatusWaiting, CreatedByProfileID: profileID}, nil)
		members.EXPECT().IsMember(gomock.Any(), roomID, int64(99)).Return(true, nil)
		rooms.EXPECT().UpdateAdmin(gomock.Any(), roomID, int64(99)).Return(nil)

		if err := svc.AssignAdmin(ctx, userAccountID, roomID, 99); err != nil {
			t.Fatalf("AssignAdmin() error = %v", err)
		}
	})

	t.Run("touch waiting member", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		svc, _, members, _, _, _, _, users := newGameService(ctrl)
		users.EXPECT().GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: userAccountID}).
			Return(&userpb.GetProfileByUserAccountResponse{ProfileId: profileID}, nil)
		members.EXPECT().TouchWaiting(gomock.Any(), roomID, profileID).Return(nil)

		if err := svc.TouchWaitingRoomMember(ctx, userAccountID, roomID); err != nil {
			t.Fatalf("TouchWaitingRoomMember() error = %v", err)
		}
	})
}
