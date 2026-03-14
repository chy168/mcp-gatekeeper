package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/chy168/mcp-keeper/internal/proxy"
)

type multiFlag []string

func (f *multiFlag) String() string { return fmt.Sprintf("%v", *f) }
func (f *multiFlag) Set(v string) error {
	*f = append(*f, v)
	return nil
}

func main() {
	var filters multiFlag
	var excludes multiFlag

	flag.Var(&filters, "filter", "Allowlist glob pattern for tool names (may be specified multiple times)")
	flag.Var(&excludes, "exclude", "Denylist glob pattern for tool names (may be specified multiple times)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: mcp-keeper [--filter=<glob>]... [--exclude=<glob>]... <command> [args...]\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	os.Exit(proxy.Run(args[0], args[1:], []string(filters), []string(excludes)))
}
