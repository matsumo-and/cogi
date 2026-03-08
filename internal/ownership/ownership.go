package ownership

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/matsumo_and/cogi/internal/db"
)

// Analyzer handles ownership analysis using git blame
type Analyzer struct {
	db *db.DB
}

// New creates a new ownership analyzer
func New(database *db.DB) *Analyzer {
	return &Analyzer{
		db: database,
	}
}

// BlameResult represents a single line's blame information
type BlameResult struct {
	Line         int
	CommitHash   string
	AuthorName   string
	AuthorEmail  string
	CommitDate   time.Time
	Content      string
}

// AnalyzeFile analyzes ownership for a single file
func (a *Analyzer) AnalyzeFile(ctx context.Context, repoPath, filePath string, fileID int64) error {
	// Run git blame
	blameResults, err := a.runGitBlame(ctx, repoPath, filePath)
	if err != nil {
		return fmt.Errorf("failed to run git blame: %w", err)
	}

	if len(blameResults) == 0 {
		return nil
	}

	// Group consecutive lines by the same author and commit
	ownershipRanges := a.groupByAuthor(blameResults)

	// Delete existing ownership data for this file
	if err := a.db.DeleteOwnershipByFile(fileID); err != nil {
		return fmt.Errorf("failed to delete old ownership data: %w", err)
	}

	// Insert new ownership data
	for _, ow := range ownershipRanges {
		ownership := &db.Ownership{
			FileID:          fileID,
			StartLine:       ow.StartLine,
			EndLine:         ow.EndLine,
			AuthorName:      ow.AuthorName,
			AuthorEmail:     ow.AuthorEmail,
			LastCommitHash:  ow.CommitHash,
			LastCommitDate:  ow.CommitDate,
			CommitCount:     ow.CommitCount,
		}

		if err := a.db.CreateOwnership(ownership); err != nil {
			return fmt.Errorf("failed to create ownership: %w", err)
		}
	}

	return nil
}

// OwnershipRange represents a range of lines owned by the same author
type OwnershipRange struct {
	StartLine   int
	EndLine     int
	AuthorName  string
	AuthorEmail string
	CommitHash  string
	CommitDate  time.Time
	CommitCount int
}

// groupByAuthor groups consecutive lines by the same author and commit
func (a *Analyzer) groupByAuthor(results []BlameResult) []OwnershipRange {
	if len(results) == 0 {
		return nil
	}

	var ranges []OwnershipRange
	current := OwnershipRange{
		StartLine:   results[0].Line,
		EndLine:     results[0].Line,
		AuthorName:  results[0].AuthorName,
		AuthorEmail: results[0].AuthorEmail,
		CommitHash:  results[0].CommitHash,
		CommitDate:  results[0].CommitDate,
		CommitCount: 1,
	}

	for i := 1; i < len(results); i++ {
		r := results[i]

		// If same author and commit, extend the range
		if r.AuthorName == current.AuthorName && r.CommitHash == current.CommitHash {
			current.EndLine = r.Line
			current.CommitCount++
		} else {
			// Different author or commit, save current range and start new one
			ranges = append(ranges, current)
			current = OwnershipRange{
				StartLine:   r.Line,
				EndLine:     r.Line,
				AuthorName:  r.AuthorName,
				AuthorEmail: r.AuthorEmail,
				CommitHash:  r.CommitHash,
				CommitDate:  r.CommitDate,
				CommitCount: 1,
			}
		}
	}

	// Don't forget the last range
	ranges = append(ranges, current)

	return ranges
}

// runGitBlame executes git blame and parses the output
func (a *Analyzer) runGitBlame(ctx context.Context, repoPath, filePath string) ([]BlameResult, error) {
	// git blame format: --porcelain for machine-readable output
	cmd := exec.CommandContext(ctx, "git", "blame", "--porcelain", filePath)
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		// File might not be tracked by git
		return nil, fmt.Errorf("git blame failed: %w", err)
	}

	return parseGitBlame(string(output))
}

// parseGitBlame parses the porcelain output of git blame
func parseGitBlame(output string) ([]BlameResult, error) {
	var results []BlameResult

	scanner := bufio.NewScanner(strings.NewReader(output))

	// Regex to match the first line of each blame block
	// Format: <commit-hash> <original-line> <final-line> <num-lines-in-group>
	commitLineRe := regexp.MustCompile(`^([0-9a-f]+)\s+(\d+)\s+(\d+)(?:\s+(\d+))?`)

	var currentCommit string
	var currentLine int
	var authorName string
	var authorEmail string
	var commitDate time.Time

	for scanner.Scan() {
		line := scanner.Text()

		// Parse commit line
		if matches := commitLineRe.FindStringSubmatch(line); matches != nil {
			currentCommit = matches[1]
			currentLine, _ = strconv.Atoi(matches[3]) // final line number
			continue
		}

		// Parse metadata lines
		if strings.HasPrefix(line, "author ") {
			authorName = strings.TrimPrefix(line, "author ")
		} else if strings.HasPrefix(line, "author-mail ") {
			email := strings.TrimPrefix(line, "author-mail ")
			// Remove <> from email
			authorEmail = strings.Trim(email, "<>")
		} else if strings.HasPrefix(line, "author-time ") {
			timestamp := strings.TrimPrefix(line, "author-time ")
			ts, _ := strconv.ParseInt(timestamp, 10, 64)
			commitDate = time.Unix(ts, 0)
		} else if strings.HasPrefix(line, "\t") {
			// This is the actual line content
			content := strings.TrimPrefix(line, "\t")

			results = append(results, BlameResult{
				Line:        currentLine,
				CommitHash:  currentCommit,
				AuthorName:  authorName,
				AuthorEmail: authorEmail,
				CommitDate:  commitDate,
				Content:     content,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse git blame output: %w", err)
	}

	return results, nil
}

// GetFileOwnership retrieves ownership information for a file
func (a *Analyzer) GetFileOwnership(fileID int64) ([]*db.Ownership, error) {
	return a.db.GetOwnershipByFile(fileID)
}

// GetAuthorFiles retrieves all files modified by a specific author
func (a *Analyzer) GetAuthorFiles(authorName string) ([]*db.Ownership, error) {
	return a.db.GetOwnershipByAuthor(authorName)
}
