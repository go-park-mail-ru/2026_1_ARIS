package cursor

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Cursor struct {
	ID        uuid.UUID
	CreatedAt time.Time
}

func Encode(cursor Cursor) string {
	str := fmt.Sprintf("%s_%s", cursor.CreatedAt.UTC().Format(time.RFC3339Nano), cursor.ID.String())
	return str
}

func Decode(str string) (Cursor, error) {
	c, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		fmt.Println("Cursor decoding error")
		return Cursor{}, err
	}

	parts := strings.SplitN(string(c), "_", 2)

	if len(parts) != 2 {
		return Cursor{}, errors.New("Can't decode cursor")
	}

	id, err := uuid.Parse(parts[0])
	if err != nil {
		fmt.Println("Can't parse cursor id")
		return Cursor{}, err
	}

	t, err := time.Parse(time.RFC3339Nano, parts[1])
	if err != nil {
		fmt.Println("Can't parse cursor CreatedAt")
		return Cursor{}, err
	}

	return Cursor{ID: id, CreatedAt: t}, nil
}
