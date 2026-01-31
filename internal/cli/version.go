package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  `Displays the version and commit hash of zipfs.`,
	Args:  cobra.NoArgs,
	RunE:  runVersion,
}

func runVersion(cmd *cobra.Command, args []string) error {
	if flagJSON {
		output := map[string]interface{}{
			"version": Version,
			"commit":  Commit,
		}
		return outputJSON(output)
	}

	fmt.Printf("zipfs version %s\n", GetVersion())
	return nil
}
