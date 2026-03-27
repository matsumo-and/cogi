package indexer

import (
	"fmt"
	"strings"
	"sync/atomic"
)

// ProgressBar displays a visual progress indicator for long-running operations
type ProgressBar struct {
	total     int64
	completed atomic.Int64
	label     string
	barWidth  int
}

// NewProgressBar creates a new progress bar
func NewProgressBar(total int64, label string) *ProgressBar {
	pb := &ProgressBar{
		total:    total,
		label:    label,
		barWidth: 40,
	}
	// Show initial 0% progress
	pb.display()
	return pb
}

// Increment increments the progress by 1
func (pb *ProgressBar) Increment() {
	current := pb.completed.Add(1)
	pb.display()
	_ = current // Use variable to avoid compiler warning
}

// Add increments the progress by n
func (pb *ProgressBar) Add(n int64) {
	current := pb.completed.Add(n)
	pb.display()
	_ = current // Use variable to avoid compiler warning
}

// display renders the progress bar
func (pb *ProgressBar) display() {
	current := pb.completed.Load()
	percent := float64(current) * 100.0 / float64(pb.total)
	filled := int(float64(pb.barWidth) * float64(current) / float64(pb.total))

	var bar string
	if filled > 0 {
		bar = strings.Repeat("█", filled-1) + strings.Repeat("░", pb.barWidth-filled)
	} else {
		bar = strings.Repeat("░", pb.barWidth)
	}

	if pb.label != "" {
		fmt.Printf("\r%s [%s] %d/%d (%.1f%%)", pb.label, bar, current, pb.total, percent)
	} else {
		fmt.Printf("\r[%s] %d/%d (%.1f%%)", bar, current, pb.total, percent)
	}
}

// Finish completes the progress bar and moves to a new line
func (pb *ProgressBar) Finish() {
	fmt.Println()
}
