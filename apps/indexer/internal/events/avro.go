// Package events provides Avro decoding for Confluent-format Kafka payloads.
package events

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"

	"github.com/hamba/avro/v2"
	"github.com/twmb/franz-go/pkg/sr"
)

// AvroDecoder decodes Confluent-wire-format payloads (magic byte + 4-byte
// schema ID + Avro binary). It fetches schemas lazily by ID and caches them.
type AvroDecoder struct {
	sr     *sr.Client
	mu     sync.RWMutex
	cached map[uint32]avro.Schema
}

func NewAvroDecoder(srClient *sr.Client) *AvroDecoder {
	return &AvroDecoder{sr: srClient, cached: map[uint32]avro.Schema{}}
}

// Decode unmarshals payload into v using the schema referenced by its header.
func (d *AvroDecoder) Decode(ctx context.Context, payload []byte, v any) error {
	if len(payload) < 5 || payload[0] != 0x00 {
		return errors.New("payload missing Confluent magic byte / schema ID")
	}
	id := binary.BigEndian.Uint32(payload[1:5])

	d.mu.RLock()
	schema, ok := d.cached[id]
	d.mu.RUnlock()

	if !ok {
		fetched, err := d.sr.SchemaByID(ctx, int(id))
		if err != nil {
			return fmt.Errorf("fetch schema id=%d: %w", id, err)
		}
		schema, err = avro.Parse(fetched.Schema)
		if err != nil {
			return fmt.Errorf("parse schema id=%d: %w", id, err)
		}
		d.mu.Lock()
		d.cached[id] = schema
		d.mu.Unlock()
	}
	return avro.Unmarshal(schema, payload[5:], v)
}
