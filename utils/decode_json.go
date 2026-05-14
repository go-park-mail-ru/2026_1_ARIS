package utils

import (
	"encoding/json"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_ARIS/common"
)

func DecodeJSON(w http.ResponseWriter, r *http.Request, out any) bool {
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(out); err != nil {
		WriteJSON(w, http.StatusBadRequest, common.ErrorResponse{Error: "invalid JSON body"})
		return false
	}
	return true
}
