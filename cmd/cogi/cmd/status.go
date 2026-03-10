package cmd

import (
	"fmt"
	"os"

	"github.com/matsumo_and/cogi/internal/config"
	"github.com/matsumo_and/cogi/internal/db"
	"github.com/matsumo_and/cogi/internal/indexer"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Cogi status and statistics",
	Long: `Display information about indexed repositories, database status,
and overall system health.`,
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
		defer func() { _ = database.Close() }()

		fmt.Print("\n━━━ Cogi Status ━━━\n\n")

		// List repositories
		repos, err := database.ListRepositories()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing repositories: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Indexed repositories: %d\n\n", len(repos))

		if len(repos) == 0 {
			fmt.Println("No repositories indexed yet.")
			fmt.Println("Use 'cogi add <path>' to add a repository.")
			return
		}

		// Show repository details
		idx := indexer.New(database, cfg)
		totalSymbols := int64(0)

		for _, repo := range repos {
			stats, err := idx.GetStats(repo.ID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting stats for %s: %v\n", repo.Name, err)
				continue
			}

			fmt.Printf("📁 %s\n", repo.Name)
			fmt.Printf("   Path: %s\n", repo.Path)
			fmt.Printf("   Files: %d\n", stats.TotalFiles)
			fmt.Printf("   Symbols: %d\n", stats.TotalSymbols)
			if stats.LastIndexed != nil {
				fmt.Printf("   Last indexed: %s\n", stats.LastIndexed.Format("2006-01-02 15:04:05"))
			} else {
				fmt.Printf("   Last indexed: Never\n")
			}
			fmt.Println()

			totalSymbols += stats.TotalSymbols
		}

		// Overall statistics
		fmt.Println("━━━━━━━━━━━━━━━━━━━━")
		fmt.Printf("Total symbols: %d\n", totalSymbols)
		fmt.Printf("Database: %s\n", cfg.Database.Path)

		// Get database file size
		if fileInfo, err := os.Stat(cfg.Database.Path); err == nil {
			sizeMB := float64(fileInfo.Size()) / 1024 / 1024
			fmt.Printf("Database size: %.2f MB\n", sizeMB)
		}

		fmt.Println()
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
