# mcp-keeper

Lightweight Go stdio proxy — sits between MCP client and server, filters `tools/list` responses via glob patterns (`--filter` allowlist, `--exclude` denylist).

## Environment

**Go**: managed via `gvm`. Always prefix Go commands:
```sh
bash -l -c "source ~/.gvm/scripts/gvm && gvm use go1.24 && go <cmd>"
```
Or just use the Makefile (it handles this automatically).

**GitHub**: `chy168` → module is `github.com/chy168/mcp-keeper`

## Structure

```
cmd/mcp-keeper/main.go      # CLI entry: --filter / --exclude flags
internal/filter/filter.go   # JSON-RPC detection + glob filtering logic
internal/proxy/proxy.go     # subprocess spawn + bidirectional stdio pipe
scripts/integration_test.py # integration tests (uv run)
```

## Common Commands

```sh
make build                          # build → bin/mcp-keeper
make test                           # go test ./...
make cross                          # cross-compile linux/darwin/windows
uv run scripts/integration_test.py # integration tests (requires uvx + npx)
```

## Key Facts

- MCP stdio transport: newline-delimited JSON-RPC 2.0
- Filter order: allowlist (`--filter`) applied first, then denylist (`--exclude`)
- `bufio.Scanner` buffer set to 1 MB — handles large `tools/list` responses
- Subprocess stderr inherited directly (not proxied)
- **`uvx`** = PyPI only. npm-based MCP servers (e.g. `@modelcontextprotocol/server-filesystem`) need **`npx`**
