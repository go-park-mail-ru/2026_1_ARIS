package grpc

import (
	"errors"
	"testing"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/media/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestToStatus(t *testing.T) {
	require.Equal(t, codes.InvalidArgument, status.Code(toStatus(service.ErrInvalidInput)))
	require.Equal(t, codes.NotFound, status.Code(toStatus(service.ErrMediaNotFound)))
	require.Equal(t, codes.Internal, status.Code(toStatus(errors.New("boom"))))
	require.NotNil(t, New(nil))
}
