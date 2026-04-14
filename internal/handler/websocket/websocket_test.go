package websocket

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewWebSocketHandler(t *testing.T) {
	handler := NewWebSocketHandler(nil, nil)
	assert.NotNil(t, handler)
}
