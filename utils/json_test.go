package utils

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestJSONHelpers(t *testing.T) {
	payload := map[string]string{"name": "neo"}
	data, err := MarshalJSON(payload)
	if err != nil || !strings.Contains(string(data), `"name":"neo"`) {
		t.Fatalf("MarshalJSON() = %q, %v", data, err)
	}

	var decoded map[string]string
	if err := UnmarshalJSON(data, &decoded); err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}
	if decoded["name"] != "neo" {
		t.Fatalf("unexpected decoded payload: %+v", decoded)
	}

	rec := httptest.NewRecorder()
	WriteJSON(rec, http.StatusCreated, payload)
	if rec.Code != http.StatusCreated || rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("unexpected response: code=%d headers=%v", rec.Code, rec.Header())
	}
}

func TestDecodeJSONAndWriteError(t *testing.T) {
	var out struct {
		Name string `json:"name"`
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"neo"}`))
	if !DecodeJSON(rec, req, &out) || out.Name != "neo" {
		t.Fatalf("DecodeJSON() = %+v", out)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{bad`))
	if DecodeJSON(rec, req, &out) || rec.Code != http.StatusBadRequest {
		t.Fatalf("expected bad JSON response, got code=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	WriteError(rec, "broken", http.StatusTeapot)
	if rec.Code != http.StatusTeapot || !strings.Contains(rec.Body.String(), "broken") {
		t.Fatalf("unexpected error response: %d %s", rec.Code, rec.Body.String())
	}
}

func TestDerefString(t *testing.T) {
	if DerefString(nil) != "" {
		t.Fatal("nil string should dereference to empty")
	}
	value := "x"
	if DerefString(&value) != "x" {
		t.Fatal("unexpected dereferenced value")
	}
}
