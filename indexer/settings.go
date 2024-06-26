package indexer

import (
	"fmt"
	"os"
)

var (
	InsertBatchSize              = 1000 // Number of rows to insert in a single batch parameters_amount_per_row * InsertBatchSize <= 65535
	SeerCrawlerLabel             string
	MOONSTREAM_DB_V3_INDEXES_URI string
)

func CheckVariablesForIndexer() error {
	SeerCrawlerLabel = os.Getenv("SEER_CRAWLER_INDEXER_LABEL")
	if SeerCrawlerLabel == "" {
		return fmt.Errorf("SEER_CRAWLER_INDEXER_LABEL environment variable is required")
	}

	MOONSTREAM_DB_V3_INDEXES_URI = os.Getenv("MOONSTREAM_DB_V3_INDEXES_URI")
	if MOONSTREAM_DB_V3_INDEXES_URI == "" {
		return fmt.Errorf("MOONSTREAM_DB_V3_INDEXES_URI environment variable is required")
	}

	return nil
}
