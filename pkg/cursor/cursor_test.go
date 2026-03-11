package cursor

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestEncodeDecode(t *testing.T) {
	c := Cursor{
		CreatedAt: time.Now().Truncate(time.Millisecond),
		ID:        uuid.New(),
	}
	encoded := Encode(c)
	decoded, err := Decode(encoded)
	assert.NoError(t, err)
	assert.Equal(t, c.CreatedAt.Unix(), decoded.CreatedAt.Unix()) // точност до секунды
	assert.Equal(t, c.ID, decoded.ID)
}

func TestDecodeInvalid(t *testing.T) {
	_, err := Decode("invalid-base64")
	assert.Error(t, err)
}
