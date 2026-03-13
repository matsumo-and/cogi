package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/matsumo_and/cogi/internal/config"
	"github.com/matsumo_and/cogi/internal/db"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all indexed repositories",
	Long: `List all repositories that have been added to Cogi's index.

Shows repository name, path, and last indexed time.`,
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

		// List repositories
		repos, err := database.ListRepositories()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing repositories: %v\n", err)
			os.Exit(1)
		}

		if len(repos) == 0 {
			fmt.Println("No repositories indexed yet.")
			fmt.Println("Use 'cogi add <path>' to add a repository.")
			return
		}

		// Create table writer
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "NAME\tPATH\tLAST INDEXED")
		_, _ = fmt.Fprintln(w, "────\t────\t────────────")

		for _, repo := range repos {
			lastIndexed := "Never"
			if repo.LastIndexedAt != nil {
				lastIndexed = repo.LastIndexedAt.Format("2006-01-02 15:04:05")
			}
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", repo.Name, repo.Path, lastIndexed)
		}

		if w.Flush() != nil {
			fmt.Fprintf(os.Stderr, "Error flushing writer: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
