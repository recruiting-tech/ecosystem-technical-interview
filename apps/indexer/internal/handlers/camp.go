// Package handlers wires Kafka messages to OpenSearch index writes.
package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/imgacademy/em-challenge/apps/indexer/internal/domain"
	"github.com/imgacademy/em-challenge/apps/indexer/internal/events"
)

// Indexer is the subset of *opensearch.Client that the handler uses. Defining
// it here lets tests substitute a fake without spinning up OpenSearch.
type Indexer interface {
	IndexCamp(ctx context.Context, doc domain.CampDoc) error
}

// campEventV1 mirrors the Avro shape of Camp.V1.
type campEventV1 struct {
	ID         string    `avro:"id"`
	Name       string    `avro:"name"`
	Sport      string    `avro:"sport"`
	Location   string    `avro:"location"`
	Capacity   int32     `avro:"capacity"`
	StartDate  time.Time `avro:"start_date"`
	EndDate    time.Time `avro:"end_date"`
	OccurredAt time.Time `avro:"occurred_at"`
}

// CampHandler indexes Camp.V1 events into the camps OpenSearch index.
type CampHandler struct {
	dec *events.AvroDecoder
	idx Indexer
	now func() time.Time
}

func NewCampHandler(dec *events.AvroDecoder, idx Indexer) *CampHandler {
	return &CampHandler{dec: dec, idx: idx, now: time.Now}
}

// Handle is the watermill message.NoPublishHandlerFunc signature.
func (h *CampHandler) Handle(msg *message.Message) error {
	var ev campEventV1
	ctx := msg.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	if err := h.dec.Decode(ctx, msg.Payload, &ev); err != nil {
		return fmt.Errorf("decode Camp.V1: %w", err)
	}
	doc := domain.CampDoc{
		ID:        ev.ID,
		Name:      ev.Name,
		Sport:     ev.Sport,
		Location:  ev.Location,
		Capacity:  ev.Capacity,
		StartDate: ev.StartDate,
		EndDate:   ev.EndDate,
		IndexedAt: h.now(),
	}
	if err := h.idx.IndexCamp(ctx, doc); err != nil {
		return fmt.Errorf("index camp %s: %w", ev.ID, err)
	}
	return nil
}
