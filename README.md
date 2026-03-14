# mcp-keeper

A lightweight stdio proxy that sits between an MCP client (Claude Desktop, Cursor, etc.) and an MCP server, letting you filter which tools are visible to the LLM without modifying the server.

## Install

**Homebrew (macOS / Linux):**
```sh
brew install chy168/tap/mcp-keeper
```

**Download binary** from [GitHub Releases](https://github.com/chy168/mcp-keeper/releases).

**Build from source:**
```sh
go install github.com/chy168/mcp-keeper/cmd/mcp-keeper@latest
```

## Usage

```
mcp-keeper [--allow=<glob>]... [--exclude=<glob>]... <command> [args...]
```

Prefix your existing MCP server command with `mcp-keeper`:

```sh
# Before
uvx mcp-server-time

# After (transparent proxy, no filtering)
mcp-keeper uvx mcp-server-time

# Only expose tools matching get_*
mcp-keeper --allow="get_*" uvx mcp-server-time

# Expose all tools except delete_*
mcp-keeper --exclude="delete_*" uvx mcp-server-filesystem /tmp

# Combine: keep *_file tools, but remove delete_file
mcp-keeper --allow="*_file" --exclude="delete_*" uvx mcp-server-filesystem /tmp
```

## Flags

| Flag | Description |
|------|-------------|
| `--allow=<glob>` | **Allowlist**: only keep tools whose name matches this glob. Can be specified multiple times (OR logic). |
| `--exclude=<glob>` | **Denylist**: remove tools whose name matches this glob. Can be specified multiple times (OR logic). |

When both are specified, `--allow` is applied first, then `--exclude`.

Glob patterns use Go's [`path.Match`](https://pkg.go.dev/path#Match) syntax (`*`, `?`, `[abc]`).

## MCP Client Config Example

`claude_desktop_config.json`:
```json
{
  "mcpServers": {
    "time": {
      "command": "mcp-keeper",
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
mcp-keeper  ← intercepts tools/list responses, filters tool list
    │ subprocess stdin / stdout
    ▼
MCP Server (uvx mcp-server-time)
```

- All messages other than `tools/list` responses are forwarded verbatim.
- Only stdio transport is supported (no SSE/HTTP/WebSocket).
- No configuration files — all settings are CLI flags.
