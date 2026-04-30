package media

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestObjectPublicPath(t *testing.T) {
	t.Parallel()

	require.Equal(t, "/media/file.png", objectPublicPath("media", "file.png"))
	require.Equal(t, "/media/nested/file.png", objectPublicPath("/media/", "/nested/file.png"))
}
