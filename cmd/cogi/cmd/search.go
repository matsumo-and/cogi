package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/matsumo_and/cogi/internal/config"
	"github.com/matsumo_and/cogi/internal/db"
	"github.com/matsumo_and/cogi/internal/embedding"
	"github.com/matsumo_and/cogi/internal/search"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search code using various methods",
	Long: `Search code using symbol search, keyword search, or semantic search.

Use subcommands to specify the search method:
  - symbol:   Search for symbols (functions, classes, variables)
  - keyword:  Full-text search using SQLite FTS5
  - semantic: Semantic search using vector embeddings
  - hybrid:   Hybrid search combining keyword and semantic search`,
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

// Semantic search
var (
	semanticGranularity string
	semanticLimit       int
	semanticLang        string
	semanticRepo        string
)

// Hybrid search
var (
	hybridKind      string
	hybridLimit     int
	hybridLang      string
	hybridRepo      string
	hybridKwWeight  float64
	hybridSemWeight float64
)

var searchHybridCmd = &cobra.Command{
	Use:   "hybrid <query>",
	Short: "Hybrid search combining keyword and semantic search",
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

		// Initialize Ollama embedding client
		embedClient := embedding.NewOllamaClient(
			cfg.Embedding.Endpoint,
			cfg.Embedding.Model,
			cfg.Embedding.Dimension,
		)

		// Create searchers
		ctx := context.Background()
		keywordSearcher := search.NewKeywordSearcher(database)
		semanticSearcher := search.NewSemanticSearcher(database, embedClient)
		hybridSearcher := search.NewHybridSearcher(keywordSearcher, semanticSearcher)

		// Perform hybrid search
		results, err := hybridSearcher.Search(ctx, search.HybridSearchOptions{
			Query:          query,
			SymbolKind:     hybridKind,
			Language:       hybridLang,
			Repository:     hybridRepo,
			Limit:          hybridLimit,
			KeywordWeight:  float32(hybridKwWeight),
			SemanticWeight: float32(hybridSemWeight),
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error performing hybrid search: %v\n", err)
			fmt.Println("\n⚠️  Make sure Ollama is running:")
			fmt.Println("   ollama serve")
			fmt.Printf("   ollama pull %s\n", cfg.Embedding.Model)
			os.Exit(1)
		}

		if len(results) == 0 {
			fmt.Println("\nNo results found.")
			fmt.Println("\nℹ️  Note: You need to build embeddings first:")
			fmt.Println("   cogi index")
			return
		}

		fmt.Printf("\n━━━ Found %d result(s) ━━━\n", len(results))
		fmt.Printf("(Keyword weight: %.1f, Semantic weight: %.1f)\n\n", hybridKwWeight, hybridSemWeight)

		for i, result := range results {
			fmt.Printf("%d. %s (%s) - Score: %.4f\n", i+1, result.SymbolName, result.SymbolKind, result.Score)
			fmt.Printf("   %s:%d\n", result.FilePath, result.StartLine)
			if result.Signature != "" {
				fmt.Printf("   %s\n", result.Signature)
			}
			if result.Docstring != "" {
				fmt.Printf("   %s\n", result.Docstring)
			}
			fmt.Println()
		}
	},
}

var searchSemanticCmd = &cobra.Command{
	Use:   "semantic <query>",
	Short: "Semantic search using vector embeddings",
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

		// Initialize Ollama embedding client
		embedClient := embedding.NewOllamaClient(
			cfg.Embedding.Endpoint,
			cfg.Embedding.Model,
			cfg.Embedding.Dimension,
		)

		// Create semantic searcher (SQLite-based vector search)
		ctx := context.Background()
		searcher := search.NewSemanticSearcher(database, embedClient)

		// Perform semantic search
		results, err := searcher.Search(ctx, search.SemanticSearchOptions{
			Query:       query,
			Granularity: semanticGranularity,
			Language:    semanticLang,
			Repository:  semanticRepo,
			Limit:       semanticLimit,
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error performing semantic search: %v\n", err)
			fmt.Println("\n⚠️  Make sure Ollama is running:")
			fmt.Println("   ollama serve")
			fmt.Printf("   ollama pull %s\n", cfg.Embedding.Model)
			os.Exit(1)
		}

		if len(results) == 0 {
			fmt.Println("\nNo results found.")
			fmt.Println("\nℹ️  Note: You need to build embeddings first:")
			fmt.Println("   cogi index --embeddings")
			return
		}

		fmt.Printf("\n━━━ Found %d result(s) ━━━\n\n", len(results))

		for i, result := range results {
			fmt.Printf("%d. %s (%s) - Score: %.4f\n", i+1, result.SymbolName, result.SymbolKind, result.Score)
			fmt.Printf("   %s:%d\n", result.FilePath, result.StartLine)
			if result.Signature != "" {
				fmt.Printf("   %s\n", result.Signature)
			}
			if result.Docstring != "" {
				fmt.Printf("   %s\n", result.Docstring)
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

	// Semantic search
	searchCmd.AddCommand(searchSemanticCmd)
	searchSemanticCmd.Flags().StringVarP(&semanticGranularity, "granularity", "g", "", "Search granularity (class or function, empty for both)")
	searchSemanticCmd.Flags().IntVarP(&semanticLimit, "limit", "n", 10, "Maximum number of results")
	searchSemanticCmd.Flags().StringVarP(&semanticLang, "lang", "l", "", "Filter by language")
	searchSemanticCmd.Flags().StringVarP(&semanticRepo, "repo", "r", "", "Filter by repository")

	// Hybrid search
	searchCmd.AddCommand(searchHybridCmd)
	searchHybridCmd.Flags().StringVarP(&hybridKind, "kind", "k", "", "Filter by symbol kind (function, class, etc.)")
	searchHybridCmd.Flags().IntVarP(&hybridLimit, "limit", "n", 10, "Maximum number of results")
	searchHybridCmd.Flags().StringVarP(&hybridLang, "lang", "l", "", "Filter by language")
	searchHybridCmd.Flags().StringVarP(&hybridRepo, "repo", "r", "", "Filter by repository")
	searchHybridCmd.Flags().Float64Var(&hybridKwWeight, "kw-weight", 0.3, "Keyword search weight (0.0-1.0)")
	searchHybridCmd.Flags().Float64Var(&hybridSemWeight, "sem-weight", 0.7, "Semantic search weight (0.0-1.0)")
}
