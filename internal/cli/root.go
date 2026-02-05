package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Version is set via ldflags during build
	Version = "dev"
	// Commit is set via ldflags during build
	Commit = "unknown"

	// Global flags
	flagJSON  bool
	flagQuiet bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "zipfs",
	Short: "Ephemeral workspace manager for zip files",
	Long: `zipfs extracts zip files into temporary workspaces, allows modifications,
and syncs changes back to the source zip file.

It provides both CLI and MCP server interfaces for human and AI agent use.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	err := rootCmd.Execute()
	if err != nil {
		printError(err)
		os.Exit(getExitCode(err))
	}
	return nil
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "Output in JSON format")
	rootCmd.PersistentFlags().BoolVarP(&flagQuiet, "quiet", "q", false, "Suppress non-essential output")

	// Add all subcommands
	rootCmd.AddCommand(openCmd)
	rootCmd.AddCommand(closeCmd)
	rootCmd.AddCommand(sessionsCmd)
	rootCmd.AddCommand(pruneCmd)
	rootCmd.AddCommand(lsCmd)
	rootCmd.AddCommand(treeCmd)
	rootCmd.AddCommand(readCmd)
	rootCmd.AddCommand(writeCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(grepCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(pathCmd)
	rootCmd.AddCommand(mcpCmd)
	rootCmd.AddCommand(versionCmd)
}

// GetVersion returns the version string
func GetVersion() string {
	if Commit != "unknown" {
		return fmt.Sprintf("%s (%s)", Version, Commit[:7])
	}
	return Version
}
