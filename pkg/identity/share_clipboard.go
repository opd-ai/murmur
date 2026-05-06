package identity

import (
	"fmt"
	"os/exec"
)

// writeToClipboard pipes content to a clipboard command, handling all I/O steps.
// The cmdName is typically "pbcopy" (macOS) or "clip" (Windows).
func writeToClipboard(content, cmdName string, cmdArgs ...string) error {
	cmd := exec.Command(cmdName, cmdArgs...)
	pipe, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("creating pipe to %s: %w", cmdName, err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting %s: %w", cmdName, err)
	}

	if _, err := pipe.Write([]byte(content)); err != nil {
		return fmt.Errorf("writing to %s: %w", cmdName, err)
	}

	if err := pipe.Close(); err != nil {
		return fmt.Errorf("closing pipe to %s: %w", cmdName, err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("waiting for %s: %w", cmdName, err)
	}

	return nil
}
