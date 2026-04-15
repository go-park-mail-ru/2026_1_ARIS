package post

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/dto"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/media"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/post"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/user"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"go.uber.org/zap"
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
	ID            int64                  `json:"id"`
	ProfileID     int64                  `json:"profileID"`
	Media         []dto.MediaRequestData `json:"media"`
	MediaURL      []string               `json:"mediaURL"`
	Text          *string                `json:"text"`
	FirstName     string                 `json:"firstName"`
	LastName      string                 `json:"lastName"`
	UserAccountID int64                  `json:"userAccountID"`
	AvatarURL     *string                `json:"avatarURL"`
	//AllowComments bool      `json:"allowComments"`
}

type PostListItemResponse struct {
	ID        int64                  `json:"id"`
	ProfileID int64                  `json:"profileID"`
	Text      string                 `json:"text"`
	Media     []dto.MediaRequestData `json:"media"`
	MediaURL  []string               `json:"mediaURL"`
	CreatedAt time.Time              `json:"createdAt"`
	UpdatedAt *time.Time             `json:"updatedAt,omitempty"`
}

func (h *PostHandler) buildPostListItemResponse(ctx context.Context, postModel models.Post) PostListItemResponse {
	postMedia := h.mediaService.GetMediasByPostID(ctx, postModel.ID)

	mediaURL := make([]string, 0, len(postMedia))
	mediaItems := make([]dto.MediaRequestData, 0, len(postMedia))
	for _, media := range postMedia {
		mediaURL = append(mediaURL, media.Link)
		mediaItems = append(mediaItems, dto.MediaRequestData{
			MediaID:  media.ID,
			MediaURL: media.Link,
		})
	}

	response := PostListItemResponse{
		ID:        postModel.ID,
		ProfileID: postModel.AuthorID,
		Text:      "",
		Media:     mediaItems,
		MediaURL:  mediaURL,
		CreatedAt: postModel.CreatedAt,
	}

	if postModel.Text != nil {
		response.Text = *postModel.Text
	}

	if !postModel.UpdatedAt.IsZero() {
		updatedAt := postModel.UpdatedAt
		response.UpdatedAt = &updatedAt
	}

	return response
}

