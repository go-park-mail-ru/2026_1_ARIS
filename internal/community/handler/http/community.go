package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/community/service"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils"
)

type Handler struct {
	community *service.Service
}

func New(community *service.Service) *Handler {
	return &Handler{community: community}
}

func (h *Handler) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		if authMiddleware != nil {
			r.Use(authMiddleware)
		}
		r.Get("/communities", h.List)
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
		utils.WriteError(w, xerrors.InvalidRequestBody, http.StatusBadRequest)
		return
	}
	fmt.Println("COMMUNITY CREATION REQUEST", req)
	details, err := h.community.Create(r.Context(), userAccountID, service.CreateInput{
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
	writeJSON(w, http.StatusCreated, mapDetails(details))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	limit := parseIntQuery(r, "limit", 20)
	offset := parseIntQuery(r, "offset", 0)
	viewer := optionalUserID(r)
	details, err := h.community.List(r.Context(), limit, offset, viewer)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	items := make([]communityDetailsResponse, 0, len(details))
	for i := range details {
		items = append(items, mapDetails(&details[i]))
	}
	writeJSON(w, http.StatusOK, communityListResponse{Items: items})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	communityID, ok := parseID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	details, err := h.community.Get(r.Context(), communityID, optionalUserID(r))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, mapDetails(details))
}

func (h *Handler) GetByProfileID(w http.ResponseWriter, r *http.Request) {
	profileID, ok := parseID(w, chi.URLParam(r, "profileID"))
	if !ok {
		return
	}
	details, err := h.community.GetByProfileID(r.Context(), profileID, optionalUserID(r))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, mapDetails(details))
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
		utils.WriteError(w, xerrors.InvalidRequestBody, http.StatusBadRequest)
		return
	}
	details, err := h.community.Update(r.Context(), userAccountID, communityID, service.UpdateInput{
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
	writeJSON(w, http.StatusOK, mapDetails(details))
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
	viewer := optionalUserID(r)
	includeBlocked := parseBoolQuery(r, "includeBlocked")
	members, err := h.community.ListMembers(r.Context(), communityID, viewer, includeBlocked)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	items := make([]communityMemberResponse, 0, len(members))
	for _, member := range members {
		items = append(items, mapMember(member))
	}
	writeJSON(w, http.StatusOK, communityMembersResponse{Items: items})
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
	writeJSON(w, http.StatusOK, mapMember(*member))
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
		utils.WriteError(w, xerrors.InvalidRequestBody, http.StatusBadRequest)
		return
	}
	member, err := h.community.ChangeMemberRole(r.Context(), userAccountID, communityID, memberProfileID, req.Role)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, mapMember(*member))
}

func mapDetails(details *service.Details) communityDetailsResponse {
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

func mapMember(member service.MemberDetails) communityMemberResponse {
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
		utils.WriteError(w, xerrors.InvalidCtxUserAccountValue, http.StatusUnauthorized)
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
		utils.WriteError(w, xerrors.InvalidRequest, http.StatusBadRequest)
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
	case errors.Is(err, service.ErrInvalidInput), errors.Is(err, service.ErrNothingToUpdate):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
	case errors.Is(err, service.ErrAlreadyExists):
		writeJSON(w, http.StatusConflict, errorResponse{Error: err.Error()})
	case errors.Is(err, service.ErrCommunityNotFound), errors.Is(err, service.ErrCommunityMemberNotFound), errors.Is(err, service.ErrProfileNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: err.Error()})
	case errors.Is(err, service.ErrForbidden):
		writeJSON(w, http.StatusForbidden, errorResponse{Error: err.Error()})
	default:
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: xerrors.InternalServerErrorStr})
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func escapePtr(value *string) *string {
	if value == nil {
		return nil
	}
	escaped := html.EscapeString(*value)
	return &escaped
}
