## 1. 擴充 Backend interface（internal/secret/backend.go）

- [x] 1.1 在 `Backend` interface 新增 `Set(ctx context.Context, name, value string) error`

## 2. GCP backend 實作 Set（internal/secret/gcp.go）

- [x] 2.1 實作 `gcpBackend.Set`：呼叫 GCP API 確認 secret 是否存在
- [x] 2.2 若 secret 不存在，執行 `CreateSecret` 建立
- [x] 2.3 執行 `AddSecretVersion` 寫入新值
- [x] 2.4 錯誤時回傳含 IAM 權限提示的清楚 error message

## 3. AWS backend 實作 Set（internal/secret/aws.go）

- [x] 3.1 實作 `awsBackend.Set`：嘗試 `PutSecretValue`
- [x] 3.2 若 secret 不存在（ResourceNotFoundException），先執行 `CreateSecret` 再 `PutSecretValue`

## 4. Keychain backend 實作 Set（internal/secret/keychain.go）

- [x] 4.1 實作 `keychainBackend.Set`：呼叫 `keyring.Set(keychainService, name, value)`

## 5. Bundle 寫入輔助函式（internal/secret/bundle.go）

- [x] 5.1 新增 `SerializeBundle(bundle map[string]string) (string, error)` — 將 map 序列化為 YAML string
- [x] 5.2 新增 `SetBundleKey(ctx, backend, bundleName, key, value string) error` — fetch → parse → upsert key → serialize → `backend.Set`（bundle 不存在時從空 map 開始）
- [x] 5.3 新增 `DeleteBundleKey(ctx, backend, bundleName, key string) error` — fetch → parse → delete key → serialize → `backend.Set`

## 6. 新增 cobra 依賴與 binary 骨架（cmd/mcp-gatekeeper-secret/）

- [x] 6.1 在 `go.mod` 加入 `github.com/spf13/cobra`
- [x] 6.2 建立 `cmd/mcp-gatekeeper-secret/main.go`：root command，`PersistentFlags` 定義 `--secret-source`、`--secret-source-name`（default `mcp-gatekeeper`）

## 7. 實作四個 subcommands

- [x] 7.1 `list`：fetch bundle → parse → 逐行輸出所有 key 至 stdout
- [x] 7.2 `get <key>`：fetch bundle → lookup key → 輸出 `<key>: ****`；`--reveal` 時輸出明文
- [x] 7.3 `set <key> [value]`：`--from-file` 優先讀檔；否則用 positional arg；兩者皆無則 error；呼叫 `SetBundleKey`，輸出確認訊息
- [x] 7.4 `delete <key>`：呼叫 `DeleteBundleKey`，key 不存在時 error，成功時輸出確認訊息

## 8. Makefile 更新

- [x] 8.1 新增 `mcp-gatekeeper-secret` build target
- [x] 8.2 更新 `cross` target，加入 `mcp-gatekeeper-secret` 的 linux/darwin/windows cross-compile

## 9. 測試

- [x] 9.1 為 `SerializeBundle` 新增單元測試：正常 map、含多行 value、空 map
- [x] 9.2 為 `SetBundleKey` 新增單元測試：新增 key、更新既有 key、bundle 不存在時從空 map 建立
- [x] 9.3 為 `DeleteBundleKey` 新增單元測試：成功刪除、key 不存在時 error
- [x] 9.4 為各 backend `Set` 新增單元測試（mock GCP / AWS client；keychain 用 test helper）

## 10. 文件

- [x] 10.1 更新 README，加入 `mcp-gatekeeper-secret` 使用說明及各 subcommand 範例
- [x] 10.2 說明 GCP 需額外 IAM 權限：`roles/secretmanager.secretVersionAdder`
