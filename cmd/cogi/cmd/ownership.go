package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/matsumo_and/cogi/internal/config"
	"github.com/matsumo_and/cogi/internal/db"
	"github.com/spf13/cobra"
)

var (
	ownershipFile   string
	ownershipLine   int
	ownershipAuthor string
	ownershipTop    int
)

var ownershipCmd = &cobra.Command{
	Use:   "ownership",
	Short: "Show code ownership information",
	Long: `Show code ownership information based on git blame.

This command analyzes git history to determine who owns which parts of the code.`,
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
		defer database.Close()

		// Handle different query modes
		if ownershipAuthor != "" {
			showAuthorOwnership(database, ownershipAuthor)
		} else if ownershipTop > 0 {
			showTopAuthors(database, ownershipTop)
		} else if ownershipFile != "" {
			showFileOwnership(database, ownershipFile, ownershipLine)
		} else {
			fmt.Println("Please specify --file, --author, or --top")
			cmd.Help()
			os.Exit(1)
		}
	},
}

func showFileOwnership(database *db.DB, filePath string, lineNumber int) {
	// Get file ID
	// We need to search across all repositories
	files, err := database.ListFiles()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing files: %v\n", err)
		os.Exit(1)
	}

	var fileID int64
	var found bool
	for _, file := range files {
		if filepath.Base(file.Path) == filepath.Base(filePath) || file.Path == filePath {
			fileID = file.ID
			found = true
			break
		}
	}

	if !found {
		fmt.Printf("File not found: %s\n", filePath)
		os.Exit(1)
	}

	if lineNumber > 0 {
		// Show ownership for specific line
		ownership, err := database.GetOwnershipByLine(fileID, lineNumber)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting ownership: %v\n", err)
			os.Exit(1)
		}

		if ownership == nil {
			fmt.Printf("No ownership information for line %d\n", lineNumber)
			return
		}

		fmt.Printf("Line %d ownership:\n", lineNumber)
		fmt.Printf("  Author: %s <%s>\n", ownership.AuthorName, ownership.AuthorEmail)
		fmt.Printf("  Last modified: %s\n", ownership.LastCommitDate.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Commit: %s\n", ownership.LastCommitHash[:8])
		fmt.Printf("  Lines: %d-%d (%d lines)\n", ownership.StartLine, ownership.EndLine, ownership.EndLine-ownership.StartLine+1)
	} else {
		// Show all ownership ranges for the file
		ownerships, err := database.GetOwnershipByFile(fileID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting ownership: %v\n", err)
			os.Exit(1)
		}

		if len(ownerships) == 0 {
			fmt.Printf("No ownership information for file: %s\n", filePath)
			return
		}

		fmt.Printf("Ownership for %s:\n\n", filePath)
		fmt.Printf("%-20s %-30s %s\n", "Lines", "Author", "Last Modified")
		fmt.Println("────────────────────────────────────────────────────────────────────────")

		for _, o := range ownerships {
			linesStr := fmt.Sprintf("%d-%d", o.StartLine, o.EndLine)
			fmt.Printf("%-20s %-30s %s\n",
				linesStr,
				truncate(o.AuthorName, 30),
				o.LastCommitDate.Format("2006-01-02"))
		}
	}
}

func showAuthorOwnership(database *db.DB, authorName string) {
	ownerships, err := database.GetOwnershipByAuthor(authorName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting ownership: %v\n", err)
		os.Exit(1)
	}

	if len(ownerships) == 0 {
		fmt.Printf("No ownership information for author: %s\n", authorName)
		return
	}

	// Group by file
	fileMap := make(map[int64][]int)
	for _, o := range ownerships {
		fileMap[o.FileID] = append(fileMap[o.FileID], o.EndLine-o.StartLine+1)
	}

	fmt.Printf("Files owned by %s:\n\n", authorName)
	fmt.Printf("%-50s %s\n", "File", "Lines")
	fmt.Println("────────────────────────────────────────────────────────────────────────")

	for fileID, lineCounts := range fileMap {
		file, err := database.GetFile(fileID)
		if err != nil {
			continue
		}

		totalLines := 0
		for _, count := range lineCounts {
			totalLines += count
		}

		fmt.Printf("%-50s %d\n", truncate(file.Path, 50), totalLines)
	}
}

func showTopAuthors(database *db.DB, limit int) {
	authors, err := database.GetTopAuthors(limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting top authors: %v\n", err)
		os.Exit(1)
	}

	if len(authors) == 0 {
		fmt.Println("No ownership information available")
		return
	}

	fmt.Printf("Top %d contributors:\n\n", limit)
	fmt.Printf("%-30s %-10s %-10s %s\n", "Author", "Files", "Commits", "Last Commit")
	fmt.Println("────────────────────────────────────────────────────────────────────────")

	for _, a := range authors {
		fmt.Printf("%-30s %-10d %-10d %s\n",
			truncate(a.AuthorName, 30),
			a.FileCount,
			a.TotalCommits,
			a.LastCommit.Format("2006-01-02"))
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func init() {
	rootCmd.AddCommand(ownershipCmd)

	ownershipCmd.Flags().StringVarP(&ownershipFile, "file", "f", "", "Show ownership for a specific file")
	ownershipCmd.Flags().IntVarP(&ownershipLine, "line", "l", 0, "Show ownership for a specific line number")
	ownershipCmd.Flags().StringVarP(&ownershipAuthor, "author", "a", "", "Show all files owned by an author")
	ownershipCmd.Flags().IntVarP(&ownershipTop, "top", "t", 0, "Show top N contributors")
}
