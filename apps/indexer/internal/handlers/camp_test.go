package handlers_test

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/hamba/avro/v2"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/sr"

	"github.com/imgacademy/em-challenge/apps/indexer/internal/domain"
	"github.com/imgacademy/em-challenge/apps/indexer/internal/events"
	"github.com/imgacademy/em-challenge/apps/indexer/internal/handlers"
)

// fakeIndexer captures index calls without hitting OpenSearch.
type fakeIndexer struct {
	indexed []domain.CampDoc
}

func (f *fakeIndexer) IndexCamp(_ context.Context, doc domain.CampDoc) error {
	f.indexed = append(f.indexed, doc)
	return nil
}

func TestCampHandler_DecodesAndIndexes(t *testing.T) {
	const schemaID uint32 = 42
	schemaJSON := `{
		"type":"record","name":"Camp","namespace":"com.imgacademy.api.events",
		"fields":[
			{"name":"id","type":"string"},
			{"name":"name","type":"string"},
			{"name":"sport","type":"string"},
			{"name":"location","type":"string"},
			{"name":"capacity","type":"int"},
			{"name":"start_date","type":{"type":"int","logicalType":"date"}},
			{"name":"end_date","type":{"type":"int","logicalType":"date"}},
			{"name":"occurred_at","type":{"type":"long","logicalType":"timestamp-millis"}}
		]
	}`

	// Stand up an in-memory schema registry that resolves /schemas/ids/42.
	srMux := http.NewServeMux()
	srMux.HandleFunc("GET /schemas/ids/42", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"schema": schemaJSON})
	})
	srSrv := httptest.NewServer(srMux)
	defer srSrv.Close()

	srClient, err := sr.NewClient(sr.URLs(srSrv.URL))
	require.NoError(t, err)

	// Encode a Camp.V1 record in Confluent wire format with schemaID=42.
	schema, err := avro.Parse(schemaJSON)
	require.NoError(t, err)
	id := uuid.NewString()
	bin, err := avro.Marshal(schema, map[string]any{
		"id":          id,
		"name":        "Summer Football Camp",
		"sport":       "football",
		"location":    "Bradenton, FL",
		"capacity":    int32(60),
		"start_date":  time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC),
		"end_date":    time.Date(2026, 6, 22, 0, 0, 0, 0, time.UTC),
		"occurred_at": time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)
	wire := append([]byte{0x00, 0, 0, 0, 0}, bin...)
	binary.BigEndian.PutUint32(wire[1:5], schemaID)

	idx := &fakeIndexer{}
	handler := handlers.NewCampHandler(events.NewAvroDecoder(srClient), idx)

	require.NoError(t, handler.Handle(message.NewMessage(uuid.NewString(), wire)))

	require.Len(t, idx.indexed, 1)
	got := idx.indexed[0]
	require.Equal(t, id, got.ID)
	require.Equal(t, "football", got.Sport)
	require.Equal(t, int32(60), got.Capacity)
	require.Equal(t, "camp:"+id, domain.BuildCampDocID(got.ID))
}
