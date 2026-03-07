package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var addName string

var addCmd = &cobra.Command{
	Use:   "add <repo-path>",
	Short: "Add a repository to the index",
	Long: `Add a repository to Cogi's index.

The repository will be scanned and indexed for code intelligence features.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repoPath := args[0]
		name := addName

		if name == "" {
			// TODO: Use basename of repo path as default name
			name = "repository"
		}

		fmt.Printf("Adding repository: %s (name: %s)\n", repoPath, name)
		// TODO: Implement repository addition logic
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().StringVarP(&addName, "name", "n", "", "Custom name for the repository")
}
