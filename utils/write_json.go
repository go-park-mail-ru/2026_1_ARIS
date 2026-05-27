package utils

import (
	"encoding/json"
	"net/http"

	"github.com/mailru/easyjson"
)

func MarshalJSON(payload any) ([]byte, error) {
	if m, ok := payload.(easyjson.Marshaler); ok {
		return easyjson.Marshal(m)
	}
	return json.Marshal(payload)
}

func UnmarshalJSON(data []byte, out any) error {
	if u, ok := out.(easyjson.Unmarshaler); ok {
		return easyjson.Unmarshal(data, u)
	}
	return json.Unmarshal(data, out)
}

func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	data, err := MarshalJSON(payload)
	if err == nil {
		_, _ = w.Write(data)
		_, _ = w.Write([]byte("\n"))
		return
	}
	_ = json.NewEncoder(w).Encode(payload)
}
