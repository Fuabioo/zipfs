package cli

import (
	"encoding/base64"
	"fmt"
	"os"
	"unicode/utf8"

	"github.com/Fuabioo/zipfs/internal/core"
	"github.com/spf13/cobra"
)

var readCmd = &cobra.Command{
	Use:   "read <session>:<path> | read [<session>] <path>",
	Short: "Read a file from workspace",
	Long: `Reads a file from the workspace and outputs to stdout.

Supports both colon syntax (session:path) and positional arguments.
Binary files are base64 encoded with a warning to stderr.`,
	Args: cobra.MinimumNArgs(1),
	RunE: runRead,
}

func runRead(cmd *cobra.Command, args []string) error {
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
	dirName := session.DirName()
	contentsDir, err := core.ContentsDir(dirName)
	if err != nil {
		return err
	}

	// Read file
	data, err := core.ReadFile(contentsDir, relativePath)
	if err != nil {
		return err
	}

	// Check if data is valid UTF-8
	if !utf8.Valid(data) {
		// Binary file - base64 encode
		fmt.Fprintln(os.Stderr, "Warning: binary file detected, outputting base64 encoding")
		encoded := base64.StdEncoding.EncodeToString(data)
		fmt.Println(encoded)
	} else {
		// Text file - output as-is
		os.Stdout.Write(data)
	}

	return nil
}
