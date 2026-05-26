package http

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/media/internal/usecase"
	"github.com/go-park-mail-ru/2026_1_ARIS/utils"
	"go.uber.org/zap"
)

type Handler struct {
	media *usecase.Service
}

func New(media *usecase.Service) *Handler {
	return &Handler{media: media}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/{id}", h.RedirectToFile)
	r.Get("/{id}/url", h.GetFileURL)
	r.Post("/upload", h.SaveFiles)
}

const maxBodySize = 55 << 20 // 55 МБ с запасом на multipart-overhead

func (h *Handler) SaveFiles(w http.ResponseWriter, r *http.Request) {
	logg := logger.FromContext(r.Context())

	userAccountID, ok := r.Context().Value("user_id").(int64)
	if !ok {
		utils.WriteJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		utils.WriteJSON(w, http.StatusBadRequest, errorResponse{Error: "request too large"})
		return
	}

	fileFor := r.URL.Query().Get("for")
	if fileFor == "" {
		utils.WriteJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid request"})
		return
	}

	savedFiles, fileErrors, err := h.media.SaveFiles(r.Context(), usecase.SaveFilesInput{
		FileHeaders:   r.MultipartForm.File["files"],
		FileFor:       fileFor,
		UserAccountID: userAccountID,
	})
	if err != nil {
		writeUploadError(w, err)
		return
	}

	files := mapSavedFiles(savedFiles)
	filesErrs := mapFileErrors(fileErrors)
	if len(files) == 0 && len(filesErrs) == 0 {
		utils.WriteJSON(w, http.StatusBadRequest, errorResponse{Error: "files are required"})
		return
	}
	if len(filesErrs) != 0 {
		utils.WriteJSON(w, http.StatusMultiStatus, fileResponse{Files: files, FilesErrors: filesErrs})
		return
	}

	if logg != nil {
		logg.Info("files_saved", zap.Int64("user_account_id", userAccountID), zap.Int("count", len(files)), zap.String("for", fileFor))
	}
	utils.WriteJSON(w, http.StatusOK, fileResponse{Files: files, FilesErrors: filesErrs})
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
	utils.WriteJSON(w, http.StatusOK, urlResponse{MediaID: mediaID, MediaURL: mediaURL})
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
		utils.WriteJSON(w, http.StatusBadRequest, errorResponse{Error: usecase.ErrInvalidInput.Error()})
		return 0, false
	}
	return mediaID, true
}

func writeUploadError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, usecase.ErrFilesAreRequired), errors.Is(err, usecase.ErrInvalidInput):
		utils.WriteJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
	case errors.Is(err, usecase.ErrProfileNotFound):
		utils.WriteJSON(w, http.StatusNotFound, errorResponse{Error: err.Error()})
	default:
		utils.WriteJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
	}
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, usecase.ErrInvalidInput):
		utils.WriteJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
	case errors.Is(err, usecase.ErrMediaNotFound):
		utils.WriteJSON(w, http.StatusNotFound, errorResponse{Error: err.Error()})
	default:
		utils.WriteJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
	}
}

func mapSavedFiles(files []usecase.SavedFile) []mediaResponse {
	res := make([]mediaResponse, len(files))
	for i, file := range files {
		res[i] = mediaResponse{Index: file.Index, MediaID: file.ID, MediaURL: file.URL}
	}
	return res
}

func mapFileErrors(fileErrors []usecase.FileError) []fileError {
	res := make([]fileError, len(fileErrors))
	for i, err := range fileErrors {
		res[i] = fileError{Index: err.Index, Error: err.Error}
	}
	return res
}
