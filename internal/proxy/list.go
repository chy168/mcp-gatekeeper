package proxy

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/chy168/mcp-gatekeeper/internal/filter"
	"github.com/chy168/mcp-gatekeeper/internal/secret"
)

type toolEntry struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type jsonRPCMsg struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Result  *struct {
		Tools json.RawMessage `json:"tools"`
	} `json:"result"`
}

// ListTools starts the subprocess, performs the MCP handshake, sends tools/list,
// and prints tool names and descriptions to stdout.
// If allows/excludes are provided, filtering is applied before printing.
func ListTools(command string, args, allows, excludes, envInjections, fileInjections []string, secretSource string) int {
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

	// Build extra env vars from envInjections
	var resolvedEnvs []string
	for _, inj := range envInjections {
		substituted := inj
		if resolved != nil {
			substituted = secret.Substitute(inj, resolved)
		}
		resolvedEnvs = append(resolvedEnvs, substituted)
	}

	// Handle fileInjections (no temp file cleanup needed for list mode)
	for _, inj := range fileInjections {
		idx := strings.Index(inj, "=")
		if idx < 0 {
			fmt.Fprintf(os.Stderr, "mcp-gatekeeper: invalid --file injection (missing '='): %q\n", inj)
			return 1
		}
		lhs := inj[:idx]
		rhs := inj[idx+1:]

		if resolved != nil {
			rhs = secret.Substitute(rhs, resolved)
		}

		if strings.HasPrefix(lhs, "/") || strings.HasPrefix(lhs, "~") {
			if err := os.WriteFile(lhs, []byte(rhs), 0600); err != nil {
				fmt.Fprintf(os.Stderr, "mcp-gatekeeper: failed to write secret to %q: %v\n", lhs, err)
				return 1
			}
		} else {
			f, err := os.CreateTemp("", "mcp-secret-*")
			if err != nil {
				fmt.Fprintf(os.Stderr, "mcp-gatekeeper: failed to create temp file: %v\n", err)
				return 1
			}
			tmpPath := f.Name()
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
	defer cmd.Wait()
	defer stdinPipe.Close()

	scanner := bufio.NewScanner(stdoutPipe)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	sendLine := func(s string) {
		stdinPipe.Write([]byte(s + "\n"))
	}

	// Step 1: initialize
	sendLine(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"mcp-gatekeeper","version":"0.1.0"}}}`)

	// Wait for initialize response (id=1)
	for scanner.Scan() {
		var msg jsonRPCMsg
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			continue
		}
		if string(msg.ID) == "1" {
			break
		}
	}

	// Step 2: notify initialized
	sendLine(`{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}`)

	// Step 3: tools/list
	sendLine(`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`)

	// Wait for tools/list response (id=2)
	for scanner.Scan() {
		var msg jsonRPCMsg
		line := scanner.Bytes()
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}
		if string(msg.ID) != "2" || msg.Result == nil || msg.Result.Tools == nil {
			continue
		}

		filtered, err := filter.FilterToolsListResponse(line, allows, excludes)
		if err != nil {
			fmt.Fprintf(os.Stderr, "mcp-gatekeeper: failed to filter tools: %v\n", err)
			return 1
		}

		var filteredMsg jsonRPCMsg
		if err := json.Unmarshal(filtered, &filteredMsg); err != nil || filteredMsg.Result == nil {
			fmt.Fprintf(os.Stderr, "mcp-gatekeeper: failed to parse filtered response\n")
			return 1
		}

		var tools []toolEntry
		if err := json.Unmarshal(filteredMsg.Result.Tools, &tools); err != nil {
			fmt.Fprintf(os.Stderr, "mcp-gatekeeper: failed to parse tools: %v\n", err)
			return 1
		}

		fmt.Printf("%-40s %s\n", "TOOL NAME", "DESCRIPTION")
		fmt.Printf("%-40s %s\n", "─────────────────────────────────────────", "─────────────────────────────────────────────")
		for _, t := range tools {
			fmt.Printf("%-40s %s\n", t.Name, t.Description)
		}
		return 0
	}

	fmt.Fprintf(os.Stderr, "mcp-gatekeeper: no tools/list response received\n")
	return 1
}
