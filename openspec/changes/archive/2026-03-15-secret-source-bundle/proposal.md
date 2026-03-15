## Why

目前每個 secret 在 backend（GCP Secret Manager、AWS Secrets Manager、OS Keychain）中各自獨立存放，使用者需管理 N 筆 backend entries。引入 YAML bundle 概念，將所有 secret 集中存放在單一 backend entry，大幅降低 secret 管理的複雜度，並減少對 backend 的 API 呼叫次數。

## What Changes

- 新增 `--secret-source-name` flag，指定 backend 中 bundle 的 secret 名稱（default: `mcp-gatekeeper`）
- **BREAKING**: `{$secret.key}` 語意由「從 backend 直接取名為 `key` 的 secret」改為「從 YAML bundle 中取 key `key` 的值」
- Secret bundle 格式為 YAML，支援多行純量（`|` block scalar），適用於 private key、SA credential 等多行內容
- YAML parse 失敗或 key 值非 scalar，立即 fail fast 並輸出含範例說明的錯誤訊息

## Capabilities

### New Capabilities

- `secret-bundle`: 以單一 YAML bundle 管理所有 secret，透過 `--secret-source-name` 指定 bundle 名稱，並定義 bundle 的 YAML 格式規範與錯誤處理行為

### Modified Capabilities

- `secret-management`: `--secret-source-name` flag 加入既有 secret source 選擇機制；secret 解析流程改為先 fetch bundle、再從 YAML 取值

## Impact

- `internal/secret/resolve.go`：解析邏輯改為先 fetch 單一 bundle，parse YAML 後再 lookup key
- `internal/secret/backend.go`：backend interface 維持不變（仍為 `Get(name) (string, error)`），bundle fetch 在 resolve 層處理
- `cmd/mcp-gatekeeper/main.go`：新增 `--secret-source-name` flag
- 新增 YAML 相關 dependency（`gopkg.in/yaml.v3`）
- 現有使用個別 secret 名稱的用法為 breaking change，需更新 `mcp-settings.json`
