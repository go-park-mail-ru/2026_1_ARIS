package elasticsearch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	esv8 "github.com/elastic/go-elasticsearch/v8"
)

func TestMappings(t *testing.T) {
	settings := ngramSettings()
	if settings["max_ngram_diff"] != 18 {
		t.Fatalf("unexpected ngram settings: %+v", settings)
	}

	for name, mapping := range map[string]map[string]any{
		"users":       userMapping(),
		"communities": communityMapping(),
		"posts":       postMapping(),
	} {
		if _, ok := mapping["mappings"]; !ok {
			t.Fatalf("%s mapping misses mappings key: %+v", name, mapping)
		}
	}
}

func TestEnsureIndices(t *testing.T) {
	created := make(map[string]bool)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
		index := strings.Trim(r.URL.Path, "/")
		switch r.Method {
		case http.MethodHead:
			if created[index] {
				w.WriteHeader(http.StatusOK)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		case http.MethodPut:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode index body: %v", err)
			}
			if _, ok := body["mappings"]; !ok {
				t.Fatalf("create %s misses mappings: %+v", index, body)
			}
			created[index] = true
			_, _ = w.Write([]byte(`{"acknowledged":true}`))
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	}))
	defer server.Close()

	client, err := esv8.NewClient(esv8.Config{Addresses: []string{server.URL}})
	if err != nil {
		t.Fatalf("create es client: %v", err)
	}
	if err := EnsureIndices(context.Background(), client); err != nil {
		t.Fatalf("EnsureIndices() error = %v", err)
	}
	for _, index := range []string{IndexUsers, IndexCommunities, IndexPosts} {
		if !created[index] {
			t.Fatalf("expected %s to be created", index)
		}
	}
	if err := EnsureIndices(context.Background(), client); err != nil {
		t.Fatalf("EnsureIndices() second run error = %v", err)
	}
}
