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
mcp-gatekeeper [--allow=<glob>]... [--exclude=<glob>]... [--list-tools | --list-all-tools] \
               [--secret-source=<backend>] [--env=KEY={$secret.name}]... [--file=VAR={$secret.name}]... \
               <command> [args...]
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
| `--secret-source=<backend>` | Secret backend to use: `gcp`, `aws`, or `keychain`. Required when any `{$secret.*}` reference is used. |
| `--env=KEY={$secret.name}` | Inject a secret as an env var into the subprocess. Can be specified multiple times. |
| `--file=VAR={$secret.name}` | Write secret to a temp file (deleted on exit), inject its path as env var `VAR`. |
| `--file=/path/to/file={$secret.name}` | Write secret to a fixed path. No env var injected. |

When both `--allow` and `--exclude` are specified, `--allow` is applied first, then `--exclude`.

Glob patterns use Go's [`path.Match`](https://pkg.go.dev/path#Match) syntax (`*`, `?`, `[abc]`).

## Secret Injection

`{$secret.name}` placeholders are resolved at startup from the configured backend and substituted into `--env`, `--file`, and subprocess args before the server is launched.

### Basic usage

```sh
# Inject a secret as env var (from GCP Secret Manager)
mcp-gatekeeper --secret-source=gcp --env=API_TOKEN={$secret.my_api_token} uvx mcp-server-foo

# Inject into subprocess args
mcp-gatekeeper --secret-source=aws --allow="query_*" uvx mcp-server-foo --token={$secret.my_token}

# Write credential file to temp path (e.g. for Google ADC)
mcp-gatekeeper --secret-source=gcp \
  --file=GOOGLE_APPLICATION_CREDENTIALS={$secret.gcp_sa_key} \
  uvx mcp-server-gcp

# Write to a fixed path
mcp-gatekeeper --secret-source=keychain \
  --file=/tmp/creds.json={$secret.my_creds} \
  uvx mcp-server-foo
```

### Verify your setup (GCP example)

```sh
# 1. Create a test secret
echo -n "hello-from-gcp" | gcloud secrets create my-test-secret \
  --project=my-gcp-project \
  --data-file=-

# 2. Verify mcp-gatekeeper can fetch and inject it
GOOGLE_CLOUD_PROJECT=my-gcp-project \
  mcp-gatekeeper --secret-source=gcp \
  --env='MY_SECRET={$secret.my-test-secret}' \
  sh -c 'echo "SECRET=$MY_SECRET"'

# Expected output: SECRET=hello-from-gcp
```

> **Note**: Always quote `{$secret.*}` placeholders with single quotes in the shell to prevent `$secret` from being expanded before mcp-gatekeeper sees it.

### Backend setup
#### GCP Secret Manager
**`--secret-source=gcp`**
- Requires `GOOGLE_CLOUD_PROJECT` env var
- Authentication via Application Default Credentials (ADC): run `gcloud auth application-default login`

#### AWS Secrets Manager
**`--secret-source=aws`**
- Requires `AWS_DEFAULT_REGION` or `AWS_REGION` env var
- Authentication via standard AWS credential chain (`~/.aws/credentials`, IAM role, env vars, etc.)

#### macOS Keychain
**`--secret-source=keychain`**
- macOS: Keychain, Linux: Secret Service (requires D-Bus — desktop only), Windows: Credential Manager
- Service name is fixed to `mcp-gatekeeper`; account name is the secret name
- Add a secret on macOS:
  ```
  # secret value
  security add-generic-password -s mcp-gatekeeper -a my_token -w "secret-value"

  # credential file
  security add-generic-password \
    -s mcp-gatekeeper \
    -a my_creds_file \
    -w "$(cat /path/to/credentials.json)"
  ```

### Security note

> **Note**: Using `{$secret.name}` inside subprocess args (e.g. `--token={$secret.x}`) will expose the secret value in process listings (`ps`). Prefer `--env` or `--file` for sensitive values.

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

### With secret injection

```json
{
  "mcpServers": {
    "jira": {
      "command": "mcp-gatekeeper",
      "args": [
        "--secret-source=gcp",
        "--env=JIRA_API_TOKEN={$secret.jira_api_token}",
        "--allow=get_*",
        "uvx", "mcp-server-jira"
      ]
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
