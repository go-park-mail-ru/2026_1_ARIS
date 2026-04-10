package media

import (
	"encoding/json"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/media"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/session"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils"
)

type MediaHandler struct {
	mediaService   media.MediaService
	sessionService session.SessionService
}

func NewMediaHandler(mediaService media.MediaService, sessionService session.SessionService) *MediaHandler {
	return &MediaHandler{
		mediaService:   mediaService,
		sessionService: sessionService,
	}
}

type mediaResponse struct {
	MediaID  int64  `json:"mediaID"`
	MediaURL string `json:"mediaURL"`
}

type fileResponse struct {
	Files []mediaResponse `json:"media"`
}

func (h *MediaHandler) SaveFiles(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		utils.WriteError(w, "Parsing form error", http.StatusBadRequest)
		return
	}

	var fileLinks []mediaResponse

	for _, fileHeader := range r.MultipartForm.File["files"] {

		file, err := fileHeader.Open()
		if err != nil {
			utils.WriteError(w, "Can't get file", http.StatusBadRequest)
			return
		}

		mediaID, mediaLink, err := h.mediaService.Save(r.Context(), fileHeader.Filename, fileHeader.Size, file)
		if err != nil {
			utils.WriteError(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		fileLinks = append(fileLinks, mediaResponse{MediaID: mediaID, MediaURL: mediaLink})
	}

	response := fileResponse{
		Files: fileLinks,
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
