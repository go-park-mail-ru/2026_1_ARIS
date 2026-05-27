package grpc

import (
	"context"
	"errors"
	"testing"

	supportpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/support"
	models "github.com/go-park-mail-ru/2026_1_ARIS/services/support/internal/model"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/support/internal/usecase"
	ticketmocks "github.com/go-park-mail-ru/2026_1_ARIS/services/support/internal/usecase/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestSupportGRPCGetProfileRole(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	svc := ticketmocks.NewMockTicketService(ctrl)
	server := New(svc)
	svc.EXPECT().GetProfileRole(gomock.Any(), int64(10)).Return(models.SupportRoleAdmin, nil)

	resp, err := server.GetProfileRole(context.Background(), &supportpb.GetProfileRoleRequest{ProfileId: 10})

	require.NoError(t, err)
	require.Equal(t, string(models.SupportRoleAdmin), resp.GetRole())
}

func TestSupportGRPCToStatus(t *testing.T) {
	t.Parallel()

	require.Equal(t, codes.InvalidArgument, status.Code(toStatus(usecase.ErrInvalidInput)))
	require.Equal(t, codes.Internal, status.Code(toStatus(errors.New("boom"))))
}
