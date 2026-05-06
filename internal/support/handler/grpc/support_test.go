package grpc

import (
	"context"
	"errors"
	"testing"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	supportmock "github.com/go-park-mail-ru/2026_1_ARIS/internal/support/repository/mock"
	supportservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/support/service"
	supportpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/support"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGetProfileRole(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := supportmock.NewMockTicketRepository(ctrl)
	svc := supportservice.NewTicketService(repo, nil)
	server := New(svc)

	repo.EXPECT().GetProfileRole(gomock.Any(), int64(10)).Return(&models.SupportProfileRole{ProfileID: 10, Role: models.SupportRoleAdmin}, nil)
	resp, err := server.GetProfileRole(context.Background(), &supportpb.GetProfileRoleRequest{ProfileId: 10})
	require.NoError(t, err)
	require.Equal(t, string(models.SupportRoleAdmin), resp.Role)

	_, err = server.GetProfileRole(context.Background(), &supportpb.GetProfileRoleRequest{ProfileId: 0})
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	repo.EXPECT().GetProfileRole(gomock.Any(), int64(11)).Return(nil, errors.New("db"))
	_, err = server.GetProfileRole(context.Background(), &supportpb.GetProfileRoleRequest{ProfileId: 11})
	require.Equal(t, codes.Internal, status.Code(err))
}
