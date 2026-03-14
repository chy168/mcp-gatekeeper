package main

import (
	"fmt"
	"os"

	flag "github.com/spf13/pflag"

	"github.com/chy168/mcp-gatekeeper/internal/proxy"
)

func main() {
	var allows []string
	var excludes []string
	var listTools bool
	var listAllTools bool

	flag.SetInterspersed(false)
	flag.StringArrayVar(&allows, "allow", nil, "Allowlist glob pattern for tool names (may be specified multiple times)")
	flag.StringArrayVar(&excludes, "exclude", nil, "Denylist glob pattern for tool names (may be specified multiple times)")
	flag.BoolVar(&listTools, "list-tools", false, "List tools after applying --allow/--exclude filters, then exit")
	flag.BoolVar(&listAllTools, "list-all-tools", false, "List all available tools from the MCP server (ignores filters), then exit")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: mcp-gatekeeper [--allow=<glob>]... [--exclude=<glob>]... [--list-tools | --list-all-tools] <command> [args...]\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	if listAllTools {
		os.Exit(proxy.ListTools(args[0], args[1:], nil, nil))
	}

	if listTools {
		os.Exit(proxy.ListTools(args[0], args[1:], allows, excludes))
	}

	os.Exit(proxy.Run(args[0], args[1:], allows, excludes))
}
