package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Fuabioo/zipfs/internal/core"
	"github.com/Fuabioo/zipfs/internal/errors"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// resolveSession resolves a session identifier from command arguments or auto-resolves.
// It handles the session resolution logic per ADR-003.
func resolveSession(cmd *cobra.Command, args []string, argIndex int) (*core.Session, string, error) {
	var identifier string

	// Check if we have an argument at the specified index
	if argIndex < len(args) {
		identifier = args[argIndex]
	}

	// Try to resolve the session
	session, err := core.ResolveSession(identifier)
	if err != nil {
		return nil, "", err
	}

	return session, identifier, nil
}

// parseColonSyntax parses "session:path" syntax into session and path components.
// Returns empty session string if no colon is found.
func parseColonSyntax(arg string) (session, path string) {
	parts := strings.SplitN(arg, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", arg
}

// outputJSON marshals and prints JSON to stdout.
func outputJSON(v interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(v); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}
	return nil
}

// isTerminal checks if the given file descriptor is a TTY.
func isTerminal(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}

// getExitCode maps error codes to CLI exit codes per ADR-006.
func getExitCode(err error) int {
	if err == nil {
		return 0
	}

	code := errors.Code(err)
	switch code {
	case errors.CodeSessionNotFound, errors.CodeAmbiguousSession, errors.CodeNoSessions:
		return 4 // Session not found
	case errors.CodeZipBombDetected:
		return 5 // Zip bomb / security
	case errors.CodeConflictDetected:
		return 3 // Conflict detected
	case "":
		// Not a zipfs error - could be usage error
		return 1 // General error
	default:
		return 1 // General error
	}
}

// loadConfig loads the configuration from the data directory.
func loadConfig() (*core.Config, error) {
	dataDir, err := core.DataDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get data directory: %w", err)
	}

	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	cfg, err := core.LoadConfig(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return cfg, nil
}

// printError prints an error to stderr with appropriate formatting.
func printError(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
}

// confirmPrompt prompts the user for a yes/no confirmation.
// Returns true if user confirms, false otherwise.
func confirmPrompt(message string) bool {
	if !isTerminal(os.Stdin) {
		return false
	}

	fmt.Fprintf(os.Stderr, "%s (y/N): ", message)

	var response string
	if _, err := fmt.Scanln(&response); err != nil {
		return false
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}
