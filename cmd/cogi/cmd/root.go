package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

const corgiArt = `
            ░░
            ░ ░             ░░░
            ░  ░         ░░░  ░░
            ░░░░░     ░░░░░   ░░           ███████████       █████████        ███████████    ███████
           ░░░░░░░░░░░░░░░    ░░          ███░░░░░░░███    ███░░░░░░░███     ███░░░░░░░███   ░░███ 
         ░░░██░░░░░░░░░░░░░░░░░░         ███       ░░░    ███       ░░███   ███       ░░░     ░███ 
        ░░░█░░░░░░░░░░░░░░░░░░░░        ░███             ░███        ░███  ░███               ░███
        ░██░░░  ░░░░░░░░░░░░░░░         ░███             ░███        ░███  ░███               ░███ 
    ░██████░░░░░░░░░░░░░░░░░░░░░        ░███             ░███        ░███  ░███               ░███ 
 ░░░░██████████████████░░░░░░░░░░       ░███             ░███        ░███  ░███               ░███
 ░  ░█████████████████████░░░░░░░       ░███             ░███        ░███  ░███               ░███  
  ░██████████░░░░░██████████░░░░░       ░███             ░███        ░███  ░███               ░███  
     ░░░░░░░░░░  ░██████████████░       ░███             ░███        ░███  ░███      █████    ░███ 
   ░░░░░░░░   ██████████████████░       ░░███       ███  ░░███       ███   ░░███    ░░███     ░███ 
     ░░░ ████████████████████████░       ░░███████████    ░░░█████████░     ░░███████████    ███████
      ████████████████████████████░       ░░░░░░░░░░░       ░░░░░░░░░        ░░░░░░░░░░░     ░░░░░
`

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cogi",
	Short: "Code Intelligence Engine",
	Long: `Cogi - Code Intelligence Engine

A local code intelligence engine that combines Tree-sitter, SQLite FTS5
to enable advanced code search and RAG capabilities.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Show corgi art when no subcommand is provided
		orange := color.New(color.FgHiYellow).Add(color.Bold)
		_, _ = orange.Print(corgiArt)
		_ = cmd.Help()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cogi/config.yaml)")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		// Search config in home directory with name ".cogi" (without extension).
		viper.AddConfigPath(home + "/.cogi")
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
