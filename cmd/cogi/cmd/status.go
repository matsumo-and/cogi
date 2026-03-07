package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Cogi status and statistics",
	Long: `Display information about indexed repositories, database status,
and overall system health.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Cogi Status")
		fmt.Println("===========")
		// TODO: Implement status display
		fmt.Println("Indexed repositories: 0")
		fmt.Println("Total symbols: 0")
		fmt.Println("Database size: 0 MB")
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
