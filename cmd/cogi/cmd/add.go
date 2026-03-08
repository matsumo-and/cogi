package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/matsumo_and/cogi/internal/config"
	"github.com/matsumo_and/cogi/internal/db"
	"github.com/spf13/cobra"
)

var addName string

var addCmd = &cobra.Command{
	Use:   "add <repo-path>",
	Short: "Add a repository to the index",
	Long: `Add a repository to Cogi's index.

The repository will be scanned and indexed for code intelligence features.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repoPath := args[0]
		name := addName

		// Get absolute path
		absPath, err := filepath.Abs(repoPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid path: %v\n", err)
			os.Exit(1)
		}

		// Check if path exists
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error: path does not exist: %s\n", absPath)
			os.Exit(1)
		}

		// Use basename as default name
		if name == "" {
			name = filepath.Base(absPath)
		}

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

		// Check if repository already exists
		exists, err := database.RepositoryExists(name, absPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error checking repository: %v\n", err)
			os.Exit(1)
		}
		if exists {
			fmt.Fprintf(os.Stderr, "Error: repository '%s' already exists\n", name)
			os.Exit(1)
		}

		// Create repository
		repo, err := database.CreateRepository(name, absPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating repository: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✓ Added repository: %s (ID: %d)\n", name, repo.ID)
		fmt.Printf("  Path: %s\n", absPath)
		fmt.Printf("\nRun 'cogi index' to start indexing.\n")
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().StringVarP(&addName, "name", "n", "", "Custom name for the repository")
}
