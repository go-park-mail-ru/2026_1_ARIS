package cursor

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Cursor struct {
	CreatedAt time.Time
	ID        int64
}

func Encode(c Cursor) string {
	raw := fmt.Sprintf("%s_%d", c.CreatedAt.UTC().Format(time.RFC3339Nano), c.ID)
	return base64.StdEncoding.EncodeToString([]byte(raw))
}

func Decode(str string) (Cursor, error) {
	b, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return Cursor{}, err
	}

	parts := strings.SplitN(string(b), "_", 2)
	if len(parts) != 2 {
		return Cursor{}, errors.New("invalid cursor format")
	}

	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return Cursor{}, errors.New("invalid cursor timestamp")
	}

	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return Cursor{}, errors.New("invalid cursor id")
	}

	return Cursor{ID: id, CreatedAt: t}, nil
}
