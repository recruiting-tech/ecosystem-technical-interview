// Package opensearch wraps opensearch-go with the bits this service uses:
// index bootstrapping, document upsert, and a small search query helper.
package opensearch

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/opensearch-project/opensearch-go/v4"
	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"

	"github.com/imgacademy/ecosystem-technical-interview/apps/indexer/internal/domain"
)

const CampsIndex = "camps"

//go:embed mapping.json
var campMapping []byte

// Client is a thin wrapper over opensearchapi.Client.
type Client struct {
	api *opensearchapi.Client
}

// New returns a Client connected to the given URL.
func New(url string) (*Client, error) {
	api, err := opensearchapi.NewClient(opensearchapi.Config{
		Client: opensearch.Config{
			Addresses: []string{url},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("opensearch client: %w", err)
	}
	return &Client{api: api}, nil
}

// EnsureCampsIndex creates the camps index with the mapping if it doesn't
// already exist. Safe to call on every startup; tolerates the index existing.
func (c *Client) EnsureCampsIndex(ctx context.Context) error {
	_, err := c.api.Indices.Create(ctx, opensearchapi.IndicesCreateReq{
		Index: CampsIndex,
		Body:  bytes.NewReader(campMapping),
	})
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "resource_already_exists_exception") {
		return nil
	}
	return fmt.Errorf("indices.create: %w", err)
}

// IndexCamp upserts a camp document keyed on the deterministic doc ID.
func (c *Client) IndexCamp(ctx context.Context, doc domain.CampDoc) error {
	body, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	_, err = c.api.Index(ctx, opensearchapi.IndexReq{
		Index:      CampsIndex,
		DocumentID: domain.BuildCampDocID(doc.ID),
		Body:       bytes.NewReader(body),
		Params:     opensearchapi.IndexParams{Refresh: "wait_for"},
	})
	if err != nil {
		return fmt.Errorf("index camp: %w", err)
	}
	return nil
}

// SearchHit is one search result row.
type SearchHit struct {
	ID  string          `json:"id"`
	Doc json.RawMessage `json:"doc"`
}

// SearchQuery parameterizes SearchCamps. SkillLevel is plumbed through to
// OpenSearch even though the web's filter form control is the Ch3 stretch
// goal — keeping the backend ready for the front-end to opt in.
type SearchQuery struct {
	Q          string
	Sport      string
	SkillLevel string
	Limit      int
}

// SearchCamps runs a simple multi-match query, optionally filtered by sport
// and skill_level (both keyword fields).
func (c *Client) SearchCamps(ctx context.Context, p SearchQuery) ([]SearchHit, error) {
	var must []map[string]any
	if p.Q != "" {
		must = append(must, map[string]any{
			"multi_match": map[string]any{
				"query":  p.Q,
				"fields": []string{"name^2", "location"},
			},
		})
	} else {
		must = append(must, map[string]any{"match_all": map[string]any{}})
	}
	var filter []map[string]any
	if p.Sport != "" {
		filter = append(filter, map[string]any{"term": map[string]any{"sport": p.Sport}})
	}
	if p.SkillLevel != "" {
		filter = append(filter, map[string]any{"term": map[string]any{"skill_level": p.SkillLevel}})
	}
	limit := p.Limit
	if limit <= 0 {
		limit = 50
	}

	body := map[string]any{
		"size": limit,
		"query": map[string]any{
			"bool": map[string]any{"must": must, "filter": filter},
		},
		"sort": []map[string]any{{"start_date": "asc"}},
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		return nil, err
	}
	resp, err := c.api.Search(ctx, &opensearchapi.SearchReq{
		Indices: []string{CampsIndex},
		Body:    &buf,
	})
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	if resp.Inspect().Response != nil && resp.Inspect().Response.IsError() {
		b, _ := io.ReadAll(resp.Inspect().Response.Body)
		return nil, fmt.Errorf("search error: %s", strings.TrimSpace(string(b)))
	}
	hits := make([]SearchHit, 0, len(resp.Hits.Hits))
	for _, h := range resp.Hits.Hits {
		hits = append(hits, SearchHit{ID: h.ID, Doc: h.Source})
	}
	return hits, nil
}
