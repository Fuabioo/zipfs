package cli

import (
	"fmt"

	"github.com/Fuabioo/zipfs/internal/core"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status [<session>]",
	Short: "Show workspace status",
	Long: `Shows what changed in the workspace since extraction.

Output is similar to git status, showing modified, added, and deleted files.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	// Resolve session
	var sessionID string
	if len(args) > 0 {
		sessionID = args[0]
	}

	session, err := core.ResolveSession(sessionID)
	if err != nil {
		return err
	}

	// Get status
	status, err := core.Status(session)
	if err != nil {
		return err
	}

	// Output
	if flagJSON {
		return outputJSON(status)
	}

	// Human-readable output (git-like)
	sessionRef := session.Name
	if sessionRef == "" {
		sessionRef = session.ID[:8]
	}

	fmt.Printf("On session: %s\n", sessionRef)
	fmt.Printf("Source: %s\n\n", session.SourcePath)

	totalChanges := len(status.Modified) + len(status.Added) + len(status.Deleted)

	if totalChanges == 0 {
		fmt.Println("No changes since extraction")
		fmt.Printf("(%d files unchanged)\n", status.UnchangedCount)
		return nil
	}

	if len(status.Modified) > 0 {
		fmt.Printf("Modified files (%d):\n", len(status.Modified))
		for _, path := range status.Modified {
			fmt.Printf("  M %s\n", path)
		}
		fmt.Println()
	}

	if len(status.Added) > 0 {
		fmt.Printf("Added files (%d):\n", len(status.Added))
		for _, path := range status.Added {
			fmt.Printf("  A %s\n", path)
		}
		fmt.Println()
	}

	if len(status.Deleted) > 0 {
		fmt.Printf("Deleted files (%d):\n", len(status.Deleted))
		for _, path := range status.Deleted {
			fmt.Printf("  D %s\n", path)
		}
		fmt.Println()
	}

	fmt.Printf("%d file(s) changed, %d unchanged\n", totalChanges, status.UnchangedCount)

	return nil
}
