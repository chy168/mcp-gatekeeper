## ADDED Requirements

### Requirement: 攔截 tools/list response
`mcp-keeper` SHALL 識別子行程 stdout 中的 `tools/list` JSON-RPC response，並在轉發前對其 `tools` 陣列套用過濾邏輯。

#### Scenario: 識別 tools/list response
- **WHEN** 子行程輸出含有 `"result": {"tools": [...]}` 且對應 `tools/list` 請求的 JSON-RPC response
- **THEN** `mcp-keeper` SHALL 對該訊息的 `tools` 陣列套用過濾後再轉發

#### Scenario: 非 tools/list 訊息不受影響
- **WHEN** 子行程輸出任何非 `tools/list` response 的訊息（如 `tools/call` response）
- **THEN** 該訊息 SHALL 原封不動轉發，不做任何修改

---

### Requirement: --filter allowlist 過濾
當指定 `--filter=<glob>` 時，`mcp-keeper` SHALL 僅保留 `tools` 陣列中名稱符合任一 pattern 的 tool。

#### Scenario: 單一 --filter pattern
- **WHEN** 使用者指定 `--filter="get_*"` 且 server 回傳 `[get_current_time, convert_time]`
- **THEN** 過濾後的 tools 陣列 SHALL 僅包含 `get_current_time`

#### Scenario: 多個 --filter pattern（OR 邏輯）
- **WHEN** 使用者指定 `--filter="get_*" --filter="convert_*"` 且 server 回傳 `[get_current_time, convert_time, delete_event]`
- **THEN** 過濾後的 tools 陣列 SHALL 包含 `get_current_time` 與 `convert_time`，不含 `delete_event`

#### Scenario: --filter 無符合項目
- **WHEN** 使用者指定 `--filter="write_*"` 且所有 tool 名稱均不符合
- **THEN** 過濾後的 tools 陣列 SHALL 為空陣列 `[]`

---

### Requirement: --exclude denylist 過濾
當指定 `--exclude=<glob>` 時，`mcp-keeper` SHALL 從 `tools` 陣列中移除名稱符合任一 pattern 的 tool。

#### Scenario: 單一 --exclude pattern
- **WHEN** 使用者指定 `--exclude="delete_*"` 且 server 回傳 `[read_file, write_file, delete_file]`
- **THEN** 過濾後的 tools 陣列 SHALL 包含 `read_file` 與 `write_file`，不含 `delete_file`

#### Scenario: 多個 --exclude pattern（OR 邏輯）
- **WHEN** 使用者指定 `--exclude="delete_*" --exclude="write_*"` 且 server 回傳 `[read_file, write_file, delete_file]`
- **THEN** 過濾後的 tools 陣列 SHALL 僅包含 `read_file`

---

### Requirement: --filter 與 --exclude 組合
當同時指定 `--filter` 與 `--exclude` 時，SHALL 先套用 allowlist 再套用 denylist。

#### Scenario: 先 allowlist 後 denylist
- **WHEN** 使用者指定 `--filter="*_file" --exclude="delete_*"` 且 server 回傳 `[read_file, write_file, delete_file, list_directory]`
- **THEN** 先保留 `[read_file, write_file, delete_file]`，再移除 `delete_file`，最終結果為 `[read_file, write_file]`

---

### Requirement: 無 flag 時透明代理
當未指定任何 `--filter` 或 `--exclude` 時，`mcp-keeper` SHALL 原封不動轉發 `tools/list` response。

#### Scenario: 無過濾 flag
- **WHEN** 使用者執行 `mcp-keeper uvx mcp-server-time`（無任何過濾 flag）
- **THEN** client 收到的 tools 陣列 SHALL 與 server 原始回傳完全相同
