package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search code using various methods",
	Long: `Search code using symbol search, keyword search, or semantic search.

Use subcommands to specify the search method:
  - symbol:   Search for symbols (functions, classes, variables)
  - keyword:  Full-text search using SQLite FTS5
  - semantic: Semantic search using vector embeddings`,
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
		fmt.Printf("Searching symbols for: %s\n", query)
		if symbolKind != "" {
			fmt.Printf("Kind filter: %s\n", symbolKind)
		}
		if symbolRepo != "" {
			fmt.Printf("Repository filter: %s\n", symbolRepo)
		}
		// TODO: Implement symbol search
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
		fmt.Printf("Keyword search for: %s\n", query)
		if keywordLang != "" {
			fmt.Printf("Language filter: %s\n", keywordLang)
		}
		if keywordRepo != "" {
			fmt.Printf("Repository filter: %s\n", keywordRepo)
		}
		// TODO: Implement keyword search with FTS5
	},
}

// Semantic search
var (
	semanticGranularity string
	semanticLimit       int
)

var searchSemanticCmd = &cobra.Command{
	Use:   "semantic <query>",
	Short: "Semantic search using vector embeddings",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := args[0]
		fmt.Printf("Semantic search for: %s\n", query)
		fmt.Printf("Granularity: %s, Limit: %d\n", semanticGranularity, semanticLimit)
		// TODO: Implement semantic search with Qdrant
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

	// Semantic search
	searchCmd.AddCommand(searchSemanticCmd)
	searchSemanticCmd.Flags().StringVarP(&semanticGranularity, "granularity", "g", "function", "Search granularity (class or function)")
	searchSemanticCmd.Flags().IntVarP(&semanticLimit, "limit", "n", 10, "Maximum number of results")
}
