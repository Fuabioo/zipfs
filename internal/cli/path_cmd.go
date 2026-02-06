package cli

import (
	"fmt"
	"os"

	"github.com/Fuabioo/zipfs/internal/core"
	"github.com/spf13/cobra"
)

var pathCmd = &cobra.Command{
	Use:   "path [<session>]",
	Short: "Output workspace contents path",
	Long: `Outputs the absolute path to the workspace contents directory.

Designed for command substitution (e.g., xlq --basepath $(zipfs path)).
No trailing newline when output is piped (not a TTY).`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPath,
}

func runPath(cmd *cobra.Command, args []string) error {
	// Resolve session
	var sessionID string
	if len(args) > 0 {
		sessionID = args[0]
	}

	session, err := core.ResolveSession(sessionID)
	if err != nil {
		return err
	}

	// Get contents directory
	dirName := session.DirName()
	contentsDir, err := core.ContentsDir(dirName)
	if err != nil {
		return err
	}

	// Output path
	// No trailing newline when not a TTY (for command substitution)
	if isTerminal(os.Stdout) {
		fmt.Println(contentsDir)
	} else {
		fmt.Print(contentsDir)
	}

	return nil
}
