## Why

使用者目前需要直接操作各 secret backend 的原生工具（gcloud、aws CLI、Keychain Access）來建立和維護 YAML bundle，每個 backend 介面不同、學習成本高。新增 `mcp-gatekeeper-secret` binary，提供統一的 CRUD 介面管理 bundle 內容，無需熟悉各 backend 工具。

## What Changes

- 新增 binary `cmd/mcp-gatekeeper-secret/`，提供 `list`、`get`、`set`、`delete` subcommands
- 擴充 `Backend` interface，新增 `Set(ctx, name, value string) error`，供 `set` / `delete` 寫回 bundle
- 各 backend 實作（gcp、aws、keychain）實作 `Set` method
- `--secret-source` 與 `--secret-source-name` 為 global flags，兩個 binary 共用語意
- `get` 預設遮蔽 value（顯示 `****`），加 `--reveal` 顯示明文
- `set` 支援直接傳值或 `--from-file=<path>` 讀取多行內容（private key、SA JSON 等）

## Capabilities

### New Capabilities

- `secret-manage-cli`: `mcp-gatekeeper-secret` binary 的 subcommand 介面（list / get / set / delete）及其 UX 行為規範
- `secret-backend-write`: `Backend` interface 的 `Set` method 及三個 backend（gcp、aws、keychain）的寫入實作

### Modified Capabilities

- `secret-management`: `Backend` interface 擴充 `Set`，影響既有 interface contract

## Impact

- `internal/secret/backend.go`：`Backend` interface 新增 `Set`
- `internal/secret/gcp.go`、`aws.go`、`keychain.go`：各自實作 `Set`
- `cmd/mcp-gatekeeper-secret/main.go`：新增 binary，dispatch subcommands
- `Makefile`：新增 `mcp-gatekeeper-secret` build target 及 cross-compile
- 與 `secret-source-bundle` change 的 `internal/secret` package 共用，無重複邏輯
