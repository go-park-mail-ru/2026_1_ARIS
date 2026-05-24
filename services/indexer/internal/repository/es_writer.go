package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	elasticsearch "github.com/elastic/go-elasticsearch/v8"
	esindex "github.com/go-park-mail-ru/2026_1_ARIS/pkg/elasticsearch"
)

type ESWriter struct {
	client *elasticsearch.Client
}

func NewESWriter(client *elasticsearch.Client) *ESWriter {
	return &ESWriter{client: client}
}

type BulkItem struct {
	Index    string
	DocID    string
	Delete   bool
	Document map[string]any
}

func (w *ESWriter) Bulk(ctx context.Context, items []BulkItem) error {
	if len(items) == 0 {
		return nil
	}

	var buf bytes.Buffer
	for _, item := range items {
		if item.Delete {
			meta := map[string]any{
				"delete": map[string]any{
					"_index": item.Index,
					"_id":    item.DocID,
				},
			}
			line, _ := json.Marshal(meta)
			buf.Write(line)
			buf.WriteByte('\n')
		} else {
			meta := map[string]any{
				"index": map[string]any{
					"_index": item.Index,
					"_id":    item.DocID,
				},
			}
			line, _ := json.Marshal(meta)
			buf.Write(line)
			buf.WriteByte('\n')
			doc, _ := json.Marshal(item.Document)
			buf.Write(doc)
			buf.WriteByte('\n')
		}
	}

	res, err := w.client.Bulk(bytes.NewReader(buf.Bytes()), w.client.Bulk.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("bulk request: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("bulk error: %s", res.String())
	}

	var bulkResp struct {
		Errors bool `json:"errors"`
		Items  []map[string]struct {
			Status int    `json:"status"`
			Error  *struct {
				Type   string `json:"type"`
				Reason string `json:"reason"`
			} `json:"error"`
		} `json:"items"`
	}
	if err := json.NewDecoder(res.Body).Decode(&bulkResp); err != nil {
		return nil
	}
	if bulkResp.Errors {
		for _, item := range bulkResp.Items {
			for op, result := range item {
				if result.Error != nil && !(op == "delete" && result.Status == 404) {
					return fmt.Errorf("bulk item error: %s: %s", result.Error.Type, result.Error.Reason)
				}
			}
		}
	}
	return nil
}

func UserDocToBulkItem(doc *UserDoc) BulkItem {
	return BulkItem{
		Index: esindex.IndexUsers,
		DocID: strconv.FormatInt(doc.UserAccountID, 10),
		Document: map[string]any{
			"user_account_id": doc.UserAccountID,
			"profile_id":      doc.ProfileID,
			"username":        doc.Username,
			"first_name":      doc.FirstName,
			"last_name":       doc.LastName,
			"full_name":       doc.FirstName + " " + doc.LastName,
			"avatar_id":       doc.AvatarID,
			"is_active":       doc.IsActive,
		},
	}
}

func CommunityDocToBulkItem(doc *CommunityDoc) BulkItem {
	return BulkItem{
		Index: esindex.IndexCommunities,
		DocID: strconv.FormatInt(doc.CommunityID, 10),
		Document: map[string]any{
			"community_id":   doc.CommunityID,
			"profile_id":     doc.ProfileID,
			"username":       doc.Username,
			"title":          doc.Title,
			"bio":            doc.Bio,
			"community_type": doc.CommunityType,
			"avatar_id":      doc.AvatarID,
			"cover_media_id": doc.CoverMediaID,
			"is_active":      doc.IsActive,
		},
	}
}

func PostDocToBulkItem(doc *PostDoc) BulkItem {
	return BulkItem{
		Index: esindex.IndexPosts,
		DocID: strconv.FormatInt(doc.PostID, 10),
		Document: map[string]any{
			"post_id":           doc.PostID,
			"post_text":         doc.PostText,
			"author_id":         doc.AuthorID,
			"author_profile_id": doc.AuthorProfileID,
			"author_username":   doc.AuthorUsername,
			"author_first_name": doc.AuthorFirstName,
			"author_last_name":  doc.AuthorLastName,
			"author_avatar_id":  doc.AuthorAvatarID,
			"community_id":      doc.CommunityID,
			"is_active":         doc.IsActive,
			"is_public":         doc.IsPublic,
			"created_at":        doc.CreatedAt.UTC().Format(time.RFC3339),
		},
	}
}

func DeleteBulkItem(index, docID string) BulkItem {
	return BulkItem{Index: index, DocID: docID, Delete: true}
}
