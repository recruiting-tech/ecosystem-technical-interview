package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/IBM/sarama"
	"github.com/ThreeDotsLabs/watermill"
	wmkafka "github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/spf13/cobra"
	"github.com/twmb/franz-go/pkg/sr"

	"github.com/imgacademy/em-challenge/apps/indexer/internal/events"
	"github.com/imgacademy/em-challenge/apps/indexer/internal/handlers"
	idxos "github.com/imgacademy/em-challenge/apps/indexer/internal/opensearch"
)

func newConsumeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "consume",
		Short: "Consume Kafka events and index into OpenSearch",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer cancel()
			return runConsume(ctx, loadConfig())
		},
	}
}

func runConsume(ctx context.Context, cfg config) error {
	logger := watermill.NewSlogLogger(slog.Default())

	osClient, err := idxos.New(cfg.OpenSearchURL)
	if err != nil {
		return fmt.Errorf("opensearch: %w", err)
	}
	if err := osClient.EnsureCampsIndex(ctx); err != nil {
		return fmt.Errorf("ensure index: %w", err)
	}

	srClient, err := sr.NewClient(sr.URLs(cfg.SchemaRegistryURL))
	if err != nil {
		return fmt.Errorf("sr client: %w", err)
	}
	dec := events.NewAvroDecoder(srClient)

	router, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		return err
	}

	saramaCfg := sarama.NewConfig()
	saramaCfg.Version = sarama.V3_5_0_0
	saramaCfg.Consumer.Offsets.Initial = sarama.OffsetOldest

	sub, err := wmkafka.NewSubscriber(
		wmkafka.SubscriberConfig{
			Brokers:               cfg.KafkaBrokers,
			Unmarshaler:           wmkafka.DefaultMarshaler{},
			OverwriteSaramaConfig: saramaCfg,
			ConsumerGroup:         cfg.ConsumerGroup,
		},
		logger,
	)
	if err != nil {
		return fmt.Errorf("subscriber: %w", err)
	}

	camp := handlers.NewCampHandler(dec, osClient)
	router.AddNoPublisherHandler("camp.v1", "Camp.V1", sub, camp.Handle)

	return router.Run(ctx)
}
