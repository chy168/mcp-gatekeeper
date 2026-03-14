package proxy

import (
	"fmt"
	"os"
	"strings"

	"github.com/chy168/mcp-gatekeeper/internal/secret"
)

// applyFileInjections processes --file=VAR=value and --file=/path=value entries.
// For VAR= entries: writes value to a temp file (0600), returns VAR=path in envAdditions.
// For /path= entries: writes value to the fixed path (0600), no env addition.
// tmpFiles contains paths of temp files that must be deleted after subprocess exits.
// resolved may be nil if no secret refs were present.
func applyFileInjections(injections []string, resolved map[string]string) (envAdditions []string, tmpFiles []string, err error) {
	for _, inj := range injections {
		idx := strings.Index(inj, "=")
		if idx < 0 {
			return nil, tmpFiles, fmt.Errorf("invalid --file injection (missing '='): %q", inj)
		}
		lhs := inj[:idx]
		rhs := inj[idx+1:]

		if resolved != nil {
			rhs = secret.Substitute(rhs, resolved)
		}

		if strings.HasPrefix(lhs, "/") || strings.HasPrefix(lhs, "~") {
			if err := os.WriteFile(lhs, []byte(rhs), 0600); err != nil {
				return nil, tmpFiles, fmt.Errorf("failed to write secret to %q: %w", lhs, err)
			}
		} else {
			f, err := os.CreateTemp("", "mcp-secret-*")
			if err != nil {
				return nil, tmpFiles, fmt.Errorf("failed to create temp file: %w", err)
			}
			tmpPath := f.Name()
			tmpFiles = append(tmpFiles, tmpPath)
			if err := f.Chmod(0600); err != nil {
				f.Close()
				return nil, tmpFiles, fmt.Errorf("failed to chmod temp file: %w", err)
			}
			if _, err := f.WriteString(rhs); err != nil {
				f.Close()
				return nil, tmpFiles, fmt.Errorf("failed to write to temp file: %w", err)
			}
			f.Close()
			envAdditions = append(envAdditions, lhs+"="+tmpPath)
		}
	}
	return envAdditions, tmpFiles, nil
}
