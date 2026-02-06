package cli

import (
	"fmt"
	"os"

	"github.com/Fuabioo/zipfs/internal/core"
	"github.com/spf13/cobra"
)

var (
	grepFlagGlob       string
	grepFlagIgnoreCase bool
	grepFlagLineNumber bool
	grepFlagMaxResults int
)

var grepCmd = &cobra.Command{
	Use:   "grep <pattern> [<session>] [<path>]",
	Short: "Search file contents in workspace",
	Long: `Searches for a pattern in files within the workspace.

The pattern is a regular expression. Session and path are optional.
Output format matches standard grep: file:line:content`,
	Args: cobra.MinimumNArgs(1),
	RunE: runGrep,
}

func init() {
	grepCmd.Flags().StringVar(&grepFlagGlob, "glob", "", "File glob filter (e.g., *.txt)")
	grepCmd.Flags().BoolVarP(&grepFlagIgnoreCase, "ignore-case", "i", false, "Case-insensitive search")
	grepCmd.Flags().BoolVarP(&grepFlagLineNumber, "line-number", "n", true, "Show line numbers (default true)")
	grepCmd.Flags().IntVar(&grepFlagMaxResults, "max-results", 100, "Maximum matches to return")
}

func runGrep(cmd *cobra.Command, args []string) error {
	// Parse arguments
	pattern := args[0]
	var sessionID, relativePath string

	if len(args) >= 2 {
		// Try to resolve second arg as session
		session, err := core.GetSession(args[1])
		if err == nil && session != nil {
			sessionID = args[1]
			if len(args) >= 3 {
				relativePath = args[2]
			}
		} else {
			// Not a session, treat as path
			relativePath = args[1]
		}
	}

	// Resolve session
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

	// Normalize path
	if relativePath == "" {
		relativePath = "."
	}

	// Perform grep
	matches, totalMatches, err := core.GrepFiles(contentsDir, relativePath, pattern, grepFlagGlob, grepFlagIgnoreCase, grepFlagMaxResults)
	if err != nil {
		return err
	}

	// Output
	if flagJSON {
		output := map[string]interface{}{
			"matches":       matches,
			"total_matches": totalMatches,
			"truncated":     totalMatches > len(matches),
		}
		return outputJSON(output)
	}

	// Human-readable output (grep format)
	for _, match := range matches {
		if grepFlagLineNumber {
			fmt.Printf("%s:%d:%s\n", match.File, match.LineNumber, match.LineContent)
		} else {
			fmt.Printf("%s:%s\n", match.File, match.LineContent)
		}
	}

	// Show truncation warning
	if totalMatches > len(matches) && !flagQuiet {
		fmt.Fprintf(os.Stderr, "Warning: output truncated to %d matches (total: %d)\n", len(matches), totalMatches)
	}

	return nil
}
