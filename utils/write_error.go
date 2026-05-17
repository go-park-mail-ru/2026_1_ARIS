package utils

import "net/http"

func WriteError(w http.ResponseWriter, message string, code int) {
	WriteJSON(w, code, map[string]string{"error": message})
}
