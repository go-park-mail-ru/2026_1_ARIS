package http

import (
	"encoding/json"
	"errors"
	"html"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/community/internal/usecase"
	"github.com/go-park-mail-ru/2026_1_ARIS/utils"
)

type Handler struct {
	community *usecase.Service
}

func New(community *usecase.Service) *Handler {
	return &Handler{community: community}
}

func (h *Handler) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		if authMiddleware != nil {
			r.Use(authMiddleware)
		}
		r.Get("/communities", h.List)
		r.Post("/communities/check-exists", h.CheckExists)
		r.Get("/communities/{id}", h.Get)
		r.Get("/communities/by-profile/{profileID}", h.GetByProfileID)
		r.Get("/communities/{id}/members", h.ListMembers)
		r.Post("/communities/{id}/join", h.Join)
		r.Post("/communities/{id}/leave", h.Leave)
		r.Delete("/communities/{id}/members/{profileID}", h.RemoveMember)
		r.Patch("/communities/{id}/members/{profileID}/role", h.ChangeMemberRole)
		r.Post("/communities", h.Create)
		r.Patch("/communities/{id}", h.Update)
		r.Delete("/communities/{id}", h.Delete)
	})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userAccountID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	var req createCommunityRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	details, err := h.community.Create(r.Context(), userAccountID, usecase.CreateInput{
		Title:        req.Title,
		Bio:          req.Bio,
		Type:         req.Type,
		Username:     req.Username,
		AvatarID:     req.AvatarID,
		CoverMediaID: req.CoverMediaID,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusCreated, mapDetails(details))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	limit := parseIntQuery(r, "limit", 20)
	offset := parseIntQuery(r, "offset", 0)
	details, err := h.community.List(r.Context(), limit, offset, optionalUserID(r))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	items := make([]communityDetailsResponse, 0, len(details))
	for i := range details {
		items = append(items, mapDetails(&details[i]))
	}
	utils.WriteJSON(w, http.StatusOK, communityListResponse{Items: items})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	communityID, ok := parseID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	details, err := h.community.GetDetails(r.Context(), communityID, optionalUserID(r))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, mapDetails(details))
}

func (h *Handler) GetByProfileID(w http.ResponseWriter, r *http.Request) {
	profileID, ok := parseID(w, chi.URLParam(r, "profileID"))
	if !ok {
		return
	}
	details, err := h.community.GetDetailsByProfileID(r.Context(), profileID, optionalUserID(r))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, mapDetails(details))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	userAccountID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	communityID, ok := parseID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req updateCommunityRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	details, err := h.community.Update(r.Context(), userAccountID, communityID, usecase.UpdateInput{
		Title:        req.Title,
		Bio:          req.Bio,
		Type:         req.Type,
		Username:     req.Username,
		AvatarID:     req.AvatarID,
		CoverMediaID: req.CoverMediaID,
		RemoveAvatar: req.RemoveAvatar,
		RemoveCover:  req.RemoveCover,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, mapDetails(details))
}

