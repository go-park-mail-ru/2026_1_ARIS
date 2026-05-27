package utils

import (
	"io"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_ARIS/common"
)

func DecodeJSON(w http.ResponseWriter, r *http.Request, out any) bool {
	defer r.Body.Close()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, common.ErrorResponse{Error: "invalid JSON body"})
		return false
	}
	if err := UnmarshalJSON(data, out); err != nil {
		WriteJSON(w, http.StatusBadRequest, common.ErrorResponse{Error: "invalid JSON body"})
		return false
	}
	return true
}
