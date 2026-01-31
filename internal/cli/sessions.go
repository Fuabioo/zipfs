package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/Fuabioo/zipfs/internal/core"
	"github.com/spf13/cobra"
)

var sessionsCmd = &cobra.Command{
	Use:   "sessions",
	Short: "List all open sessions",
	Long: `Lists all currently open zipfs sessions.

Outputs a table by default, or JSON with the --json flag.`,
	Args: cobra.NoArgs,
	RunE: runSessions,
}

func runSessions(cmd *cobra.Command, args []string) error {
	sessions, err := core.ListSessions()
	if err != nil {
		return err
	}

	if flagJSON {
		// Build JSON output matching MCP format
		output := make([]map[string]interface{}, 0, len(sessions))
		for _, s := range sessions {
			dirName := s.Name
			if dirName == "" {
				dirName = s.ID
			}
			workspacePath, err := core.ContentsDir(dirName)
			if err != nil {
				continue
			}

			sessionData := map[string]interface{}{
				"id":                   s.ID,
				"name":                 s.Name,
				"source_path":          s.SourcePath,
				"state":                s.State,
				"created_at":           s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				"last_accessed_at":     s.LastAccessedAt.Format("2006-01-02T15:04:05Z07:00"),
				"file_count":           s.FileCount,
				"extracted_size_bytes": s.ExtractedSizeBytes,
				"workspace_path":       workspacePath,
			}

			if s.LastSyncedAt != nil {
				sessionData["last_synced_at"] = s.LastSyncedAt.Format("2006-01-02T15:04:05Z07:00")
			}

			output = append(output, sessionData)
		}
		return outputJSON(output)
	}

	// Human-readable table output
	if len(sessions) == 0 {
		if !flagQuiet {
			fmt.Println("No open sessions")
		}
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tSOURCE\tFILES\tSIZE")

	for _, s := range sessions {
		name := s.Name
		if name == "" {
			name = "-"
		}

		// Shorten ID for display
		shortID := s.ID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}

		// Format size
		sizeStr := formatBytes(s.ExtractedSizeBytes)

		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n",
			shortID, name, s.SourcePath, s.FileCount, sizeStr)
	}

	w.Flush()
	return nil
}

// formatBytes formats bytes into human-readable format
func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
