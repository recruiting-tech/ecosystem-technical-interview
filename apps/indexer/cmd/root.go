// Package cmd is the Cobra root for the indexer CLI.
package cmd

import (
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	root := &cobra.Command{
		Use:   "indexer",
		Short: "Indexer service: consumes Camp events and indexes into OpenSearch.",
	}
	root.AddCommand(newConsumeCmd())
	root.AddCommand(newSearchServerCmd())
	root.AddCommand(newRunCmd())
	return root
}
