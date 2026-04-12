package post

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/dto"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/media"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/post"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/user"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils"
)

type PostHandler struct {
	userService  user.UserService
	postService  post.PostService
	mediaService media.MediaService
}

func NewPostHandler(userService user.UserService, postService post.PostService, mediaService media.MediaService) *PostHandler {
	return &PostHandler{
		userService:  userService,
		postService:  postService,
		mediaService: mediaService,
	}
}

type PostCreationRequest struct {
	Media *[]dto.MediaRequestData `json:"media"`
	Text  *string                 `json:"text"`
	//Documents     *[]int64 `json:"documents"`
	//AllowComments bool `json:"allowComments"`
	//IsPublicDemo  bool `json:"isPublicDemo"`
	//NotifyFriends bool `json:"notifyFriends"`
}

type PostCreationResponse struct {
	MediaURL      []string `json:"mediaURL"`
	Text          *string  `json:"text"`
	FirstName     string   `json:"firstName"`
	LastName      string   `json:"lastName"`
	UserAccountID int64    `json:"userAccountID"`
	AvatarURL     *string  `json:"avatarURL"`
	//AllowComments bool      `json:"allowComments"`
}

func (h *PostHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
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

	//////////////////////////////////////////////////////////////////////////////////////////////

	var request PostCreationRequest

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		utils.WriteError(w, xerrors.InvalidRequestBody, http.StatusBadRequest)
		return
	}

	if request.Media == nil && request.Text == nil {
		utils.WriteError(w, xerrors.PostContentRequired, http.StatusBadRequest)
		return
	}

	post := models.NewPost(request.Text, profile.ID, true, true)

	postID, err := h.postService.Save(r.Context(), *post)
	if err != nil {
		utils.WriteError(w, "Can't save post", http.StatusInternalServerError)
		return
	}

	if request.Media != nil {
		// проверка на тип
		err := h.postService.AttachMedia(r.Context(), postID, *request.Media)
		if len(err.Errs) != 0 {
			utils.WriteError(w, "Can't attach media", http.StatusInternalServerError)
			json.NewEncoder(w).Encode(err)
			return
		}
	}

	//////////////////////////////////////////////////////////////////////////////////////////////

	var response PostCreationResponse

	if request.Text != nil {
		// postText := html.EscapeString(*request.Text)
		// response.Text = &postText
		response.Text = request.Text
	}

	if request.Media != nil {
		for _, media := range *request.Media {
			response.MediaURL = append(response.MediaURL, media.MediaURL)
		}
	}

	userProfile, err := h.userService.GetUserProfileByProfileID(r.Context(), profile.ID)
	if err != nil {
		utils.WriteError(w, "", 500)
		return
	}

	response.FirstName = userProfile.FirstName
	response.LastName = userProfile.LastName

	if profile.AvatarID != nil {
		avatar, err := h.mediaService.GetAvatarByID(r.Context(), profile.AvatarID)
		if err != nil {
			utils.WriteError(w, "", 500)
			return
		}
		response.AvatarURL = &avatar.Link
	}

	response.UserAccountID = userAccountID

	//////////////////////////////////////////////////////////////////////////////////////////////

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (h *PostHandler) DeletePost(w http.ResponseWriter, r *http.Request) {
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

	postIDstr := chi.URLParam(r, "id")
	postID, err := strconv.Atoi(postIDstr)
	if err != nil {
		utils.WriteError(w, xerrors.InvalidRequest, http.StatusBadRequest)
		return
	}

	post, err := h.postService.Get(r.Context(), int64(postID))
	if err != nil {
		if errors.Is(err, xerrors.PostNotFound) {
			utils.WriteError(w, err.Error(), http.StatusNotFound)
			return
		}
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	if post.AuthorID != profile.ID {
		utils.WriteError(w, "Denied", http.StatusForbidden)
		return
	}

	if err := h.postService.Delete(r.Context(), post.ID); err != nil {
		switch {
		case errors.Is(err, xerrors.PostNotFound):
			utils.WriteError(w, err.Error(), http.StatusNotFound)
			return
		case errors.Is(err, xerrors.MultipleRowsAffect):
			utils.WriteError(w, err.Error(), http.StatusInternalServerError)
			return
		default:
			utils.WriteError(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (h *PostHandler) GetPost(w http.ResponseWriter, r *http.Request) {
	postIDstr := chi.URLParam(r, "id")
	postID, err := strconv.Atoi(postIDstr)
	if err != nil {
		utils.WriteError(w, xerrors.InvalidRequest, http.StatusBadRequest)
		return
	}

	post, err := h.postService.Get(r.Context(), int64(postID))
	if err != nil {
		if errors.Is(err, xerrors.PostNotFound) {
			utils.WriteError(w, err.Error(), http.StatusNotFound)
			return
		}
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	authorID := post.AuthorID

	userProfile, err := h.userService.GetUserProfileByProfileID(r.Context(), authorID)
	if err != nil {
		if errors.Is(err, xerrors.UserProfileNotFound) {
			utils.WriteError(w, err.Error(), http.StatusNotFound)
			return
		}
		utils.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var response PostCreationResponse

	response.FirstName = userProfile.FirstName
	response.LastName = userProfile.LastName

	profile, err := h.userService.GetProfileByProfileID(r.Context(), userProfile.ProfileID)
	if err != nil {
		if errors.Is(err, xerrors.ProfileNotFound) {
			utils.WriteError(w, err.Error(), http.StatusNotFound)
			return
		}
		utils.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if profile.AvatarID != nil {
		avatar, err := h.mediaService.GetAvatarByID(r.Context(), profile.AvatarID)
		if err != nil {
			utils.WriteError(w, "", 500)
			return
		}
		response.AvatarURL = &avatar.Link
	}

	response.Text = post.Text
	response.UserAccountID = userProfile.UserAccountID

	postMedia := h.mediaService.GetMediasByPostID(r.Context(), int64(postID))

	postMediaURL := make([]string, 0, len(postMedia))

	for _, media := range postMedia {
		postMediaURL = append(postMediaURL, media.Link)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *PostHandler) UpdatePost(w http.ResponseWriter, r *http.Request) {
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

	//////////////////////////////////////////////////////

	postIDstr := chi.URLParam(r, "id")
	postID, err := strconv.Atoi(postIDstr)
	if err != nil {
		utils.WriteError(w, xerrors.InvalidRequest, http.StatusBadRequest)
		return
	}

	post, err := h.postService.Get(r.Context(), int64(postID))
	if err != nil {
		if errors.Is(err, xerrors.PostNotFound) {
			utils.WriteError(w, err.Error(), http.StatusNotFound)
			return
		}
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	if post.AuthorID != profile.ID {
		utils.WriteError(w, "Denied", http.StatusForbidden)
		return
	}

	var request PostCreationRequest

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		utils.WriteError(w, xerrors.InvalidRequestBody, http.StatusBadRequest)
		return
	}
}
