package proxy

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/chy168/mcp-gatekeeper/internal/filter"
)

// Run starts the subprocess and proxies stdio between client and subprocess.
// allows is the allowlist, excludes is the denylist.
func Run(command string, args, allows, excludes []string) int {
	cmd := exec.Command(command, args...)
	cmd.Stderr = os.Stderr

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
