package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/matsumo_and/cogi/internal/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP (Model Context Protocol) server",
	Long: `Start an MCP server that exposes Cogi's search capabilities via the Model Context Protocol.

The server uses stdio transport and can be integrated with MCP clients like Claude Desktop.

Available tools:
  - cogi_search_symbol:   Search for code symbols by name or kind
  - cogi_search_keyword:  Full-text keyword search using SQLite FTS5
  - cogi_search_semantic: Semantic search using vector embeddings
  - cogi_search_hybrid:   Hybrid search combining keyword and semantic

Example Claude Desktop configuration (~/.config/Claude/claude_desktop_config.json):
  {
    "mcpServers": {
      "cogi": {
        "command": "cogi",
        "args": ["mcp"]
      }
    }
  }`,
	Run: func(cmd *cobra.Command, args []string) {
		// Create context with cancellation
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Set up signal handling for graceful shutdown
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		go func() {
			<-sigChan
			fmt.Fprintln(os.Stderr, "\nReceived shutdown signal, stopping MCP server...")
			cancel()
		}()

		// Create and start MCP server
		server, err := mcp.New()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating MCP server: %v\n", err)
			os.Exit(1)
		}

		fmt.Fprintln(os.Stderr, "Starting Cogi MCP server...")
		fmt.Fprintln(os.Stderr, "Server ready. Listening on stdio...")

		if err := server.Start(ctx); err != nil {
			if err != context.Canceled {
				fmt.Fprintf(os.Stderr, "Error running MCP server: %v\n", err)
				os.Exit(1)
			}
		}

		fmt.Fprintln(os.Stderr, "MCP server stopped.")
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
