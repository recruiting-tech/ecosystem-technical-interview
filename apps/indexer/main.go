package main

import (
	"log/slog"
	"os"

	"github.com/imgacademy/ecosystem-technical-interview/apps/indexer/cmd"
)

func main() {
	if err := cmd.New().Execute(); err != nil {
		slog.Error("indexer exited", "err", err)
		os.Exit(1)
	}
}
