## 1. 依賴與模組設定

- [ ] 1.1 新增 Go module 依賴：`github.com/helmfile/vals`
- [ ] 1.2 建立 `internal/secret/` 目錄結構（`translate.go`、`resolve.go`）

## 2. URI Translation Layer

- [ ] 2.1 實作 `internal/secret/translate.go`：`ToValsURI(source, name string) (string, error)`
  - `gcp` → `ref+gcpsecrets://projects/$GOOGLE_CLOUD_PROJECT/secrets/<name>`
  - `aws` → `ref+awssecrets://<name>`
  - `keychain` → `ref+keychain://mcp-gatekeeper/<name>`
  - 其他值回傳 error
- [ ] 2.2 `gcp` 時驗證 `GOOGLE_CLOUD_PROJECT` 存在，否則提前回傳明確 error

## 3. Template 解析與 Secret 解析

- [ ] 3.1 實作 `internal/secret/resolve.go`：解析字串中所有 `{$secret.<name>}` pattern，回傳所有引用的 name 清單
- [ ] 3.2 實作 `ResolveAll(source string, refs []string) (map[string]string, error)`：將所有 name 轉為 vals URI，呼叫 `vals.New(256)` 批次解析，任一失敗即回傳 error（fail fast）
- [ ] 3.3 實作字串替換函式：將已解析的值替換回原始字串

## 4. CLI Flag 擴充

- [ ] 4.1 在 `cmd/mcp-gatekeeper/main.go` 新增 `--secret-source` flag
- [ ] 4.2 新增 `--env=KEY={$secret.name}` flag（支援重複指定）
- [ ] 4.3 新增 `--file=TARGET={$secret.name}` flag（支援重複指定）
- [ ] 4.4 擴充子行程 args 處理：對所有 subprocess arg token 執行 `{$secret.*}` 替換

## 5. 注入邏輯

- [ ] 5.1 在子行程啟動前，對所有 `--env` value、`--file` value、subprocess args 收集 `{$secret.*}` 引用
- [ ] 5.2 若有 `{$secret.*}` 引用但未指定 `--secret-source`，輸出錯誤並 exit
- [ ] 5.3 呼叫 `ResolveAll` 批次解析所有 secret，任一失敗則輸出錯誤並 exit，不啟動子行程
- [ ] 5.4 將解析後的 env var 注入子行程的 `cmd.Env`
- [ ] 5.5 實作 `--file=VAR=...`（temp file 模式）：`os.CreateTemp`、寫入內容、chmod `0600`、將路徑注入 `cmd.Env`
- [ ] 5.6 實作 `--file=/path=...`（fixed path 模式）：寫入指定路徑、chmod `0600`，不設 env var
- [ ] 5.7 將替換後的 args 傳給子行程

## 6. Temp File 生命週期管理

- [ ] 6.1 使用 `defer` 在子行程結束後刪除所有 temp file
- [ ] 6.2 設定 signal handler（`SIGTERM`、`SIGINT`）確保 temp file 在收到訊號時也被清理

## 7. 測試

- [ ] 7.1 為 `translate.go` 加 unit test（各 source 的 URI 轉換、缺少環境變數的 error）
- [ ] 7.2 為 `resolve.go` 加 unit test（pattern 解析、字串替換、fail fast 行為）
- [ ] 7.3 為 `--file` 兩種模式加 unit test（temp file 建立與 cleanup、fixed path 寫入）
- [ ] 7.4 更新整合測試 `scripts/integration_test.py`：加入 `--env` 注入情境

## 8. 文件

- [ ] 8.1 更新 `README.md`：新增 `--secret-source`、`--env`、`--file` flag 說明與範例
- [ ] 8.2 說明各 backend 的前置設定（GCP ADC、AWS credential chain、Keychain 新增方式）
- [ ] 8.3 說明 `--arg` 注入的 `ps` 曝光風險，建議改用 `--env` 或 `--file`
- [ ] 8.4 說明 `keychain` backend 僅適用於桌面環境（Linux 需要 D-Bus）
