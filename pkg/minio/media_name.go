package xminio

import (
	"fmt"

	"github.com/google/uuid"
)

func GenerateMediaName(mediaUUID uuid.UUID, _ int64, extension string) string {
	return fmt.Sprintf("%s%s", mediaUUID.String(), extension)
}
