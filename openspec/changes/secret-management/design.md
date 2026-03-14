## Context

mcp-gatekeeper is a CLI stdio proxy that spawns a subprocess and pipes JSON-RPC between client and server. It currently supports tool filtering via `--allow` / `--exclude` flags. All config is purely CLI-based with no config files.

Secret injection must happen before subprocess exec — mcp-gatekeeper is the right place because it owns the subprocess lifecycle. The goal is to keep the same CLI-flag-only design philosophy.

## Goals / Non-Goals

**Goals:**
- Fetch secrets from GCP Secret Manager, AWS Secrets Manager, or OS Keychain at startup
- Inject secrets into subprocess env vars, command args, or temp files via `{$secret.name}` template syntax
- One backend per invocation (`--secret-source` is a single flag)
- Zero config files — all settings via CLI flags and standard backend env vars
- Temp files written for `--file=VAR=...` are deleted after subprocess exits (including on crash/signal)

**Non-Goals:**
- Supporting multiple secret backends in a single invocation
- Secret rotation or caching during subprocess lifetime
- OAuth login flow management
- Secret creation / write-back to backends
- Windows support for `--file` temp file cleanup on hard kill (best-effort)

## Decisions

**D1: 使用 `helmfile/vals` 作為 secret 解析引擎**

不自行實作各 backend，改用 `github.com/helmfile/vals` library：
```go
runtime, _ := vals.New(256) // 256 = LRU cache size
resolved, _ := runtime.Eval(map[string]interface{}{"v": valsURI})
```
vals 負責實際的 GCP / AWS / Keychain 通訊，mcp-gatekeeper 只需維護一層 translation layer。好處：30+ backends 免費取得，SDK 依賴由 vals 管理。

**D2: `{$secret.name}` 語法轉譯為 vals URI**

`internal/secret/translate.go` 負責將使用者語法轉為 vals URI：

| `--secret-source` | `{$secret.name}` 轉譯結果 |
|---|---|
| `gcp` | `ref+gcpsecrets://projects/$GOOGLE_CLOUD_PROJECT/secrets/name` |
| `aws` | `ref+awssecrets://name` |
| `keychain` | `ref+keychain://mcp-gatekeeper/name` |

使用者介面完全不變，translation 在內部透明處理。

**D3: Template resolution before subprocess spawn**

Parse all `--env`, `--arg`, and `--file` values for `{$secret.name}` patterns before `exec`. All secrets are fetched in a single pass at startup — no lazy resolution during proxy lifetime. This keeps the hot path (stdio forwarding) completely free of secret logic.

**D3: `--file` modes distinguished by left-hand side prefix**

- Left side starts with `/` or `~` → fixed path mode: write secret to that path, no env var
- Otherwise → temp file mode: write to `os.CreateTemp("", "mcp-secret-*")`, inject path as env var

Temp files use `defer` + signal handler (`os/signal`, `syscall.SIGTERM`, `SIGINT`) to ensure cleanup. File permissions set to `0600`.

**D4: `--arg` substitution applies to subprocess args only**

`{$secret.name}` in `--arg` values replaces the placeholder in the token passed to the subprocess. mcp-gatekeeper's own flags are parsed before substitution, so no risk of secret values being interpreted as gatekeeper flags.

**D5: 唯一外部依賴：`github.com/helmfile/vals`**

不直接依賴 GCP SDK、AWS SDK 或 `go-keyring`——這些全由 vals 管理。mcp-gatekeeper 只需 `go get github.com/helmfile/vals`。

**D6: Fail fast on missing / unresolvable secrets**

If any `{$secret.name}` cannot be resolved (backend error, secret not found, missing `--secret-source`), mcp-gatekeeper exits with a non-zero status and prints an error to stderr before spawning the subprocess. No partial injection.

## Risks / Trade-offs

- **Startup latency**: fetching secrets over network (GCP/AWS) adds ~100–500ms cold start. Acceptable for MCP server use case (infrequent restarts). → No mitigation needed.
- **Secret values in process args**: `--arg=--token={$secret.x}` will appear in `ps` output after substitution. → Document this; recommend `--env` or `--file` for sensitive values.
- **Temp file leak on SIGKILL**: `defer` and signal handlers cannot catch `SIGKILL`. → Document as known limitation; use OS-level temp dir cleanup (`/tmp` is cleared on reboot).
- **`go-keyring` requires D-Bus on Linux**: headless Linux environments (CI, Docker) may not have a keyring daemon. → Document; `keychain` backend is intended for desktop use only.

## Open Questions

- Should `--arg` support be positional (replace in any subprocess arg token) or require explicit flag-value syntax? Current design: replace in any token that contains `{$secret.*}`.
- Should we support `{$secret.name}` inside `--allow` / `--exclude` globs? (Probably not — tool names are not secrets.)
