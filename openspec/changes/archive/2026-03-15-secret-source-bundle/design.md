## Context

目前 `ResolveAll` 對每個 `{$secret.key}` 分別呼叫一次 backend `Get(ctx, name)`，secret name 直接對應 backend 中的 entry 名稱。引入 bundle 後，改為一次 `Get(ctx, bundleName)` 取回整份 YAML，再從解析結果中 lookup 各個 key。

`Backend` interface（`Get(ctx, name) (string, error)`）維持不變，bundle fetch 完全在 `resolve.go` 層處理，各 backend 實作（gcp、aws、keychain）無需修改。

## Goals / Non-Goals

**Goals:**
- 一次 backend API call 取回所有 secrets
- 支援 YAML bundle 格式，包含多行純量（private key、SA JSON 等）
- `--secret-source-name` flag 指定 bundle 名稱，default `mcp-gatekeeper`
- 清楚的 fail fast 錯誤訊息，含原因與 YAML 範例

**Non-Goals:**
- 不提供 bundle 的建立/更新 CLI（secret 管理仍由使用者自行操作各 backend 工具）
- 不支援 bundle 內的 nested YAML object（僅支援 top-level scalar values）
- 不維持對舊版個別 secret 名稱語意的向後相容

## Decisions

### 決策一：Bundle 在 resolve 層處理，不下沉至 backend

`Backend.Get` 仍取單一 secret。`ResolveAll` 改成先 fetch bundle、parse YAML、再 lookup。

**理由**：各 backend 實作不需改動，變更範圍最小；YAML parsing 屬應用邏輯，不屬 backend transport。

**替代方案**：為 backend interface 新增 `GetBundle` method → 每個 backend 都要改，過度耦合。

---

### 決策二：Bundle 格式為 YAML

**理由**：多行純量（`|` block scalar）對 private key、JSON credential 等內容的手動管理明顯優於 JSON（不需 `\n` escape）；`gopkg.in/yaml.v3` 已是 Go 生態成熟套件。

**替代方案**：JSON → 多行內容需手動 escape，UX 較差。`.env` 格式 → 不支援多行。

---

### 決策三：Non-scalar value 視為錯誤

若 YAML bundle 中某 key 的值是 map 或 sequence，直接 fail fast。

**理由**：所有注入目標（env var、file 內容、command arg）均為 string，非 scalar 無法有意義地注入。明確報錯優於靜默行為。

---

### 決策四：錯誤訊息附 YAML 範例

所有 bundle 相關錯誤（parse 失敗、non-scalar、key 不存在）均附上正確格式範例與可用 key 列表（key 不存在時）。

**理由**：使用者首次設定 bundle 時最需要引導；範例比文件連結更直接有效。

## Risks / Trade-offs

**Breaking change：語意完全改變**
→ 現有使用個別 secret 名稱的設定（如 `mcp-settings.json`）必須遷移至 YAML bundle。需在 CHANGELOG 與文件中明確標示。

**Bundle 越來越大**
→ 所有 secrets 集中在一個 entry，長期可能累積過多。目前屬個人工具，規模不大，暫不處理；未來可考慮多 bundle 支援。

**一個 access control 邊界**
→ 任何能讀取 bundle 的 process 可取得全部 secrets。對企業級 fine-grained IAM 使用情境是退步。但 mcp-gatekeeper 主要定位為個人開發工具，此 trade-off 可接受。

**YAML 解析邊界情況**
→ YAML 有已知 edge cases（Norway problem、octal 數字等）。使用 `gopkg.in/yaml.v3` 並要求所有 value 為 string type，可避免大多數問題；敏感值（token、key）通常不會踩到這些邊界。

## Migration Plan

1. 使用者在各自 backend 建立名為 `mcp-gatekeeper` 的 secret，內容為 YAML bundle（包含原有所有 secret key-value）
2. 從 `mcp-settings.json` 移除 `--secret-source-name`（若使用 default）；或加上 `--secret-source-name=<name>` 指向自訂 bundle 名稱
3. 原有 backend 中的個別 secret entries 可安全刪除

Rollback：無自動 rollback 機制，若需回退須手動恢復原有個別 secret entries 並還原設定。

## Open Questions

- 無
