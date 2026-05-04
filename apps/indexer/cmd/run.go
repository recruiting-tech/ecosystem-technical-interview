package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func newRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Run consumer and search-server together (default for local dev)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer cancel()
			cfg := loadConfig()
			g, gctx := errgroup.WithContext(ctx)
			g.Go(func() error { return runConsume(gctx, cfg) })
			g.Go(func() error { return runSearchServer(gctx, cfg) })
			return g.Wait()
		},
	}
}

