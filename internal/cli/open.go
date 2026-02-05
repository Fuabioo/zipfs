package cli

import (
	"fmt"

	"github.com/Fuabioo/zipfs/internal/core"
	"github.com/spf13/cobra"
)

var (
	openFlagName    string
	openFlagMaxSize uint64
)

var openCmd = &cobra.Command{
	Use:   "open <path.zip>",
	Short: "Open a zip file and create a workspace session",
	Long: `Opens a zip file, extracts it to a workspace, and creates a session.

The session can be referenced by name (if provided) or by session ID.
All files are extracted to a temporary workspace that can be modified.`,
	Args: cobra.ExactArgs(1),
	RunE: runOpen,
}

func init() {
	openCmd.Flags().StringVar(&openFlagName, "name", "", "Human-readable session name")
	openCmd.Flags().Uint64Var(&openFlagMaxSize, "max-size", 0, "Override max extracted size (bytes)")
}

func runOpen(cmd *cobra.Command, args []string) error {
	zipPath := args[0]

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// Apply max-size override if provided
	if openFlagMaxSize > 0 {
		cfg.Security.MaxExtractedSizeBytes = openFlagMaxSize
	}

	// Create session
	session, err := core.CreateSession(zipPath, openFlagName, cfg)
	if err != nil {
		return err
	}

	// Get workspace path
	dirName := session.Name
	if dirName == "" {
		dirName = session.ID
	}
	workspacePath, err := core.ContentsDir(dirName)
	if err != nil {
		return err
	}

	// Output results
	if flagJSON {
		output := map[string]interface{}{
			"session_id":           session.ID,
			"name":                 session.Name,
			"workspace_path":       workspacePath,
			"file_count":           session.FileCount,
			"extracted_size_bytes": session.ExtractedSizeBytes,
		}
		return outputJSON(output)
	}

	// Human-readable output
	fmt.Printf("Session opened: %s\n", session.ID)
	if session.Name != "" {
		fmt.Printf("Name: %s\n", session.Name)
	}
	fmt.Printf("Workspace: %s\n", workspacePath)
	fmt.Printf("Files: %d\n", session.FileCount)
	fmt.Printf("Size: %d bytes\n", session.ExtractedSizeBytes)

	return nil
}
