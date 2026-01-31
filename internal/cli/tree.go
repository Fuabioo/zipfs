package cli

import (
	"fmt"

	"github.com/Fuabioo/zipfs/internal/core"
	"github.com/spf13/cobra"
)

var (
	treeFlagMaxDepth int
)

var treeCmd = &cobra.Command{
	Use:   "tree [<session>] [<path>]",
	Short: "Display directory tree",
	Long: `Displays a tree view of the workspace directory structure.

The session argument is optional and will auto-resolve if only one session is open.
The path argument is optional and defaults to the root of the workspace.`,
	Args: cobra.MaximumNArgs(2),
	RunE: runTree,
}

func init() {
	treeCmd.Flags().IntVar(&treeFlagMaxDepth, "max-depth", 0, "Maximum depth to traverse (0 = unlimited)")
}

func runTree(cmd *cobra.Command, args []string) error {
	// Parse arguments: session is optional, path is optional
	var sessionID, relativePath string

	if len(args) == 0 {
		// No args - auto-resolve session, list root
		relativePath = ""
	} else if len(args) == 1 {
		// One arg - could be session or path
		// Try to resolve as session first
		session, err := core.GetSession(args[0])
		if err == nil && session != nil {
			sessionID = args[0]
			relativePath = ""
		} else {
			// Not a session, treat as path with auto-resolved session
			relativePath = args[0]
		}
	} else {
		// Two args - session and path
		sessionID = args[0]
		relativePath = args[1]
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

	// Normalize path
	if relativePath == "" || relativePath == "." {
		relativePath = ""
	}

	// Build tree
	treeStr, fileCount, dirCount, err := core.TreeView(contentsDir, relativePath, treeFlagMaxDepth)
	if err != nil {
		return err
	}

	// Output
	if flagJSON {
		output := map[string]interface{}{
			"tree":       treeStr,
			"file_count": fileCount,
			"dir_count":  dirCount,
		}
		return outputJSON(output)
	}

	// Human-readable output
	// Print root indicator
	if relativePath == "" {
		fmt.Println(".")
	} else {
		fmt.Println(relativePath)
	}

	// Print tree
	fmt.Print(treeStr)

	// Print summary
	if !flagQuiet {
		fmt.Printf("\n%d directories, %d files\n", dirCount, fileCount)
	}

	return nil
}
