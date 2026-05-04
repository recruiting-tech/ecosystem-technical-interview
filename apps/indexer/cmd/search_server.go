package cmd

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

	"github.com/spf13/cobra"

	idxos "github.com/imgacademy/em-challenge/apps/indexer/internal/opensearch"
	"github.com/imgacademy/em-challenge/apps/indexer/internal/searchapi"
)

func newSearchServerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "search-server",
		Short: "Run the HTTP /search endpoint backed by OpenSearch",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer cancel()
			return runSearchServer(ctx, loadConfig())
		},
	}
}

func runSearchServer(ctx context.Context, cfg config) error {
	osClient, err := idxos.New(cfg.OpenSearchURL)
	if err != nil {
		return fmt.Errorf("opensearch: %w", err)
	}
	if err := osClient.EnsureCampsIndex(ctx); err != nil {
		return fmt.Errorf("ensure index: %w", err)
	}
	server := searchapi.NewServer(osClient)
	httpSrv := &http.Server{
		Addr:              cfg.SearchAddr,
		Handler:           withCORS(server.Routes()),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		slog.Info("search-server listening", "addr", cfg.SearchAddr)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("search-server", "err", err)
		}
	}()
	<-ctx.Done()
	shutCtx, c := context.WithTimeout(context.Background(), 5*time.Second)
	defer c()
	return httpSrv.Shutdown(shutCtx)
}

func withCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})
}
