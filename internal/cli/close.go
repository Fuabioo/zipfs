package cli

import (
	"fmt"
	"os"

	"github.com/Fuabioo/zipfs/internal/core"
	"github.com/spf13/cobra"
)

var (
	closeFlagSync   bool
	closeFlagNoSync bool
)

var closeCmd = &cobra.Command{
	Use:   "close [<session>]",
	Short: "Close a session and remove its workspace",
	Long: `Closes a session and removes its workspace directory.

If the workspace has unsaved changes and neither --sync nor --no-sync is specified:
- On TTY: prompts for confirmation
- Non-TTY: returns an error

The session argument is optional. If not provided, auto-resolves to the only
open session (fails if zero or multiple sessions are open).`,
	Args: cobra.MaximumNArgs(1),
	RunE: runClose,
}

func init() {
	closeCmd.Flags().BoolVar(&closeFlagSync, "sync", false, "Sync changes before closing")
	closeCmd.Flags().BoolVar(&closeFlagNoSync, "no-sync", false, "Close without syncing (discard changes)")
	closeCmd.MarkFlagsMutuallyExclusive("sync", "no-sync")
}

func runClose(cmd *cobra.Command, args []string) error {
	// Resolve session
	var identifier string
	if len(args) > 0 {
		identifier = args[0]
	}

	session, err := core.ResolveSession(identifier)
	if err != nil {
		return err
	}

	// Load config for sync operation if needed
	var cfg *core.Config
	if closeFlagSync {
		cfg, err = loadConfig()
		if err != nil {
			return err
		}
	}

	// Check if there are unsaved changes
	hasChanges := false
	if !closeFlagSync && !closeFlagNoSync {
		status, err := core.Status(session)
		if err != nil {
			return fmt.Errorf("failed to check status: %w", err)
		}

		hasChanges = len(status.Modified) > 0 || len(status.Added) > 0 || len(status.Deleted) > 0

		if hasChanges {
			// Prompt for confirmation on TTY, error on non-TTY
			if !isTerminal(os.Stdin) {
				return fmt.Errorf("session has unsaved changes; use --sync or --no-sync")
			}

			message := fmt.Sprintf("Session %q has unsaved changes. Close without syncing?", session.Name)
			if session.Name == "" {
				message = fmt.Sprintf("Session %s has unsaved changes. Close without syncing?", session.ID)
			}

			if !confirmPrompt(message) {
				return fmt.Errorf("close cancelled")
			}
		}
	}

	// Sync if requested
	synced := false
	if closeFlagSync {
		_, err := core.Sync(session, false, cfg)
		if err != nil {
			return fmt.Errorf("sync failed: %w", err)
		}
		synced = true
	}

	// Delete the session
	if err := core.DeleteSession(session.ID); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	// Output results
	if flagJSON {
		output := map[string]interface{}{
			"closed": true,
			"synced": synced,
		}
		return outputJSON(output)
	}

	// Human-readable output
	if !flagQuiet {
		sessionRef := session.Name
		if sessionRef == "" {
			sessionRef = session.ID
		}
		fmt.Printf("Session closed: %s\n", sessionRef)
		if synced {
			fmt.Println("Changes synced before closing")
		}
	}

	return nil
}
