package proxy

import (
	"crypto/sha256"
	"fmt"
	"os"
	"strings"

	"github.com/chy168/mcp-gatekeeper/internal/secret"
)

// FileWriteback tracks a file injection whose content should be synced back to
// the secret bundle when the subprocess exits (triggered by the :w modifier).
type FileWriteback struct {
	Path      string
	SecretKey string
	OrigHash  [32]byte
}

// applyFileInjections processes --file=VAR=value and --file=/path=value entries.
// For VAR= entries: writes value to a temp file (0600), returns VAR=path in envAdditions.
// For /path= entries: writes value to the fixed path (0600), no env addition.
// tmpFiles contains paths of temp files that must be deleted after subprocess exits.
// writebacks contains files marked with :w that should be synced back on exit.
// resolved may be nil if no secret refs were present.
func applyFileInjections(injections []string, resolved map[string]string) (envAdditions []string, tmpFiles []string, writebacks []FileWriteback, err error) {
	for _, inj := range injections {
		idx := strings.Index(inj, "=")
		if idx < 0 {
			return nil, tmpFiles, writebacks, fmt.Errorf("invalid --file injection (missing '='): %q", inj)
		}
		lhs := inj[:idx]
		rhs := inj[idx+1:]

		// Determine write-back key before substitution (need the original ref).
		// Only track write-back when there is exactly one {$secret.name:w} ref.
		var writebackKey string
		refs := secret.ExtractRefsWithModifiers(rhs)
		if len(refs) == 1 && refs[0].Writeback {
			writebackKey = refs[0].Name
		}

		if resolved != nil {
			rhs = secret.Substitute(rhs, resolved)
		}

		origHash := sha256.Sum256([]byte(rhs))

		var filePath string
		if strings.HasPrefix(lhs, "/") || strings.HasPrefix(lhs, "~") {
			if err := os.WriteFile(lhs, []byte(rhs), 0600); err != nil {
				return nil, tmpFiles, writebacks, fmt.Errorf("failed to write secret to %q: %w", lhs, err)
			}
			filePath = lhs
		} else {
			f, err := os.CreateTemp("", "mcp-secret-*")
			if err != nil {
				return nil, tmpFiles, writebacks, fmt.Errorf("failed to create temp file: %w", err)
			}
			filePath = f.Name()
			tmpFiles = append(tmpFiles, filePath)
			if err := f.Chmod(0600); err != nil {
				f.Close()
				return nil, tmpFiles, writebacks, fmt.Errorf("failed to chmod temp file: %w", err)
			}
			if _, err := f.WriteString(rhs); err != nil {
				f.Close()
				return nil, tmpFiles, writebacks, fmt.Errorf("failed to write to temp file: %w", err)
			}
			f.Close()
			envAdditions = append(envAdditions, lhs+"="+filePath)
		}

		if writebackKey != "" {
			writebacks = append(writebacks, FileWriteback{
				Path:      filePath,
				SecretKey: writebackKey,
				OrigHash:  origHash,
			})
		}
	}
	return envAdditions, tmpFiles, writebacks, nil
}
