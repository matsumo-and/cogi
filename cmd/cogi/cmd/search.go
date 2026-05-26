package cmd

import (
	"fmt"
	"os"

	"github.com/matsumo_and/cogi/internal/config"
	"github.com/matsumo_and/cogi/internal/db"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search code using various methods",
	Long: `Search code using symbol search or keyword search.

Use subcommands to specify the search method:
  - symbol:  Search for symbols (functions, classes, variables)
  - keyword: Full-text search using SQLite FTS5`,
}

// Symbol search
var (
	symbolKind string
	symbolRepo string
)

var searchSymbolCmd = &cobra.Command{
	Use:   "symbol <query>",
	Short: "Search for symbols",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := args[0]

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

		// Search symbols
		var symbols []*db.Symbol
		if symbolKind != "" {
			symbols, err = database.SearchSymbolsByKind(symbolKind)
		} else {
			symbols, err = database.SearchSymbolsByName(query)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error searching symbols: %v\n", err)
			os.Exit(1)
		}

		if len(symbols) == 0 {
			fmt.Println("No symbols found.")
			return
		}

		fmt.Printf("\n━━━ Found %d symbol(s) ━━━\n\n", len(symbols))

		for _, sym := range symbols {
			file, err := database.GetFile(sym.FileID)
			if err != nil {
				continue
			}

			repo, err := database.GetRepository(file.RepositoryID)
			if err != nil {
				continue
			}

			fmt.Printf("📍 %s (%s)\n", sym.Name, sym.Kind)
			fmt.Printf("   %s:%s:%d\n", repo.Name, file.Path, sym.StartLine)
			if sym.Signature != "" {
				fmt.Printf("   %s\n", sym.Signature)
			}
			fmt.Println()
		}
	},
}

// Keyword search
var (
	keywordLang string
	keywordRepo string
)

var searchKeywordCmd = &cobra.Command{
	Use:   "keyword <query>",
	Short: "Full-text keyword search",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := args[0]

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

		// Perform full-text search
		symbols, err := database.FullTextSearch(query, 20)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error performing search: %v\n", err)
			os.Exit(1)
		}

		if len(symbols) == 0 {
			fmt.Println("No results found.")
			return
		}

		fmt.Printf("\n━━━ Found %d result(s) ━━━\n\n", len(symbols))

		for _, sym := range symbols {
			file, err := database.GetFile(sym.FileID)
			if err != nil {
				continue
			}

			repo, err := database.GetRepository(file.RepositoryID)
			if err != nil {
				continue
			}

			// Apply language filter if specified
			if keywordLang != "" && file.Language != keywordLang {
				continue
			}

			// Apply repository filter if specified
			if keywordRepo != "" && repo.Name != keywordRepo {
				continue
			}

			fmt.Printf("📍 %s (%s)\n", sym.Name, sym.Kind)
			fmt.Printf("   %s:%s:%d\n", repo.Name, file.Path, sym.StartLine)
			if sym.Docstring != "" {
				fmt.Printf("   %s\n", sym.Docstring)
			}
			fmt.Println()
		}
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)

	// Symbol search
	searchCmd.AddCommand(searchSymbolCmd)
	searchSymbolCmd.Flags().StringVarP(&symbolKind, "kind", "k", "", "Filter by symbol kind (function, class, etc.)")
	searchSymbolCmd.Flags().StringVarP(&symbolRepo, "repo", "r", "", "Filter by repository")

	// Keyword search
	searchCmd.AddCommand(searchKeywordCmd)
	searchKeywordCmd.Flags().StringVarP(&keywordLang, "lang", "l", "", "Filter by language")
	searchKeywordCmd.Flags().StringVarP(&keywordRepo, "repo", "r", "", "Filter by repository")
}
