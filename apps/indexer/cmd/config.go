package cmd

import "os"

type config struct {
	KafkaBrokers      []string
	SchemaRegistryURL string
	OpenSearchURL     string
	ConsumerGroup     string
	SearchAddr        string
}

func loadConfig() config {
	get := func(k, def string) string {
		if v := os.Getenv(k); v != "" {
			return v
		}
		return def
	}
	return config{
		KafkaBrokers:      []string{get("KAFKA_BROKERS", "localhost:19092")},
		SchemaRegistryURL: get("SCHEMA_REGISTRY_URL", "http://localhost:18081"),
		OpenSearchURL:     get("OPENSEARCH_URL", "http://localhost:19200"),
		ConsumerGroup:     get("CONSUMER_GROUP", "indexer"),
		SearchAddr:        get("SEARCH_ADDR", ":8003"),
	}
}