func (h *Handler) CheckExists(w http.ResponseWriter, r *http.Request) {
	var req communityExistenceRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	result, err := h.community.CheckExists(r.Context(), usecase.CheckExistsInput{
		Title:    req.Title,
		Username: req.Username,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, communityExistenceResponse{
		Exists:         result.Exists,
		TitleExists:    result.TitleExists,
		UsernameExists: result.UsernameExists,
	})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userAccountID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	communityID, ok := parseID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	if err := h.community.Delete(r.Context(), userAccountID, communityID); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListMembers(w http.ResponseWriter, r *http.Request) {
	communityID, ok := parseID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	members, err := h.community.ListMembers(r.Context(), communityID, optionalUserID(r), parseBoolQuery(r, "includeBlocked"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	items := make([]communityMemberResponse, 0, len(members))
	for _, member := range members {
		items = append(items, mapMember(member))
	}
	utils.WriteJSON(w, http.StatusOK, communityMembersResponse{Items: items})
}

func (h *Handler) Join(w http.ResponseWriter, r *http.Request) {
	userAccountID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	communityID, ok := parseID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	member, err := h.community.Join(r.Context(), userAccountID, communityID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, mapMember(*member))
}

func (h *Handler) Leave(w http.ResponseWriter, r *http.Request) {
	userAccountID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	communityID, ok := parseID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	if err := h.community.Leave(r.Context(), userAccountID, communityID); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	userAccountID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	communityID, ok := parseID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	memberProfileID, ok := parseID(w, chi.URLParam(r, "profileID"))
	if !ok {
		return
	}
	if err := h.community.RemoveMember(r.Context(), userAccountID, communityID, memberProfileID); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ChangeMemberRole(w http.ResponseWriter, r *http.Request) {
	userAccountID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	communityID, ok := parseID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	memberProfileID, ok := parseID(w, chi.URLParam(r, "profileID"))
	if !ok {
		return
	}
	var req updateMemberRoleRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	member, err := h.community.ChangeMemberRole(r.Context(), userAccountID, communityID, memberProfileID, req.Role)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, mapMember(*member))
}

func mapDetails(details *usecase.Details) communityDetailsResponse {
	community := details.Community
	return communityDetailsResponse{
		Community: communityResponse{
			ID:           community.ID,
			UID:          community.Uid.String(),
			ProfileID:    community.ProfileID,
			Username:     html.EscapeString(community.Username),
			Title:        html.EscapeString(community.Title),
			Bio:          escapePtr(community.Bio),
			Type:         community.Type,
			AvatarID:     details.AvatarID,
			AvatarURL:    details.AvatarURL,
			CoverMediaID: community.CoverMediaID,
			CoverURL:     details.CoverURL,
			IsActive:     community.IsActive,
			CreatedAt:    community.CreatedAt,
			UpdatedAt:    community.UpdatedAt,
		},
		Membership: membershipResponse{
			IsMember: details.Membership.IsMember,
			Role:     details.Membership.Role,
			Blocked:  details.Membership.Blocked,
		},
		Permissions: permissionsResponse{
			CanEditCommunity:   details.Permission.CanEditCommunity,
			CanDeleteCommunity: details.Permission.CanDeleteCommunity,
			CanPost:            details.Permission.CanPost,
			CanPostAsCommunity: details.Permission.CanPostAsCommunity,
			CanPostAsMember:    details.Permission.CanPostAsMember,
			CanManageMembers:   details.Permission.CanManageMembers,
			CanChangeRoles:     details.Permission.CanChangeRoles,
		},
	}
}

func mapMember(member usecase.MemberDetails) communityMemberResponse {
	return communityMemberResponse{
		ProfileID:     member.ProfileID,
		UserAccountID: member.UserAccountID,
		FirstName:     html.EscapeString(member.FirstName),
		LastName:      html.EscapeString(member.LastName),
		Username:      html.EscapeString(member.Username),
		AvatarID:      member.AvatarID,
		AvatarURL:     member.AvatarURL,
		Role:          member.Role,
		Blocked:       member.Blocked,
		IsSelf:        member.IsSelf,
		JoinedAt:      member.JoinedAt,
	}
}

func userIDFromContext(w http.ResponseWriter, r *http.Request) (int64, bool) {
	userAccountID, ok := r.Context().Value("user_id").(int64)
	if !ok || userAccountID <= 0 {
		writeError(w, "invalid user context", http.StatusUnauthorized)
		return 0, false
	}
	return userAccountID, true
}

func optionalUserID(r *http.Request) *int64 {
	userAccountID, ok := r.Context().Value("user_id").(int64)
	if !ok || userAccountID <= 0 {
		return nil
	}
	return &userAccountID
}

func parseID(w http.ResponseWriter, value string) (int64, bool) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, "invalid request", http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

func parseIntQuery(r *http.Request, name string, fallback int) int {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func parseBoolQuery(r *http.Request, name string) bool {
	value, err := strconv.ParseBool(r.URL.Query().Get(name))
	return err == nil && value
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, usecase.ErrInvalidInput), errors.Is(err, usecase.ErrNothingToUpdate):
		utils.WriteJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
	case errors.Is(err, usecase.ErrAlreadyExists):
		utils.WriteJSON(w, http.StatusConflict, errorResponse{Error: err.Error()})
	case errors.Is(err, usecase.ErrCommunityNotFound), errors.Is(err, usecase.ErrCommunityMemberNotFound), errors.Is(err, usecase.ErrProfileNotFound):
		utils.WriteJSON(w, http.StatusNotFound, errorResponse{Error: err.Error()})
	case errors.Is(err, usecase.ErrForbidden):
		utils.WriteJSON(w, http.StatusForbidden, errorResponse{Error: err.Error()})
	default:
		utils.WriteJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
	}
}

func writeError(w http.ResponseWriter, message string, status int) {
	utils.WriteJSON(w, status, errorResponse{Error: message})
}

func escapePtr(value *string) *string {
	if value == nil {
		return nil
	}
	escaped := html.EscapeString(*value)
	return &escaped
}
