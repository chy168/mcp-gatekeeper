package proxy

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/chy168/mcp-gatekeeper/internal/filter"
	"github.com/chy168/mcp-gatekeeper/internal/secret"
)

// Run starts the subprocess and proxies stdio between client and subprocess.
// allows is the allowlist, excludes is the denylist.
// secretSource, envInjections, and fileInjections enable secret resolution and injection.
func Run(command string, args, allows, excludes, envInjections, fileInjections []string, secretSource string) int {
	// Collect all {$secret.*} refs from args, envInjections, and fileInjections
	allStrings := append(append(append([]string{}, args...), envInjections...), fileInjections...)
	allRefs := secret.ExtractRefsFromSlice(allStrings)

	var resolved map[string]string
	if len(allRefs) > 0 {
		if secretSource == "" {
			fmt.Fprintf(os.Stderr, "mcp-gatekeeper: secret references found but --secret-source is not set\n")
			return 1
		}
		var err error
		resolved, err = secret.ResolveAll(secretSource, allRefs)
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
	for _, inj := range fileInjections {
		idx := strings.Index(inj, "=")
		if idx < 0 {
			fmt.Fprintf(os.Stderr, "mcp-gatekeeper: invalid --file injection (missing '='): %q\n", inj)
			return 1
		}
		lhs := inj[:idx]
		rhs := inj[idx+1:]

		// Substitute secret refs in the rhs (value)
		if resolved != nil {
			rhs = secret.Substitute(rhs, resolved)
		}

		if strings.HasPrefix(lhs, "/") || strings.HasPrefix(lhs, "~") {
			// Fixed path — write secret directly to lhs
			if err := os.WriteFile(lhs, []byte(rhs), 0600); err != nil {
				fmt.Fprintf(os.Stderr, "mcp-gatekeeper: failed to write secret to %q: %v\n", lhs, err)
				return 1
			}
		} else {
			// Temp file — create, write secret, add VAR=path to env
			f, err := os.CreateTemp("", "mcp-secret-*")
			if err != nil {
				fmt.Fprintf(os.Stderr, "mcp-gatekeeper: failed to create temp file: %v\n", err)
				return 1
			}
			tmpPath := f.Name()
			tmpFiles = append(tmpFiles, tmpPath)

			if err := f.Chmod(0600); err != nil {
				f.Close()
				fmt.Fprintf(os.Stderr, "mcp-gatekeeper: failed to chmod temp file: %v\n", err)
				return 1
			}
			if _, err := f.WriteString(rhs); err != nil {
				f.Close()
				fmt.Fprintf(os.Stderr, "mcp-gatekeeper: failed to write to temp file: %v\n", err)
				return 1
			}
			f.Close()

			resolvedEnvs = append(resolvedEnvs, lhs+"="+tmpPath)
		}
	}

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

	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		return 1
	}
	return 0
}
