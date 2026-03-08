package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/matsumo_and/cogi/internal/config"
	"github.com/matsumo_and/cogi/internal/db"
	"github.com/matsumo_and/cogi/internal/indexer"
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
		// Load config
		cfg, err := config.Load("")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		// Open database
		database, err := db.Open(cfg.Database.Path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
			os.Exit(1)
		}
		defer database.Close()

		// Create indexer
		idx := indexer.New(database, cfg)

		ctx := context.Background()

		// Get repositories to index
		var repos []*db.Repository
		if indexRepo != "" {
			repo, err := database.GetRepositoryByName(indexRepo)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: repository '%s' not found\n", indexRepo)
				os.Exit(1)
			}
			repos = []*db.Repository{repo}
		} else {
			repos, err = database.ListRepositories()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error listing repositories: %v\n", err)
				os.Exit(1)
			}
		}

		if len(repos) == 0 {
			fmt.Println("No repositories to index. Use 'cogi add' to add repositories.")
			return
		}

		// Index each repository
		for _, repo := range repos {
			fmt.Printf("\n━━━ Indexing: %s ━━━\n", repo.Name)
			fmt.Printf("Path: %s\n\n", repo.Path)

			err := idx.IndexRepository(ctx, repo.ID, repo.Path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error indexing repository: %v\n", err)
				continue
			}

			// Show statistics
			stats, err := idx.GetStats(repo.ID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting stats: %v\n", err)
				continue
			}

			fmt.Printf("\n✓ Indexing complete!\n")
			fmt.Printf("  Files indexed: %d\n", stats.TotalFiles)
			fmt.Printf("  Symbols found: %d\n", stats.TotalSymbols)
		}

		fmt.Println("\n✓ All repositories indexed successfully!")
	},
}

func init() {
	rootCmd.AddCommand(indexCmd)
	indexCmd.Flags().StringVarP(&indexRepo, "repo", "r", "", "Index specific repository by name")
	indexCmd.Flags().BoolVarP(&indexFull, "full", "f", false, "Force full reindex")
}
