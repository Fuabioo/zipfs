package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/Fuabioo/zipfs/internal/core"
	"github.com/spf13/cobra"
)

var (
	pruneFlagAll    bool
	pruneFlagStale  string
	pruneFlagDryRun bool
)

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Remove stale or all workspaces",
	Long: `Removes workspace directories based on criteria.

Use --all to remove all sessions, or --stale with a duration (e.g., "24h", "7d")
to remove sessions that haven't been accessed within that time period.`,
	Args: cobra.NoArgs,
	RunE: runPrune,
}

func init() {
	pruneCmd.Flags().BoolVar(&pruneFlagAll, "all", false, "Remove all sessions")
	pruneCmd.Flags().StringVar(&pruneFlagStale, "stale", "", "Remove sessions older than duration (e.g., 24h, 7d)")
	pruneCmd.Flags().BoolVar(&pruneFlagDryRun, "dry-run", false, "Show what would be removed without removing")
}

func runPrune(cmd *cobra.Command, args []string) error {
	// Parse stale duration if provided
	var staleDuration time.Duration
	var err error
	if pruneFlagStale != "" {
		staleDuration, err = parseDuration(pruneFlagStale)
		if err != nil {
			return fmt.Errorf("invalid duration format: %w", err)
		}
	}

	// Validate flags
	if !pruneFlagAll && pruneFlagStale == "" {
		return fmt.Errorf("must specify either --all or --stale")
	}

	// Get all sessions
	sessions, err := core.ListSessions()
	if err != nil {
		return err
	}

	// Filter sessions to prune
	var toPrune []*core.Session
	now := time.Now()

	for _, s := range sessions {
		shouldPrune := false

		if pruneFlagAll {
			shouldPrune = true
		} else if pruneFlagStale != "" {
			age := now.Sub(s.LastAccessedAt)
			if age > staleDuration {
				shouldPrune = true
			}
		}

		if shouldPrune {
			toPrune = append(toPrune, s)
		}
	}

	// Calculate total size freed
	var totalFreed uint64
	for _, s := range toPrune {
		totalFreed += s.ExtractedSizeBytes
	}

	// Build result for JSON output
	type PruneEntry struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Reason string `json:"reason"`
	}

	pruned := make([]PruneEntry, 0, len(toPrune))

	// Perform pruning
	for _, s := range toPrune {
		reason := "all sessions"
		if pruneFlagStale != "" {
			age := now.Sub(s.LastAccessedAt)
			reason = fmt.Sprintf("stale (%s)", formatDuration(age))
		}

		pruned = append(pruned, PruneEntry{
			ID:     s.ID,
			Name:   s.Name,
			Reason: reason,
		})

		if !pruneFlagDryRun {
			if err := core.DeleteSession(s.ID); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to delete session %s: %v\n", s.ID, err)
				continue
			}
		}
	}

	// Output results
	if flagJSON {
		output := map[string]interface{}{
			"pruned":      pruned,
			"freed_bytes": totalFreed,
		}
		return outputJSON(output)
	}

	// Human-readable output
	if len(pruned) == 0 {
		if !flagQuiet {
			fmt.Println("No sessions to prune")
		}
		return nil
	}

	if pruneFlagDryRun {
		fmt.Printf("Would prune %d session(s):\n", len(pruned))
	} else {
		fmt.Printf("Pruned %d session(s):\n", len(pruned))
	}

	for _, p := range pruned {
		name := p.Name
		if name == "" {
			name = p.ID[:8]
		}
		fmt.Printf("  - %s (%s)\n", name, p.Reason)
	}

	fmt.Printf("Total space freed: %s\n", formatBytes(totalFreed))

	return nil
}

// parseDuration parses duration strings like "24h", "7d", "30d"
func parseDuration(s string) (time.Duration, error) {
	// Try standard duration format first
	d, err := time.ParseDuration(s)
	if err == nil {
		return d, nil
	}

	// Try days format (e.g., "7d")
	if len(s) >= 2 && s[len(s)-1] == 'd' {
		days := s[:len(s)-1]
		var count int
		_, err := fmt.Sscanf(days, "%d", &count)
		if err != nil {
			return 0, err
		}
		return time.Duration(count) * 24 * time.Hour, nil
	}

	return 0, fmt.Errorf("invalid duration format (use 24h, 7d, etc.)")
}

// formatDuration formats a duration into a human-readable string
func formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	if days > 0 {
		return fmt.Sprintf("%dd", days)
	}
	hours := int(d.Hours())
	if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	}
	minutes := int(d.Minutes())
	return fmt.Sprintf("%dm", minutes)
}
