package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/matsumo_and/cogi/internal/config"
	"github.com/matsumo_and/cogi/internal/db"
	"github.com/matsumo_and/cogi/internal/embedding"
	"github.com/matsumo_and/cogi/internal/search"
	"github.com/matsumo_and/cogi/internal/vector"
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
		defer database.Close()

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
		defer database.Close()

		// Initialize Ollama embedding client
		embedClient := embedding.NewOllamaClient(
			cfg.Embedding.Endpoint,
			cfg.Embedding.Model,
			cfg.Embedding.Dimension,
		)

		// Initialize Qdrant vector client
		vectorClient, err := vector.NewClient(
			cfg.Qdrant.Endpoint,
			cfg.Qdrant.CollectionName,
			cfg.Embedding.Dimension,
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error connecting to Qdrant: %v\n", err)
			fmt.Println("\n⚠️  Make sure Qdrant is running:")
			fmt.Println("   docker run -p 6333:6333 qdrant/qdrant")
			os.Exit(1)
		}
		defer vectorClient.Close()

		// Ensure collection exists
		ctx := context.Background()
		if err := vectorClient.EnsureCollection(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Error ensuring Qdrant collection: %v\n", err)
			os.Exit(1)
		}

		// Create semantic searcher
		searcher := search.NewSemanticSearcher(database, vectorClient, embedClient)

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
}
