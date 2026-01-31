package cli

import (
	"fmt"

	"github.com/Fuabioo/zipfs/internal/core"
	"github.com/spf13/cobra"
)

var (
	syncFlagForce  bool
	syncFlagDryRun bool
)

var syncCmd = &cobra.Command{
	Use:   "sync [<session>]",
	Short: "Sync workspace changes back to zip",
	Long: `Syncs workspace changes back to the source zip file.

Creates a backup of the original zip file before syncing.
Use --force to ignore external modification conflicts.
Use --dry-run to preview changes without syncing.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSync,
}

func init() {
	syncCmd.Flags().BoolVar(&syncFlagForce, "force", false, "Ignore external modification conflict")
	syncCmd.Flags().BoolVar(&syncFlagDryRun, "dry-run", false, "Preview changes without syncing")
}

func runSync(cmd *cobra.Command, args []string) error {
	// Resolve session
	var sessionID string
	if len(args) > 0 {
		sessionID = args[0]
	}

	session, err := core.ResolveSession(sessionID)
	if err != nil {
		return err
	}

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// Dry run: just show status
	if syncFlagDryRun {
		status, err := core.Status(session)
		if err != nil {
			return err
		}

		if flagJSON {
			output := map[string]interface{}{
				"dry_run":        true,
				"files_modified": len(status.Modified),
				"files_added":    len(status.Added),
				"files_deleted":  len(status.Deleted),
				"modified":       status.Modified,
				"added":          status.Added,
				"deleted":        status.Deleted,
			}
			return outputJSON(output)
		}

		fmt.Println("Dry run - changes to be synced:")
		if len(status.Modified) > 0 {
			fmt.Printf("\nModified (%d):\n", len(status.Modified))
			for _, path := range status.Modified {
				fmt.Printf("  M %s\n", path)
			}
		}
		if len(status.Added) > 0 {
			fmt.Printf("\nAdded (%d):\n", len(status.Added))
			for _, path := range status.Added {
				fmt.Printf("  A %s\n", path)
			}
		}
		if len(status.Deleted) > 0 {
			fmt.Printf("\nDeleted (%d):\n", len(status.Deleted))
			for _, path := range status.Deleted {
				fmt.Printf("  D %s\n", path)
			}
		}

		if len(status.Modified)+len(status.Added)+len(status.Deleted) == 0 {
			fmt.Println("No changes to sync")
		}

		return nil
	}

	// Perform sync
	result, err := core.Sync(session, syncFlagForce, cfg)
	if err != nil {
		return err
	}

	// Output
	if flagJSON {
		output := map[string]interface{}{
			"synced":             true,
			"backup_path":        result.BackupPath,
			"files_modified":     result.FilesModified,
			"files_added":        result.FilesAdded,
			"files_deleted":      result.FilesDeleted,
			"new_zip_size_bytes": result.NewZipSizeBytes,
		}
		return outputJSON(output)
	}

	// Human-readable output
	if !flagQuiet {
		fmt.Printf("Synced to: %s\n", session.SourcePath)
		fmt.Printf("Backup: %s\n", result.BackupPath)
		fmt.Printf("New size: %s\n", formatBytes(result.NewZipSizeBytes))
	}

	return nil
}
