package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/media/service"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"go.uber.org/zap"
)

type Handler struct {
	media *service.Service
}

func New(media *service.Service) *Handler {
	return &Handler{media: media}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/{id}", h.RedirectToFile)
	r.Get("/{id}/url", h.GetFileURL)
	r.Post("/upload", h.SaveFiles)
}

func (h *Handler) SaveFiles(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	userAccountID, ok := r.Context().Value("user_id").(int64)
	if !ok {
		utils.WriteError(w, xerrors.InvalidCtxUserAccountValue, http.StatusUnauthorized)
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

	savedFiles, filesErrors, err := h.media.SaveFiles(r.Context(), service.SaveFilesInput{
		FileHeaders:   r.MultipartForm.File["files"],
		FileFor:       fileFor,
		UserAccountID: userAccountID,
	})
	if err != nil {
		if errors.Is(err, service.ErrFilesAreRequired) {
			utils.WriteError(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, service.ErrProfileNotFound) {
			utils.WriteError(w, err.Error(), http.StatusNotFound)
			return
		}

		utils.WriteError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	files := mapSavedFiles(savedFiles)
	filesErrs := mapFileErrors(filesErrors)

	if len(files) == 0 && len(filesErrs) == 0 {
		utils.WriteError(w, "files are required", http.StatusBadRequest)
		return
	}

	if len(filesErrs) != 0 {
		writeJSON(w, http.StatusMultiStatus, fileResponse{Files: files, FilesErrors: filesErrs})
		return
	}

	if log != nil {
		log.Info("files_saved", zap.Int64("user_account_id", userAccountID), zap.Int("count", len(files)), zap.String("for", fileFor))
	}

	writeJSON(w, http.StatusOK, fileResponse{Files: files, FilesErrors: filesErrs})
}

func (h *Handler) GetFileURL(w http.ResponseWriter, r *http.Request) {
	mediaID, ok := parseMediaID(w, r)
	if !ok {
		return
	}

	mediaURL, err := h.media.GetFileURL(r.Context(), mediaID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, urlResponse{MediaID: mediaID, MediaURL: mediaURL})
}

func (h *Handler) RedirectToFile(w http.ResponseWriter, r *http.Request) {
	mediaID, ok := parseMediaID(w, r)
	if !ok {
		return
	}

	mediaURL, err := h.media.GetFileURL(r.Context(), mediaID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	http.Redirect(w, r, mediaURL, http.StatusTemporaryRedirect)
}

func parseMediaID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	mediaID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || mediaID <= 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: service.ErrInvalidInput.Error()})
		return 0, false
	}
	return mediaID, true
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
	case errors.Is(err, service.ErrMediaNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: err.Error()})
	default:
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: xerrors.InternalServerErrorStr})
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func mapSavedFiles(files []service.SavedFile) []mediaResponse {
	res := make([]mediaResponse, len(files))

	for i, file := range files {
		res[i] = mediaResponse{
			Index:    file.Index,
			MediaID:  file.ID,
			MediaURL: file.URL,
		}
	}

	return res
}

func mapFileErrors(fileErrors []service.FileError) []FileError {
	res := make([]FileError, len(fileErrors))

	for i, err := range fileErrors {
		res[i] = FileError{Index: err.Index, Error: err.Error}
	}

	return res
}
