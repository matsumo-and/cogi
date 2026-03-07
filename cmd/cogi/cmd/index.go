package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	indexRepo string
	indexFull bool
)

var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "Build or update the code index",
	Long: `Build or update the code index for repositories.

This command parses code files, extracts symbols, and creates various indices
including symbol index, call graph, import graph, and vector embeddings.`,
	Run: func(cmd *cobra.Command, args []string) {
		if indexFull {
			fmt.Println("Running full index rebuild...")
		} else {
			fmt.Println("Running incremental index update...")
		}

		if indexRepo != "" {
			fmt.Printf("Indexing repository: %s\n", indexRepo)
		} else {
			fmt.Println("Indexing all repositories...")
		}

		// TODO: Implement indexing logic
	},
}

func init() {
	rootCmd.AddCommand(indexCmd)
	indexCmd.Flags().StringVarP(&indexRepo, "repo", "r", "", "Index specific repository by name")
	indexCmd.Flags().BoolVarP(&indexFull, "full", "f", false, "Force full reindex")
}
