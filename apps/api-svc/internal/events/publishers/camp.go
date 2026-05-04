// Package publishers wires a Watermill+Kafka publisher with Confluent Avro
// wire-format encoding. One CampPublisher exists per topic.
package publishers

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/IBM/sarama"
	"github.com/google/uuid"
	"github.com/hamba/avro/v2"
	"github.com/twmb/franz-go/pkg/sr"
)

const (
	CampTopic   = "Camp.V1"
	CampSubject = "Camp.V1-value"

	keyMetadataKey = "kafka_partition_key"
)

// CampEvent is the Avro-shaped payload for Camp.V1.
type CampEvent struct {
	ID         string    `avro:"id"`
	Name       string    `avro:"name"`
	Sport      string    `avro:"sport"`
	Location   string    `avro:"location"`
	Capacity   int32     `avro:"capacity"`
	StartDate  time.Time `avro:"start_date"`
	EndDate    time.Time `avro:"end_date"`
	OccurredAt time.Time `avro:"occurred_at"`
}

// CampPublisher publishes Camp events to Kafka with Confluent-format Avro.
type CampPublisher struct {
	pub      message.Publisher
	schema   avro.Schema
	schemaID uint32
}

// NewCampPublisher reads the Avro schema, ensures it's registered with SR,
// and constructs a Watermill kafka Publisher keyed on the Camp ID.
func NewCampPublisher(
	ctx context.Context,
	brokers []string,
	srClient *sr.Client,
	rawSchema string,
	logger watermill.LoggerAdapter,
) (*CampPublisher, error) {
	schema, err := avro.Parse(rawSchema)
	if err != nil {
		return nil, fmt.Errorf("parse schema: %w", err)
	}

	subjectSchema, err := srClient.CreateSchema(ctx, CampSubject, sr.Schema{
		Schema: rawSchema,
		Type:   sr.TypeAvro,
	})
	if err != nil {
		return nil, fmt.Errorf("register schema: %w", err)
	}

	saramaCfg := sarama.NewConfig()
	saramaCfg.Producer.Return.Successes = true
	saramaCfg.Producer.RequiredAcks = sarama.WaitForAll

	pub, err := kafka.NewPublisher(
		kafka.PublisherConfig{
			Brokers:               brokers,
			Marshaler:             keyAwareMarshaler{},
			OverwriteSaramaConfig: saramaCfg,
		},
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("kafka publisher: %w", err)
	}

	return &CampPublisher{
		pub:      pub,
		schema:   schema,
		schemaID: uint32(subjectSchema.ID), //nolint:gosec // registry IDs are small positive integers
	}, nil
}

// Publish encodes the event in Confluent Avro wire format and publishes it.
func (p *CampPublisher) Publish(ctx context.Context, ev CampEvent) error {
	payload, err := avro.Marshal(p.schema, ev)
	if err != nil {
		return fmt.Errorf("avro marshal: %w", err)
	}

	out := make([]byte, 5+len(payload))
	out[0] = 0x00 // Confluent magic byte.
	binary.BigEndian.PutUint32(out[1:5], p.schemaID)
	copy(out[5:], payload)

	msg := message.NewMessage(uuid.NewString(), out)
	msg.Metadata.Set(keyMetadataKey, ev.ID)
	return p.pub.Publish(CampTopic, msg)
}

// Close flushes pending messages and shuts down the producer.
func (p *CampPublisher) Close() error { return p.pub.Close() }

// keyAwareMarshaler writes the partition key from message metadata so all
// events for one camp land on the same Kafka partition.
type keyAwareMarshaler struct{}

func (keyAwareMarshaler) Marshal(topic string, msg *message.Message) (*sarama.ProducerMessage, error) {
	headers := []sarama.RecordHeader{
		{Key: []byte(kafka.UUIDHeaderKey), Value: []byte(msg.UUID)},
	}
	for k, v := range msg.Metadata {
		if k == keyMetadataKey {
			continue
		}
		headers = append(headers, sarama.RecordHeader{Key: []byte(k), Value: []byte(v)})
	}
	pm := &sarama.ProducerMessage{
		Topic:   topic,
		Value:   sarama.ByteEncoder(msg.Payload),
		Headers: headers,
	}
	if k := msg.Metadata.Get(keyMetadataKey); k != "" {
		pm.Key = sarama.StringEncoder(k)
	}
	return pm, nil
}

func (keyAwareMarshaler) Unmarshal(*sarama.ConsumerMessage) (*message.Message, error) {
	return nil, fmt.Errorf("keyAwareMarshaler is publish-only")
}
