## Why

MCP servers often require secrets (API tokens, credentials, OAuth-generated files) at startup. Currently users must expose these as plaintext in command args or environment, which is insecure and hard to manage. mcp-gatekeeper sits at the ideal interception point to fetch secrets from secure backends and inject them transparently before spawning the subprocess.

## What Changes

- Add `--secret-source=<backend>` flag to select the secret backend (`gcp`, `aws`, `keychain`)
- Add `--env=VAR={$secret.name}` flag to inject a secret as an environment variable into the subprocess
- Add `--arg=...{$secret.name}...` support to substitute secrets into subprocess command arguments
- Add `--file` flag with two modes:
  - `--file=VAR={$secret.name}`: write secret to a random temp file, inject path as env var, delete after subprocess exits
  - `--file=/path/to/file={$secret.name}`: write secret to a fixed path, no env var injected
- Secret names in `{$secret.name}` map directly to the secret's name in the chosen backend
- Backend connection config is sourced from standard environment variables (no extra flags needed):
  - `gcp` → GCP Secret Manager, project from `GOOGLE_CLOUD_PROJECT`, always fetches `latest` version
  - `aws` → AWS Secrets Manager, region from `AWS_DEFAULT_REGION` / `AWS_REGION`
  - `keychain` → OS Keychain (macOS Keychain / Linux Secret Service / Windows Credential Manager)

## Capabilities

### New Capabilities

- `secret-management`: Fetch secrets from pluggable backends (GCP Secret Manager, AWS Secrets Manager, OS Keychain) using a unified `--secret-source` flag
- `secret-injection`: Inject resolved secrets into the subprocess via env vars (`--env`), command args (`--arg`), or temp files (`--file`) using `{$secret.name}` template syntax

### Modified Capabilities

- `stdio-proxy`: subprocess spawn must apply secret injection (env, args, temp files) before exec

## Impact

- `cmd/mcp-gatekeeper/main.go`: new flags `--secret-source`, extended `--env`, `--arg`, `--file`
- `internal/proxy/proxy.go`: inject resolved env vars and args before spawning subprocess; manage temp file lifecycle
- New package `internal/secret/`: backend interface + implementations (gcp, aws, keychain)
- New dependencies: GCP Secret Manager client, AWS SDK, `zalando/go-keyring`
- Subprocess stderr remains inherited (no change)
