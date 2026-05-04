// Package domain holds index document shapes and the deterministic-ID logic
// that lets us re-process events safely (idempotent writes).
package domain

import "time"

// CampDoc is the OpenSearch document shape for a camp.
type CampDoc struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Sport     string    `json:"sport"`
	Location  string    `json:"location"`
	Capacity  int32     `json:"capacity"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	IndexedAt time.Time `json:"indexed_at"`
}

// BuildCampDocID is deterministic: a given camp ID always maps to the same
// OpenSearch document ID. This makes upserts safe under retries / replays /
// backfill — re-processing the same Kafka event is a no-op.
func BuildCampDocID(campID string) string {
	return "camp:" + campID
}
