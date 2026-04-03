package media

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/media"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/session"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils"
	minioclient "github.com/go-park-mail-ru/2026_1_ARIS/pkg/minio"
)

type MediaHandler struct {
	mediaService   media.MediaService
	sessionService session.SessionService
}

func NewMediaHandler(mediaService media.MediaService, sessionService session.SessionService, minioClient minioclient.MinioClient) *MediaHandler {
	return &MediaHandler{
		mediaService:   mediaService,
		sessionService: sessionService,
	}
}

type fileResponse struct {
	Files []string `json:"files"`
}

func (h *MediaHandler) SaveFiles(w http.ResponseWriter, r *http.Request) {
	// cookie, err := r.Cookie("session_id")
	// if err != nil {
	// 	utils.WriteError(w, "Unauthorized", http.StatusUnauthorized)
	// 	return
	// }

	// session, err := h.sessionService.Get(r.Context(), models.SessionID(cookie.Value))
	// if err != nil {
	// 	utils.WriteError(w, "Unauthorized", http.StatusUnauthorized)
	// 	return
	// }

	// _ = session

	// Вынести это в env           ||
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		utils.WriteError(w, "Parsing form error", http.StatusBadRequest)
		return
	}

	var fileLinks []string

	for _, fileHeader := range r.MultipartForm.File["files"] {

		file, err := fileHeader.Open()
		if err != nil {
			utils.WriteError(w, "Can't get file", http.StatusBadRequest)
			return
		}

		mediaLink, err := h.mediaService.Save(r.Context(), fileHeader.Filename, fileHeader.Size, file)
		if err != nil {
			fmt.Println(err)
			utils.WriteError(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		fileLinks = append(fileLinks, mediaLink)
	}

	response := fileResponse{
		Files: fileLinks,
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
