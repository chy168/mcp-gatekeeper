## 1. 新增相依套件

- [x] 1.1 在 `go.mod` / `go.sum` 加入 `gopkg.in/yaml.v3`

## 2. Bundle 解析邏輯（internal/secret/bundle.go）

- [x] 2.1 新增 `ParseBundle(content string) (map[string]string, error)` — parse YAML，驗證所有 top-level value 為 scalar；parse 失敗或 non-scalar 回傳含範例說明的 error
- [x] 2.2 新增 `LookupBundle(bundle map[string]string, key, bundleName string) (string, error)` — key 不存在時回傳列出所有可用 key 的 error

## 3. 修改 resolve 邏輯（internal/secret/resolve.go）

- [x] 3.1 修改 `ResolveAll(source, bundleName string, names []string)` — 改為一次 `backend.Get(ctx, bundleName)` 取回 bundle，呼叫 `ParseBundle`，再對每個 name 呼叫 `LookupBundle`
- [x] 3.2 移除 `resolveAllWithBackend` 的逐一 Get 邏輯，改以 bundle lookup 取代

## 4. CLI 新增 --secret-source-name flag（cmd/mcp-gatekeeper/main.go）

- [x] 4.1 新增 `--secret-source-name` string flag，default `"mcp-gatekeeper"`
- [x] 4.2 將 `secretSourceName` 傳入 `proxy.Run` 與 `proxy.ListTools`
- [x] 4.3 更新 flag Usage 說明

## 5. 更新 proxy 層傳參（internal/proxy）

- [x] 5.1 更新 `proxy.Run` 與 `proxy.ListTools` 的 signature，接受 `secretSourceName string`
- [x] 5.2 將 `secretSourceName` 傳入 `secret.ResolveAll` 呼叫處

## 6. 測試

- [x] 6.1 為 `ParseBundle` 新增單元測試：合法 YAML、多行純量、parse 失敗、non-scalar value
- [x] 6.2 為 `LookupBundle` 新增單元測試：key 存在、key 不存在（含可用 key 列表）、bundle 為空
- [x] 6.3 更新 `resolve.go` 相關測試以符合新的 bundle 語意
- [x] 6.4 更新或新增 integration test，驗證 `--secret-source-name` flag 與 YAML bundle 的 end-to-end 行為

## 7. 文件與遷移說明

- [x] 7.1 更新 `mcp-settings.json` 範例，改用 YAML bundle 格式
- [x] 7.2 在 CHANGELOG 或 README 標注 breaking change：`{$secret.key}` 語意變更，需將個別 secret entries 遷移至 YAML bundle
