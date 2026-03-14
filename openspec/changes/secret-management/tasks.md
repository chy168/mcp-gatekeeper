## 1. 依賴與模組設定

- [x] 1.1 新增 Go module 依賴：`cloud.google.com/go/secretmanager`、`github.com/aws/aws-sdk-go-v2/service/secretsmanager`、`github.com/zalando/go-keyring`（原規劃用 `helmfile/vals`，改為直接依賴各 SDK）
- [x] 1.2 建立 `internal/secret/` 目錄結構（`backend.go`、`resolve.go`、`gcp.go`、`aws.go`、`keychain.go`）

## 2. Backend 介面與實作

- [x] 2.1 實作 `internal/secret/backend.go`：定義 `Backend` interface 與 `NewBackend(source string)` factory（原規劃為 `translate.go` + vals URI，改為直接 backend 實作）
- [x] 2.2 `gcp` backend 在 `newGCPBackend()` 驗證 `GOOGLE_CLOUD_PROJECT` 存在，否則提前回傳明確 error

## 3. Template 解析與 Secret 解析

- [x] 3.1 實作 `internal/secret/resolve.go`：`ExtractRefs`、`ExtractRefsFromSlice`——解析字串中所有 `{$secret.<name>}` pattern
- [x] 3.2 實作 `ResolveAll(source string, refs []string) (map[string]string, error)`：透過 `NewBackend` 取得 backend，逐一呼叫 `backend.Get()`，任一失敗即回傳 error（fail fast）
- [x] 3.3 實作字串替換函式：`Substitute`、`SubstituteSlice`——將已解析的值替換回原始字串

## 4. CLI Flag 擴充

- [x] 4.1 在 `cmd/mcp-gatekeeper/main.go` 新增 `--secret-source` flag
- [x] 4.2 新增 `--env=KEY={$secret.name}` flag（支援重複指定）
- [x] 4.3 新增 `--file=TARGET={$secret.name}` flag（支援重複指定）
- [x] 4.4 擴充子行程 args 處理：對所有 subprocess arg token 執行 `{$secret.*}` 替換

## 5. 注入邏輯

- [x] 5.1 在子行程啟動前，對所有 `--env` value、`--file` value、subprocess args 收集 `{$secret.*}` 引用
- [x] 5.2 若有 `{$secret.*}` 引用但未指定 `--secret-source`，輸出錯誤並 exit
- [x] 5.3 呼叫 `ResolveAll` 批次解析所有 secret，任一失敗則輸出錯誤並 exit，不啟動子行程
- [x] 5.4 將解析後的 env var 注入子行程的 `cmd.Env`
- [x] 5.5 實作 `--file=VAR=...`（temp file 模式）：`os.CreateTemp`、寫入內容、chmod `0600`、將路徑注入 `cmd.Env`
- [x] 5.6 實作 `--file=/path=...`（fixed path 模式）：寫入指定路徑、chmod `0600`，不設 env var
- [x] 5.7 將替換後的 args 傳給子行程

## 6. Temp File 生命週期管理

- [x] 6.1 使用 `defer` 在子行程結束後刪除所有 temp file
- [x] 6.2 設定 signal handler（`SIGTERM`、`SIGINT`）確保 temp file 在收到訊號時也被清理

## 7. 測試

- [x] 7.1 `internal/secret/backend_test.go`：`mockBackend` 測試 `resolveAllWithBackend`（成功、缺 secret、空 list）、`NewBackend` 不支援的 source（原規劃為 translate.go tests，改為 backend_test.go）
- [x] 7.2 `internal/secret/resolve_test.go`：pattern 解析、字串替換、`ExtractRefsFromSlice`、`SubstituteSlice`
- [ ] 7.3 為 `--file` 兩種模式加 unit test（temp file 建立與 cleanup、fixed path 寫入）
- [ ] 7.4 更新整合測試 `scripts/integration_test.py`：加入 `--env` 注入情境

## 8. 文件

- [x] 8.1 更新 `README.md`：新增 `--secret-source`、`--env`、`--file` flag 說明與範例
- [x] 8.2 說明各 backend 的前置設定（GCP ADC、AWS credential chain、Keychain 新增方式）
- [x] 8.3 說明 `--arg` 注入的 `ps` 曝光風險，建議改用 `--env` 或 `--file`
- [x] 8.4 說明 `keychain` backend 僅適用於桌面環境（Linux 需要 D-Bus）
