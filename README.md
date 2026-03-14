# mcp-gatekeeper

A lightweight stdio proxy that sits between an MCP client (Claude Desktop, Cursor, etc.) and an MCP server, letting you filter which tools are visible to the LLM without modifying the server.

## Install

**Homebrew (macOS / Linux):**
```sh
brew install chy168/tap/mcp-gatekeeper
```

**Download binary** from [GitHub Releases](https://github.com/chy168/mcp-gatekeeper/releases).

**Build from source:**
```sh
go install github.com/chy168/mcp-gatekeeper/cmd/mcp-gatekeeper@latest
```

## Usage

```
mcp-gatekeeper [--allow=<glob>]... [--exclude=<glob>]... [--list-tools | --list-all-tools] <command> [args...]
```

Prefix your existing MCP server command with `mcp-gatekeeper`:

```sh
# Before
uvx mcp-server-time

# After (transparent proxy, no filtering)
mcp-gatekeeper uvx mcp-server-time

# Discover all tools exposed by the server
mcp-gatekeeper --list-all-tools uvx mcp-server-time

# Only expose tools matching get_*
mcp-gatekeeper --allow="get_*" uvx mcp-server-time

# Verify which tools remain after filtering
mcp-gatekeeper --list-tools --allow="get_*" uvx mcp-server-time

# Expose all tools except delete_*
mcp-gatekeeper --exclude="delete_*" uvx mcp-server-filesystem /tmp

# Combine: keep *_file tools, but remove delete_file
mcp-gatekeeper --allow="*_file" --exclude="delete_*" uvx mcp-server-filesystem /tmp
```

## Flags

mcp-gatekeeper flags must appear **before** the server command. Any flags after the command are passed through to the server as-is.

| Flag | Description |
|------|-------------|
| `--allow=<glob>` | **Allowlist**: only keep tools whose name matches this glob. Can be specified multiple times (OR logic). |
| `--exclude=<glob>` | **Denylist**: remove tools whose name matches this glob. Can be specified multiple times (OR logic). |
| `--list-all-tools` | Print all available tool names and descriptions (ignores `--allow`/`--exclude`), then exit. Useful for discovering tool names. |
| `--list-tools` | Print tool names and descriptions after applying `--allow`/`--exclude` filters, then exit. Useful for verifying your filter setup. |

When both `--allow` and `--exclude` are specified, `--allow` is applied first, then `--exclude`.

Glob patterns use Go's [`path.Match`](https://pkg.go.dev/path#Match) syntax (`*`, `?`, `[abc]`).

## MCP Client Config Example

`claude_desktop_config.json`:
```json
{
  "mcpServers": {
    "time": {
      "command": "mcp-gatekeeper",
      "args": ["--allow=get_*", "uvx", "mcp-server-time"]
    }
  }
}
```

## How It Works

```
MCP Client (Claude Desktop)
    │ stdin / stdout
    ▼
mcp-gatekeeper  ← intercepts tools/list responses, filters tool list
    │ subprocess stdin / stdout
    ▼
MCP Server (uvx mcp-server-time)
```

- All messages other than `tools/list` responses are forwarded verbatim.
- Only stdio transport is supported (no SSE/HTTP/WebSocket).
- No configuration files — all settings are CLI flags.
