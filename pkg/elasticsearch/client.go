package elasticsearch

import (
	"fmt"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/go-park-mail-ru/2026_1_ARIS/utils"
)

func New() (*elasticsearch.Client, error) {
	addr := utils.EnvString("ELASTICSEARCH_ADDR", "http://elasticsearch:9200")

	cfg := elasticsearch.Config{
		Addresses: []string{addr},
	}
	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("create elasticsearch client: %w", err)
	}
	return client, nil
}
