package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/elastic/go-elasticsearch/v8"
)

const (
	IndexUsers       = "aris_users"
	IndexCommunities = "aris_communities"
	IndexPosts       = "aris_posts"
)

func EnsureIndices(ctx context.Context, client *elasticsearch.Client) error {
	if err := ensureIndex(ctx, client, IndexUsers, userMapping()); err != nil {
		return err
	}
	if err := ensureIndex(ctx, client, IndexCommunities, communityMapping()); err != nil {
		return err
	}
	return ensureIndex(ctx, client, IndexPosts, postMapping())
}

func ensureIndex(ctx context.Context, client *elasticsearch.Client, index string, mapping map[string]any) error {
	res, err := client.Indices.Exists([]string{index}, client.Indices.Exists.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("check index %s: %w", index, err)
	}
	defer res.Body.Close()
	if res.StatusCode == 200 {
		return nil
	}

	body, _ := json.Marshal(mapping)
	createRes, err := client.Indices.Create(
		index,
		client.Indices.Create.WithContext(ctx),
		client.Indices.Create.WithBody(bytes.NewReader(body)),
	)
	if err != nil {
		return fmt.Errorf("create index %s: %w", index, err)
	}
	defer createRes.Body.Close()
	if createRes.IsError() {
		return fmt.Errorf("create index %s: %s", index, createRes.String())
	}
	return nil
}

func ngramSettings() map[string]any {
	return map[string]any{
		"analysis": map[string]any{
			"tokenizer": map[string]any{
				"ngram_tokenizer": map[string]any{
					"type":        "ngram",
					"min_gram":    2,
					"max_gram":    20,
					"token_chars": []string{"letter", "digit"},
				},
			},
			"analyzer": map[string]any{
				"ngram_analyzer": map[string]any{
					"type":      "custom",
					"tokenizer": "ngram_tokenizer",
					"filter":    []string{"lowercase"},
				},
			},
			"normalizer": map[string]any{
				"lowercase": map[string]any{
					"type":   "custom",
					"filter": []string{"lowercase"},
				},
			},
		},
		"max_ngram_diff": 18,
	}
}

func userMapping() map[string]any {
	return map[string]any{
		"settings": ngramSettings(),
		"mappings": map[string]any{
			"properties": map[string]any{
				"user_account_id": map[string]any{"type": "long"},
				"profile_id":      map[string]any{"type": "long"},
				"username": map[string]any{
					"type":            "text",
					"analyzer":        "ngram_analyzer",
					"search_analyzer": "standard",
					"fields": map[string]any{
						"keyword": map[string]any{
							"type":       "keyword",
							"normalizer": "lowercase",
						},
					},
				},
				"first_name": map[string]any{"type": "text", "analyzer": "ngram_analyzer", "search_analyzer": "standard"},
				"last_name":  map[string]any{"type": "text", "analyzer": "ngram_analyzer", "search_analyzer": "standard"},
				"full_name":  map[string]any{"type": "text", "analyzer": "ngram_analyzer", "search_analyzer": "standard"},
				"avatar_id":  map[string]any{"type": "long"},
				"is_active":  map[string]any{"type": "boolean"},
			},
		},
	}
}

func communityMapping() map[string]any {
	return map[string]any{
		"settings": ngramSettings(),
		"mappings": map[string]any{
			"properties": map[string]any{
				"community_id": map[string]any{"type": "long"},
				"profile_id":   map[string]any{"type": "long"},
				"username": map[string]any{
					"type":            "text",
					"analyzer":        "ngram_analyzer",
					"search_analyzer": "standard",
					"fields": map[string]any{
						"keyword": map[string]any{
							"type":       "keyword",
							"normalizer": "lowercase",
						},
					},
				},
				"title":          map[string]any{"type": "text", "analyzer": "ngram_analyzer", "search_analyzer": "standard"},
				"bio":            map[string]any{"type": "text", "analyzer": "ngram_analyzer", "search_analyzer": "standard"},
				"community_type": map[string]any{"type": "keyword"},
				"avatar_id":      map[string]any{"type": "long"},
				"cover_media_id": map[string]any{"type": "long"},
				"is_active":      map[string]any{"type": "boolean"},
			},
		},
	}
}

func postMapping() map[string]any {
	return map[string]any{
		"mappings": map[string]any{
			"properties": map[string]any{
				"post_id":           map[string]any{"type": "long"},
				"post_text":         map[string]any{"type": "text", "analyzer": "standard"},
				"author_id":         map[string]any{"type": "long"},
				"author_profile_id": map[string]any{"type": "long"},
				"author_username":   map[string]any{"type": "keyword"},
				"author_first_name": map[string]any{"type": "keyword"},
				"author_last_name":  map[string]any{"type": "keyword"},
				"author_avatar_id":  map[string]any{"type": "long"},
				"community_id":      map[string]any{"type": "long"},
				"is_active":         map[string]any{"type": "boolean"},
				"is_public":         map[string]any{"type": "boolean"},
				"created_at":        map[string]any{"type": "date"},
			},
		},
	}
}
