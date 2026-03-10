package cmd

import (
	"fmt"
	"os"

	"github.com/matsumo_and/cogi/internal/config"
	"github.com/matsumo_and/cogi/internal/db"
	"github.com/matsumo_and/cogi/internal/export"
	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export indexed data to JSON format",
	Long: `Export indexed data (symbols, call graph, import graph, ownership) to JSON format.
This allows you to analyze the data externally or share it with other tools.`,
	RunE: runExport,
}

var (
	exportOutput     string
	exportType       string
	exportRepository string
)

func init() {
	rootCmd.AddCommand(exportCmd)

	exportCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "Output file path (default: stdout)")
	exportCmd.Flags().StringVarP(&exportType, "type", "t", "all", "Export type: all, symbols, call-graph, import-graph, ownership")
	exportCmd.Flags().StringVarP(&exportRepository, "repo", "r", "", "Filter by repository name (optional)")
}

func runExport(cmd *cobra.Command, args []string) error {
	// Load config
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Open database
	database, err := db.Open(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer func() { _ = database.Close() }()

	// Create exporter
	exporter := export.New(database)

	// Determine output writer
	writer := os.Stdout
	if exportOutput != "" {
		file, err := os.Create(exportOutput)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer func() { _ = file.Close() }()
		writer = file
	}

	// Get repository ID if filter is specified
	var repositoryID *int64
	if exportRepository != "" {
		repo, err := database.GetRepositoryByName(exportRepository)
		if err != nil {
			return fmt.Errorf("repository not found: %s", exportRepository)
		}
		repositoryID = &repo.ID
	}

	// Export based on type
	format := export.FormatJSON

	switch exportType {
	case "all":
		if err := exporter.ExportAll(writer, format); err != nil {
			return fmt.Errorf("failed to export data: %w", err)
		}
	case "symbols":
		if err := exporter.ExportSymbols(writer, format, repositoryID); err != nil {
			return fmt.Errorf("failed to export symbols: %w", err)
		}
	default:
		return fmt.Errorf("unsupported export type: %s (supported: all, symbols)", exportType)
	}

	if exportOutput != "" {
		fmt.Printf("Data exported to: %s\n", exportOutput)
	}

	return nil
}