func (h *PostHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	userAccountID, ok := r.Context().Value("user_id").(int64)
	if !ok {
		log.Warn("cannot_create_post_missing_user",
			zap.String("path", r.URL.Path),
		)
		utils.WriteError(w, xerrors.InvalidCtxUserAccountValue, http.StatusUnauthorized)
		return
	}

	profile, err := h.userService.GetProfileByUserAccountID(r.Context(), userAccountID)
	if err != nil {
		if errors.Is(err, xerrors.ProfileNotFound) {
			log.Warn("cannot_create_post_profile_not_found",
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

	//////////////////////////////////////////////////////////////////////////////////////////////

	var request PostCreationRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		log.Warn("cannot_create_post_invalid_body",
			zap.String("path", r.URL.Path),
			zap.Error(err),
		)
		utils.WriteError(w, xerrors.InvalidRequestBody, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if request.Media == nil && request.Text == nil {
		log.Warn("cannot_create_post_empty_content",
			zap.String("path", r.URL.Path),
		)
		utils.WriteError(w, xerrors.PostContentRequired, http.StatusBadRequest)
		return
	}

	post := models.NewPost(request.Text, profile.ID, false, true)

	postID, err := h.postService.Save(r.Context(), *post)
	if err != nil {
		log.Error("failed_to_save_post",
			zap.Int64("profile_id", profile.ID),
			zap.Error(err),
		)
		utils.WriteError(w, "Can't save post", http.StatusInternalServerError)
		return
	}

	if request.Media != nil {
		// проверка на тип
		err := h.postService.AttachMedia(r.Context(), postID, *request.Media)
		if len(err.Errs) != 0 {
			log.Error("failed_to_attach_media",
				zap.Int64("post_id", postID),
				zap.Int("errors", len(err.Errs)),
			)
			utils.WriteError(w, "Can't attach media", http.StatusInternalServerError)
			json.NewEncoder(w).Encode(err)
			return
		}
	}

	//////////////////////////////////////////////////////////////////////////////////////////////

	var response PostCreationResponse
	response.ID = postID
	response.ProfileID = profile.ID

	if request.Text != nil {
		// postText := html.EscapeString(*request.Text)
		// response.Text = &postText
		response.Text = request.Text
	}

	if request.Media != nil {
		for _, media := range *request.Media {
			response.Media = append(response.Media, media)
			response.MediaURL = append(response.MediaURL, media.MediaURL)
		}
	}

	userProfile, err := h.userService.GetUserProfileByProfileID(r.Context(), profile.ID)
	if err != nil {
		log.Error("failed_to_get_user_profile",
			zap.Int64("profile_id", profile.ID),
			zap.Error(err),
		)
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	response.FirstName = userProfile.FirstName
	response.LastName = userProfile.LastName

	if profile.AvatarID != nil {
		avatar, err := h.mediaService.GetAvatarByID(r.Context(), profile.AvatarID)
		if err != nil {
			log.Error("failed_to_get_avatar",
				zap.Int64("profile_id", profile.ID),
				zap.Error(err),
			)
			utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
			return
		}
		response.AvatarURL = &avatar.Link
	}

	response.UserAccountID = userAccountID

	//////////////////////////////////////////////////////////////////////////////////////////////

	log.Info("post_created",
		zap.Int64("post_id", postID),
		zap.Int64("profile_id", profile.ID),
		zap.Int("media_count", func() int {
			if request.Media != nil {
				return len(*request.Media)
			}
			return 0
		}()),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (h *PostHandler) GetMyPosts(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	userAccountID, ok := r.Context().Value("user_id").(int64)
	if !ok {
		log.Warn("cannot_get_my_posts_missing_user",
			zap.String("path", r.URL.Path),
		)
		utils.WriteError(w, xerrors.InvalidCtxUserAccountValue, http.StatusUnauthorized)
		return
	}

	profile, err := h.userService.GetProfileByUserAccountID(r.Context(), userAccountID)
	if err != nil {
		if errors.Is(err, xerrors.ProfileNotFound) {
			log.Warn("cannot_get_my_posts_profile_not_found",
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

	posts, err := h.postService.GetByAuthorID(r.Context(), profile.ID)
	if err != nil {
		log.Error("failed_to_get_my_posts",
			zap.Int64("profile_id", profile.ID),
			zap.Error(err),
		)
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	response := make([]PostListItemResponse, 0, len(posts))
	for _, postModel := range posts {
		response = append(response, h.buildPostListItemResponse(r.Context(), postModel))
	}

	log.Info("my_posts_returned",
		zap.Int64("profile_id", profile.ID),
		zap.Int("count", len(response)),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *PostHandler) GetProfilePosts(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	profileIDstr := chi.URLParam(r, "profileID")
	profileID, err := strconv.ParseInt(profileIDstr, 10, 64)
	if err != nil {
		log.Warn("cannot_get_profile_posts_invalid_id",
			zap.String("profile_id", profileIDstr),
			zap.String("path", r.URL.Path),
			zap.Error(err),
		)
		utils.WriteError(w, xerrors.InvalidRequest, http.StatusBadRequest)
		return
	}

	posts, err := h.postService.GetByAuthorID(r.Context(), profileID)
	if err != nil {
		log.Error("failed_to_get_profile_posts",
			zap.Int64("profile_id", profileID),
			zap.Error(err),
		)
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	response := make([]PostListItemResponse, 0, len(posts))
	for _, postModel := range posts {
		response = append(response, h.buildPostListItemResponse(r.Context(), postModel))
	}

	log.Info("profile_posts_returned",
		zap.Int64("profile_id", profileID),
		zap.Int("count", len(response)),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *PostHandler) DeletePost(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	userAccountID, ok := r.Context().Value("user_id").(int64)
	if !ok {
		log.Warn("cannot_delete_post_missing_user",
			zap.String("path", r.URL.Path),
		)
		utils.WriteError(w, xerrors.InvalidCtxUserAccountValue, http.StatusUnauthorized)
		return
	}

	profile, err := h.userService.GetProfileByUserAccountID(r.Context(), userAccountID)
	if err != nil {
		if errors.Is(err, xerrors.ProfileNotFound) {
			log.Warn("cannot_delete_post_profile_not_found",
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

	postIDstr := chi.URLParam(r, "id")
	postID, err := strconv.Atoi(postIDstr)
	if err != nil {
		log.Warn("cannot_delete_post_invalid_id",
			zap.String("post_id", postIDstr),
			zap.String("path", r.URL.Path),
			zap.Error(err),
		)
		utils.WriteError(w, xerrors.InvalidRequest, http.StatusBadRequest)
		return
	}

	post, err := h.postService.Get(r.Context(), int64(postID))
	if err != nil {
		if errors.Is(err, xerrors.PostNotFound) {
			log.Warn("cannot_delete_post_not_found",
				zap.Int64("post_id", int64(postID)),
			)
			utils.WriteError(w, err.Error(), http.StatusNotFound)
			return
		}
		log.Error("failed_to_get_post",
			zap.Int64("post_id", int64(postID)),
			zap.Error(err),
		)
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	if post.AuthorID != profile.ID {
		log.Warn("cannot_delete_post_denied",
			zap.Int64("post_id", post.ID),
			zap.Int64("profile_id", profile.ID),
		)
		utils.WriteError(w, "Denied", http.StatusForbidden)
		return
	}

	if err := h.postService.Delete(r.Context(), post.ID); err != nil {
		switch {
		case errors.Is(err, xerrors.PostNotFound):
			log.Warn("cannot_delete_post_not_found",
				zap.Int64("post_id", post.ID),
				zap.Error(err),
			)
			utils.WriteError(w, err.Error(), http.StatusNotFound)
			return
		case errors.Is(err, xerrors.MultipleRowsAffect):
			log.Error("failed_to_delete_post",
				zap.Int64("post_id", post.ID),
				zap.Error(err),
			)
			utils.WriteError(w, err.Error(), http.StatusInternalServerError)
			return
		default:
			log.Error("failed_to_delete_post",
				zap.Int64("post_id", post.ID),
				zap.Error(err),
			)
			utils.WriteError(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	log.Info("post_deleted",
		zap.Int64("post_id", post.ID),
		zap.Int64("profile_id", profile.ID),
	)

	w.WriteHeader(http.StatusOK)
}

func (h *PostHandler) GetPost(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	postIDstr := chi.URLParam(r, "id")
	postID, err := strconv.Atoi(postIDstr)
	if err != nil {
		log.Warn("cannot_get_post_invalid_id",
			zap.String("post_id", postIDstr),
			zap.String("path", r.URL.Path),
			zap.Error(err),
		)
		utils.WriteError(w, xerrors.InvalidRequest, http.StatusBadRequest)
		return
	}

	post, err := h.postService.Get(r.Context(), int64(postID))
	if err != nil {
		if errors.Is(err, xerrors.PostNotFound) {
			log.Warn("cannot_get_post_not_found",
				zap.Int64("post_id", int64(postID)),
			)
			utils.WriteError(w, err.Error(), http.StatusNotFound)
			return
		}
		log.Error("failed_to_get_post",
			zap.Int64("post_id", int64(postID)),
			zap.Error(err),
		)
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	authorID := post.AuthorID

	userProfile, err := h.userService.GetUserProfileByProfileID(r.Context(), authorID)
	if err != nil {
		if errors.Is(err, xerrors.UserProfileNotFound) {
			log.Warn("cannot_get_post_user_profile_not_found",
				zap.Int64("author_id", authorID),
			)
			utils.WriteError(w, err.Error(), http.StatusNotFound)
			return
		}
		log.Error("failed_to_get_user_profile",
			zap.Int64("author_id", authorID),
			zap.Error(err),
		)
		utils.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var response PostCreationResponse
	response.ID = post.ID
	response.ProfileID = post.AuthorID

	response.FirstName = userProfile.FirstName
	response.LastName = userProfile.LastName

	profile, err := h.userService.GetProfileByProfileID(r.Context(), userProfile.ProfileID)
	if err != nil {
		if errors.Is(err, xerrors.ProfileNotFound) {
			log.Warn("cannot_get_post_profile_not_found",
				zap.Int64("profile_id", userProfile.ProfileID),
			)
			utils.WriteError(w, err.Error(), http.StatusNotFound)
			return
		}
		log.Error("failed_to_get_profile",
			zap.Int64("profile_id", userProfile.ProfileID),
			zap.Error(err),
		)
		utils.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if profile.AvatarID != nil {
		avatar, err := h.mediaService.GetAvatarByID(r.Context(), profile.AvatarID)
		if err != nil {
			log.Error("failed_to_get_avatar",
				zap.Int64("profile_id", profile.ID),
				zap.Error(err),
			)
			utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
			return
		}
		response.AvatarURL = &avatar.Link
	}

	response.Text = post.Text
	response.UserAccountID = userProfile.UserAccountID

	postMedia := h.mediaService.GetMediasByPostID(r.Context(), int64(postID))

	postMediaURL := make([]string, 0, len(postMedia))

	for _, media := range postMedia {
		response.Media = append(response.Media, dto.MediaRequestData{
			MediaID:  media.ID,
			MediaURL: media.Link,
		})
		postMediaURL = append(postMediaURL, media.Link)
	}
	response.MediaURL = postMediaURL

	log.Info("post_returned",
		zap.Int64("post_id", post.ID),
		zap.Int64("author_id", authorID),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *PostHandler) UpdatePost(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	userAccountID, ok := r.Context().Value("user_id").(int64)
	if !ok {
		log.Warn("cannot_update_post_missing_user",
			zap.String("path", r.URL.Path),
		)
		utils.WriteError(w, xerrors.InvalidCtxUserAccountValue, http.StatusUnauthorized)
		return
	}

	profile, err := h.userService.GetProfileByUserAccountID(r.Context(), userAccountID)
	if err != nil {
		if errors.Is(err, xerrors.ProfileNotFound) {
			log.Warn("cannot_update_post_profile_not_found",
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

	//////////////////////////////////////////////////////

	postIDstr := chi.URLParam(r, "id")
	postID, err := strconv.Atoi(postIDstr)
	if err != nil {
		log.Warn("cannot_update_post_invalid_id",
			zap.String("post_id", postIDstr),
			zap.String("path", r.URL.Path),
			zap.Error(err),
		)
		utils.WriteError(w, xerrors.InvalidRequest, http.StatusBadRequest)
		return
	}

	post, err := h.postService.Get(r.Context(), int64(postID))
	if err != nil {
		if errors.Is(err, xerrors.PostNotFound) {
			log.Warn("cannot_update_post_not_found",
				zap.Int64("post_id", int64(postID)),
			)
			utils.WriteError(w, err.Error(), http.StatusNotFound)
			return
		}
		log.Error("failed_to_get_post",
			zap.Int64("post_id", int64(postID)),
			zap.Error(err),
		)
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	if post.AuthorID != profile.ID {
		log.Warn("cannot_update_post_denied",
			zap.Int64("post_id", post.ID),
			zap.Int64("profile_id", profile.ID),
		)
		utils.WriteError(w, "Denied", http.StatusForbidden)
		return
	}

	var request PostCreationRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		log.Warn("cannot_update_post_invalid_body",
			zap.String("path", r.URL.Path),
			zap.Error(err),
		)
		utils.WriteError(w, xerrors.InvalidRequestBody, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	if request.Text == nil {
		log.Warn("cannot_update_post_empty_text",
			zap.String("path", r.URL.Path),
		)
		utils.WriteError(w, xerrors.PostContentRequired, http.StatusBadRequest)
		return
	}

	post.Text = request.Text

	if err := h.postService.Update(r.Context(), *post); err != nil {
		log.Error("failed_to_update_post",
			zap.Int64("post_id", post.ID),
			zap.Error(err),
		)
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	if request.Media != nil {
		err := h.postService.ReplaceMedia(r.Context(), post.ID, *request.Media)
		if len(err.Errs) != 0 {
			log.Error("failed_to_replace_media",
				zap.Int64("post_id", post.ID),
				zap.Int("errors", len(err.Errs)),
			)
			utils.WriteError(w, "Can't replace media", http.StatusInternalServerError)
			json.NewEncoder(w).Encode(err)
			return
		}
	}

	updatedPost, err := h.postService.Get(r.Context(), int64(postID))
	if err != nil {
		log.Error("failed_to_get_updated_post",
			zap.Int64("post_id", int64(postID)),
			zap.Error(err),
		)
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	postMedia := h.mediaService.GetMediasByPostID(r.Context(), updatedPost.ID)
	postMediaURL := make([]string, 0, len(postMedia))
	postMediaItems := make([]dto.MediaRequestData, 0, len(postMedia))
	for _, media := range postMedia {
		postMediaURL = append(postMediaURL, media.Link)
		postMediaItems = append(postMediaItems, dto.MediaRequestData{
			MediaID:  media.ID,
			MediaURL: media.Link,
		})
	}

	log.Info("post_updated",
		zap.Int64("post_id", updatedPost.ID),
		zap.Int64("profile_id", profile.ID),
		zap.Int("media_count", len(postMediaItems)),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(PostCreationResponse{
		ID:            updatedPost.ID,
		ProfileID:     updatedPost.AuthorID,
		Text:          updatedPost.Text,
		Media:         postMediaItems,
		MediaURL:      postMediaURL,
		UserAccountID: userAccountID,
	})
}
