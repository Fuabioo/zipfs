package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/Fuabioo/zipfs/internal/core"
	"github.com/spf13/cobra"
)

var (
	writeFlagStdin   bool
	writeFlagContent string
)

var writeCmd = &cobra.Command{
	Use:   "write <session>:<path> | write [<session>] <path>",
	Short: "Write a file to workspace",
	Long: `Writes content to a file in the workspace.

Supports both colon syntax (session:path) and positional arguments.
Reads from stdin by default when piped, or use --content for inline strings.`,
	Args: cobra.MinimumNArgs(1),
	RunE: runWrite,
}

func init() {
	writeCmd.Flags().BoolVar(&writeFlagStdin, "stdin", false, "Read content from stdin (default when piped)")
	writeCmd.Flags().StringVar(&writeFlagContent, "content", "", "Content to write (inline string)")
}

func runWrite(cmd *cobra.Command, args []string) error {
	var sessionID, relativePath string

	// Parse arguments - support colon syntax
	if len(args) == 1 {
		// Try colon syntax first
		s, p := parseColonSyntax(args[0])
		if s != "" {
			sessionID = s
			relativePath = p
		} else {
			// No colon - path only, auto-resolve session
			relativePath = args[0]
		}
	} else {
		// Two args: session path
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

	// Determine content source
	var content []byte

	if writeFlagContent != "" {
		// Content from flag
		content = []byte(writeFlagContent)
	} else if writeFlagStdin || !isTerminal(os.Stdin) {
		// Content from stdin (explicit flag or piped input)
		content, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read from stdin: %w", err)
		}
	} else {
		return fmt.Errorf("no content provided; use --content or pipe data to stdin")
	}

	// Write file
	if err := core.WriteFile(contentsDir, relativePath, content, true); err != nil {
		return err
	}

	// Output
	if !flagQuiet {
		fmt.Fprintf(os.Stderr, "Wrote %d bytes to %s\n", len(content), relativePath)
	}

	return nil
}
