package authhandler

import (
	"encoding/json"
	"html"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils"
	authpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthHandler struct {
	authClient authpb.AuthServiceClient
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, "invalid request", http.StatusBadRequest)
		return
	}

	resp, err := h.authClient.Login(r.Context(), &authpb.LoginRequest{
		Login:    req.Login,
		Password: req.Password,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if !ok {
			utils.WriteError(w, "internal server error", http.StatusInternalServerError)
			return
		}

		switch st.Code() {
		case codes.Unauthenticated:
			utils.WriteError(w, "invalid credentials", http.StatusUnauthorized)
		case codes.NotFound:
			utils.WriteError(w, "user not found", http.StatusNotFound)
		default:
			utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    resp.GetSessionId(),
		Expires:  resp.GetExpiresAt().AsTime(),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   true,
		Path:     "/",
	})

	loginResponse := LoginResponse{
		ProfileID:  resp.GetProfileId(),
		FirstName:  html.EscapeString(resp.GetFirstName()),
		LastName:   html.EscapeString(resp.GetLastName()),
		AvatarLink: resp.GetAvatarUrl(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(loginResponse)
}
