package proxy

import (
	"bufio"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/chy168/mcp-gatekeeper/internal/filter"
	"github.com/chy168/mcp-gatekeeper/internal/secret"
)

// Run starts the subprocess and proxies stdio between client and subprocess.
// allows is the allowlist, excludes is the denylist.
// secretSource, secretSourceName, envInjections, and fileInjections enable secret resolution and injection.
func Run(command string, args, allows, excludes, envInjections, fileInjections []string, secretSource, secretSourceName string) int {
	// Collect all {$secret.*} refs from args, envInjections, and fileInjections
	allStrings := append(append(append([]string{}, args...), envInjections...), fileInjections...)
	allRefs := secret.ExtractRefsFromSlice(allStrings)

	// Create backend once so it can be reused for write-back after subprocess exits.
	var backend secret.Backend
	var resolved map[string]string
	if len(allRefs) > 0 {
		if secretSource == "" {
			fmt.Fprintf(os.Stderr, "mcp-gatekeeper: secret references found but --secret-source is not set\n")
			return 1
		}
		var err error
		backend, err = secret.NewBackend(secretSource)
		if err != nil {
			fmt.Fprintf(os.Stderr, "mcp-gatekeeper: failed to create secret backend: %v\n", err)
			return 1
		}
		resolved, err = secret.ResolveAllWithBackend(backend, secretSourceName, allRefs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "mcp-gatekeeper: failed to resolve secrets: %v\n", err)
			return 1
		}
	}

	// Substitute refs in args
	if resolved != nil {
		args = secret.SubstituteSlice(args, resolved)
	}

	// Temp file tracking for cleanup
	tmpFiles := []string{}

	// Signal handler for cleanup
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigCh
		for _, f := range tmpFiles {
			os.Remove(f)
		}
		os.Exit(1)
	}()
	defer func() {
		for _, f := range tmpFiles {
			os.Remove(f)
		}
	}()

	// Build extra env vars from envInjections
	var resolvedEnvs []string
	for _, inj := range envInjections {
		substituted := inj
		if resolved != nil {
			substituted = secret.Substitute(inj, resolved)
		}
		resolvedEnvs = append(resolvedEnvs, substituted)
	}

	// Handle fileInjections
	fileEnvs, fileTmps, writebacks, err := applyFileInjections(fileInjections, resolved)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mcp-gatekeeper: %v\n", err)
		return 1
	}
	tmpFiles = append(tmpFiles, fileTmps...)
	resolvedEnvs = append(resolvedEnvs, fileEnvs...)

	cmd := exec.Command(command, args...)
	cmd.Stderr = os.Stderr

	if len(resolvedEnvs) > 0 {
		cmd.Env = append(os.Environ(), resolvedEnvs...)
	}

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "mcp-gatekeeper: failed to create stdin pipe: %v\n", err)
		return 1
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "mcp-gatekeeper: failed to create stdout pipe: %v\n", err)
		return 1
	}

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "mcp-gatekeeper: failed to start subprocess: %v\n", err)
		return 1
	}

	done := make(chan struct{})

	// client → server: transparent copy
	go func() {
		io.Copy(stdinPipe, os.Stdin)
		stdinPipe.Close()
	}()

	// server → client: line-by-line, filter tools/list responses
	go func() {
		defer close(done)
		scanner := bufio.NewScanner(stdoutPipe)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Bytes()
			var out []byte
			if filter.IsToolsListResponse(line) {
				out, _ = filter.FilterToolsListResponse(line, allows, excludes)
			} else {
				out = line
			}
			os.Stdout.Write(out)
			os.Stdout.Write([]byte("\n"))
		}
	}()

	<-done
	cmd.Wait()

	// Write-back: for files marked with :w, sync modified content back to the bundle.
	// Must happen before deferred temp file cleanup so files are still readable.
	if backend != nil && len(writebacks) > 0 {
		ctx := context.Background()
		for _, wb := range writebacks {
			current, err := os.ReadFile(wb.Path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "mcp-gatekeeper: write-back: failed to read %q: %v\n", wb.Path, err)
				continue
			}
			if sha256.Sum256(current) == wb.OrigHash {
				continue // unchanged, skip
			}
			if err := secret.SetBundleKey(ctx, backend, secretSourceName, wb.SecretKey, string(current)); err != nil {
				fmt.Fprintf(os.Stderr, "mcp-gatekeeper: write-back: failed to update %q: %v\n", wb.SecretKey, err)
			} else {
				fmt.Fprintf(os.Stderr, "mcp-gatekeeper: write-back: updated %q in bundle %q\n", wb.SecretKey, secretSourceName)
			}
		}
	}

	return 0
}
