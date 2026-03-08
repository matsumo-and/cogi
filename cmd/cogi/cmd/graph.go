package cmd

import (
	"fmt"
	"os"

	"github.com/matsumo_and/cogi/internal/config"
	"github.com/matsumo_and/cogi/internal/db"
	"github.com/matsumo_and/cogi/internal/graph"
	"github.com/spf13/cobra"
)

var graphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Visualize call and import graphs",
	Long: `Visualize call graphs and import graphs to understand code relationships.

Use subcommands to specify the graph type:
  - calls:   Display call graph (who calls whom)
  - imports: Display import graph (file dependencies)`,
}

// Call graph flags
var (
	callDepth     int
	callDirection string
)

var callsCmd = &cobra.Command{
	Use:   "calls <symbol-name>",
	Short: "Display call graph for a symbol",
	Long: `Display the call graph for a function or method.

Shows which functions/methods call the given symbol (callers) or
which functions/methods are called by the given symbol (callees).`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		symbolName := args[0]

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

		// Create graph manager
		gm := graph.New(database)

		// Get call graph based on direction
		var nodes []*graph.CallNode
		if callDirection == "caller" || callDirection == "callers" {
			nodes, err = gm.GetCallersTree(symbolName, callDepth)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting callers: %v\n", err)
				os.Exit(1)
			}
		} else if callDirection == "callee" || callDirection == "callees" {
			nodes, err = gm.GetCalleesTree(symbolName, callDepth)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting callees: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Fprintf(os.Stderr, "Invalid direction: %s. Use 'caller' or 'callee'\n", callDirection)
			os.Exit(1)
		}

		// Format and display results
		output := graph.FormatCallTree(nodes, callDirection)
		fmt.Println(output)
	},
}

// Import graph flags
var (
	importDepth     int
	importDirection string
)

var importsCmd = &cobra.Command{
	Use:   "imports <file-path>",
	Short: "Display import graph for a file",
	Long: `Display the import graph for a file.

Shows which files import the given file (importers) or
which files are imported by the given file (dependencies).`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]

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

		// Create graph manager
		gm := graph.New(database)

		// Get import graph based on direction
		var nodes []*graph.ImportNode
		if importDirection == "importer" || importDirection == "importers" {
			nodes, err = gm.GetImporters(filePath, importDepth)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting importers: %v\n", err)
				os.Exit(1)
			}
		} else if importDirection == "dependency" || importDirection == "dependencies" {
			nodes, err = gm.GetImportDependencies(filePath, importDepth)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting dependencies: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Fprintf(os.Stderr, "Invalid direction: %s. Use 'importer' or 'dependency'\n", importDirection)
			os.Exit(1)
		}

		// Format and display results
		output := graph.FormatImportTree(nodes, importDirection)
		fmt.Println(output)
	},
}

func init() {
	rootCmd.AddCommand(graphCmd)

	// Add subcommands
	graphCmd.AddCommand(callsCmd)
	graphCmd.AddCommand(importsCmd)

	// Call graph flags
	callsCmd.Flags().IntVarP(&callDepth, "depth", "d", 3, "Depth of call graph traversal")
	callsCmd.Flags().StringVarP(&callDirection, "direction", "D", "caller", "Direction of call graph: 'caller' or 'callee'")

	// Import graph flags
	importsCmd.Flags().IntVarP(&importDepth, "depth", "d", 3, "Depth of import graph traversal")
	importsCmd.Flags().StringVarP(&importDirection, "direction", "D", "dependency", "Direction of import graph: 'importer' or 'dependency'")
}
