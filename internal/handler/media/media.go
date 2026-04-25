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
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"go.uber.org/zap"
)

type MediaHandler struct {
	mediaService   media.MediaService
	sessionService session.SessionService // нигде не используется
	userService    user.UserService
}

func NewMediaHandler(mediaService media.MediaService, sessionService session.SessionService, userService user.UserService) *MediaHandler {
	return &MediaHandler{
		mediaService:   mediaService,
		sessionService: sessionService,
		userService:    userService,
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
	log := logger.FromContext(r.Context())

	userAccountID, ok := r.Context().Value("user_id").(int64)
	if !ok {
		log.Warn("cannot_save_files_missing_user",
			zap.String("path", r.URL.Path),
		)
		utils.WriteError(w, xerrors.InvalidCtxUserAccountValue, http.StatusUnauthorized)
		return
	}

	profile, err := h.userService.GetProfileByUserAccountID(r.Context(), userAccountID)
	if err != nil {
		if errors.Is(err, xerrors.ProfileNotFound) {
			log.Warn("cannot_save_files_profile_not_found",
				zap.Int64("userAccount_id", userAccountID),
			)
			utils.WriteError(w, err.Error(), http.StatusNotFound)
			return
		}
		log.Error("failed_to_get_profile",
			zap.Int64("userAccount_id", userAccountID),
			zap.Error(err),
		)
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	if err := r.ParseMultipartForm(50 << 20); err != nil {
		log.Warn("cannot_save_files_parse_form",
			zap.Error(err),
		)
		utils.WriteError(w, "Parsing form error", http.StatusBadRequest)
		return
	}

	fileFor := r.URL.Query().Get("for")
	if fileFor == "" {
		log.Warn("cannot_save_files_invalid_for_query",
			zap.String("path", r.URL.Path),
		)
		utils.WriteError(w, xerrors.InvalidRequest, http.StatusBadRequest)
		return
	}

	var fileLinks []mediaResponse

	for _, fileHeader := range r.MultipartForm.File["files"] {

		file, err := fileHeader.Open()
		if err != nil {

			utils.WriteError(w, "Can't get file", http.StatusBadRequest)
			return
		}

		mediaID, mediaLink, err := h.mediaService.Save(r.Context(), fileHeader.Filename, fileHeader.Size, file, fileFor, profile.ID)

		if err != nil {
			if errors.Is(err, xerrors.UnsupportedContentType) {
				log.Warn("cannot_save_files_unsupported_content_type",
					zap.String("filename", fileHeader.Filename),
					zap.Error(err),
				)
				utils.WriteError(w, err.Error(), http.StatusUnsupportedMediaType)
				return
			}

			utils.WriteError(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		fileLinks = append(fileLinks, mediaResponse{MediaID: mediaID, MediaURL: mediaLink})
	}

	response := fileResponse{
		Files: fileLinks,
	}

	log.Info("files_saved",
		zap.Int64("profile_id", profile.ID),
		zap.Int("count", len(fileLinks)),
		zap.String("for", fileFor),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
