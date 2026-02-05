package cli

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/Fuabioo/zipfs/internal/core"
	"github.com/spf13/cobra"
)

var (
	lsFlagLong      bool
	lsFlagRecursive bool
)

var lsCmd = &cobra.Command{
	Use:   "ls [<session>] [<path>]",
	Short: "List files in workspace",
	Long: `Lists files and directories in the workspace.

The session argument is optional and will auto-resolve if only one session is open.
The path argument is optional and defaults to the root of the workspace.`,
	Args: cobra.MaximumNArgs(2),
	RunE: runLs,
}

func init() {
	lsCmd.Flags().BoolVarP(&lsFlagLong, "long", "l", false, "Long format with size and timestamp")
	lsCmd.Flags().BoolVarP(&lsFlagRecursive, "recursive", "r", false, "List subdirectories recursively")
}

func runLs(cmd *cobra.Command, args []string) error {
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

	// List files
	entries, err := core.ListFiles(contentsDir, relativePath, lsFlagRecursive)
	if err != nil {
		return err
	}

	// Output
	if flagJSON {
		output := map[string]interface{}{
			"entries": entries,
		}
		return outputJSON(output)
	}

	// Human-readable output
	if len(entries) == 0 {
		if !flagQuiet {
			fmt.Println("(empty)")
		}
		return nil
	}

	if lsFlagLong {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for _, entry := range entries {
			modTime := time.Unix(entry.ModifiedAt, 0).Format("2006-01-02 15:04:05")
			sizeStr := formatBytes(entry.SizeBytes)
			typeStr := "FILE"
			if entry.Type == "dir" {
				typeStr = "DIR"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", typeStr, sizeStr, modTime, entry.Name)
		}
		w.Flush()
	} else {
		// Simple list
		for _, entry := range entries {
			name := entry.Name
			if entry.Type == "dir" {
				name += "/"
			}
			fmt.Println(name)
		}
	}

	return nil
}
