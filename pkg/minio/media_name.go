package minioclient

import (
	"fmt"

	"github.com/google/uuid"
)

func GenerageMedaName(mediaUUID uuid.UUID, mediaSize int64, extension string) string {
	return fmt.Sprintf("%s-%d%s", mediaUUID.String(), mediaSize, extension)
}
