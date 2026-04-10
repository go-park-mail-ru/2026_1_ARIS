package media

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/media"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/session"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/user"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils"
)

type MediaHandler struct {
	mediaService   media.MediaService
	sessionService session.SessionService
	userService    user.UserService
}

func NewMediaHandler(mediaService media.MediaService, sessionService session.SessionService, userService user.UserService) *MediaHandler {
	return &MediaHandler{
		mediaService:   mediaService,
		sessionService: sessionService,
		userService:    userService,
	}
}

type fileResponse struct {
	Files []string `json:"files"`
}

func (h *MediaHandler) SaveFiles(w http.ResponseWriter, r *http.Request) {
	userAccountID, ok := r.Context().Value("user_id").(int64)
	if !ok {
		utils.WriteError(w, xerrors.InvalidCtxUserAccountValue, http.StatusUnauthorized)
		return
	}

	profile, err := h.userService.GetProfileByUserAccountID(r.Context(), userAccountID)
	if err != nil {
		if errors.Is(err, xerrors.ProfileNotFound) {
			utils.WriteError(w, err.Error(), http.StatusNotFound)
			return
		}
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	if err := r.ParseMultipartForm(50 << 20); err != nil {
		utils.WriteError(w, "Parsing form error", http.StatusBadRequest)
		return
	}

	fileFor := r.URL.Query().Get("for")
	if fileFor == "" {
		utils.WriteError(w, xerrors.InvalidRequest, http.StatusBadRequest)
		return
	}

	var fileLinks []string

	for _, fileHeader := range r.MultipartForm.File["files"] {

		file, err := fileHeader.Open()
		if err != nil {
			utils.WriteError(w, "Can't get file", http.StatusBadRequest)
			return
		}

		mediaLink, err := h.mediaService.Save(r.Context(), fileHeader.Filename, fileHeader.Size, file, fileFor, profile.ID)
		if err != nil {
			if errors.Is(err, xerrors.UnsupportedContentType) {
				utils.WriteError(w, err.Error(), http.StatusUnsupportedMediaType)
				return
			}
			utils.WriteError(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		fileLinks = append(fileLinks, mediaLink)
	}

	response := fileResponse{
		Files: fileLinks,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
