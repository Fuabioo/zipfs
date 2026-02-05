package cli

import (
	"fmt"
	"os"

	"github.com/Fuabioo/zipfs/internal/core"
	"github.com/spf13/cobra"
)

var (
	deleteFlagRecursive bool
)

var deleteCmd = &cobra.Command{
	Use:   "delete [<session>] <path>",
	Short: "Delete a file or directory from workspace",
	Long: `Deletes a file or directory from the workspace.

For directories, use --recursive flag.
The session argument is optional and will auto-resolve if only one session is open.`,
	Args: cobra.MinimumNArgs(1),
	RunE: runDelete,
}

func init() {
	deleteCmd.Flags().BoolVarP(&deleteFlagRecursive, "recursive", "r", false, "Delete directories recursively")
}

func runDelete(cmd *cobra.Command, args []string) error {
	var sessionID, relativePath string

	if len(args) == 1 {
		// One arg - could be session or path
		// Try to resolve as session first
		session, err := core.GetSession(args[0])
		if err == nil && session != nil {
			return fmt.Errorf("path required")
		}
		// Not a session, treat as path with auto-resolved session
		relativePath = args[0]
	} else {
		// Two args - session and path
		sessionID = args[0]
		relativePath = args[1]
	}

	if relativePath == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Resolve session
	session, err := core.ResolveSession(sessionID)
	if err != nil {
		return err
	}

	// Get contents directory
	dirName := session.Name
	if dirName == "" {
		dirName = session.ID
	}
	contentsDir, err := core.ContentsDir(dirName)
	if err != nil {
		return err
	}

	// Delete file/directory
	if err := core.DeleteFile(contentsDir, relativePath, deleteFlagRecursive); err != nil {
		return err
	}

	// Output
	if !flagQuiet {
		fmt.Fprintf(os.Stderr, "Deleted: %s\n", relativePath)
	}

	return nil
}
