// Package main runs the api-svc HTTP server.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"
	"github.com/twmb/franz-go/pkg/sr"

	"github.com/imgacademy/ecosystem-technical-interview/apps/api-svc/db/migrations"
	"github.com/imgacademy/ecosystem-technical-interview/apps/api-svc/internal/api"
	"github.com/imgacademy/ecosystem-technical-interview/apps/api-svc/internal/events/publishers"
	"github.com/imgacademy/ecosystem-technical-interview/apps/api-svc/internal/repository/postgres"
	"github.com/imgacademy/ecosystem-technical-interview/apps/api-svc/oapi"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server exited", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := loadConfig()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("pgx pool: %w", err)
	}
	defer pool.Close()

	if err := runMigrations(ctx, cfg.DatabaseURL); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	rawSchema, err := os.ReadFile(cfg.CampSchemaPath)
	if err != nil {
		return fmt.Errorf("read schema: %w", err)
	}
	srClient, err := sr.NewClient(sr.URLs(cfg.SchemaRegistryURL))
	if err != nil {
		return fmt.Errorf("sr client: %w", err)
	}
	pub, err := publishers.NewCampPublisher(
		ctx,
		cfg.KafkaBrokers,
		srClient,
		string(rawSchema),
		watermill.NewSlogLogger(slog.Default()),
	)
	if err != nil {
		return fmt.Errorf("publisher: %w", err)
	}
	defer pub.Close()

	server := api.NewServer(postgres.New(pool), pub)
	r := gin.New()
	r.Use(gin.Recovery())
	oapi.RegisterHandlers(r, oapi.NewStrictHandler(server, nil /*no middlewares*/))

	httpSrv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		slog.Info("api-svc listening", "addr", cfg.Addr)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http server", "err", err)
			cancel()
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	return httpSrv.Shutdown(shutCtx)
}

func runMigrations(ctx context.Context, dbURL string) error {
	goose.SetBaseFS(migrations.Files)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	db, err := goose.OpenDBWithDriver("pgx", dbURL)
	if err != nil {
		return err
	}
	defer db.Close()
	return goose.UpContext(ctx, db, ".")
}

type config struct {
	Addr              string
	DatabaseURL       string
	KafkaBrokers      []string
	SchemaRegistryURL string
	CampSchemaPath    string
}

func loadConfig() config {
	get := func(k, def string) string {
		if v := os.Getenv(k); v != "" {
			return v
		}
		return def
	}
	return config{
		Addr:              get("ADDR", ":8001"),
		DatabaseURL:       get("DATABASE_URL", "postgres://emc:emc@localhost:5532/api_svc?sslmode=disable"),
		KafkaBrokers:      []string{get("KAFKA_BROKERS", "localhost:19092")},
		SchemaRegistryURL: get("SCHEMA_REGISTRY_URL", "http://localhost:18081"),
		CampSchemaPath:    get("CAMP_SCHEMA_PATH", "../../schemas/Camp.V1.avsc"),
	}
}
