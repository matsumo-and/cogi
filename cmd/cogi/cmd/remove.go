package cmd

import (
	"fmt"
	"os"

	"github.com/matsumo_and/cogi/internal/config"
	"github.com/matsumo_and/cogi/internal/db"
	"github.com/spf13/cobra"
)

var removeForce bool

var removeCmd = &cobra.Command{
	Use:   "remove <repo-name>",
	Short: "Remove a repository from the index",
	Long: `Remove a repository from Cogi's index.

This will delete all indexed data for the repository, including symbols,
embeddings, and ownership information.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repoName := args[0]

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
		defer func() { _ = database.Close() }()

		// Get repository
		repo, err := database.GetRepositoryByName(repoName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: repository '%s' not found\n", repoName)
			os.Exit(1)
		}

		// Confirm deletion unless --force is specified
		if !removeForce {
			fmt.Printf("Are you sure you want to remove repository '%s'? (y/N): ", repoName)
			var response string
			fmt.Scanln(&response)
			if response != "y" && response != "Y" {
				fmt.Println("Cancelled.")
				return
			}
		}

		// Delete repository
		err = database.DeleteRepository(repo.ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting repository: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✓ Removed repository: %s\n", repoName)
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
	removeCmd.Flags().BoolVarP(&removeForce, "force", "f", false, "Skip confirmation prompt")
}
