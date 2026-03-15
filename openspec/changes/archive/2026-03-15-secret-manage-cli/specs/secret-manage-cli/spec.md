## ADDED Requirements

### Requirement: list — 列出 bundle 所有 key
`mcp-gatekeeper-secret list` SHALL 從指定 backend 取得 bundle，列出所有 top-level key，每行一個，不顯示 value。

#### Scenario: 成功列出 keys
- **WHEN** bundle 存在且包含一或多個 key
- **THEN** 輸出每個 key 名稱，每行一個，至 stdout

#### Scenario: Bundle 為空
- **WHEN** bundle 存在但無任何 key
- **THEN** 輸出空內容（無任何行），以零退出碼結束

#### Scenario: Bundle 不存在
- **WHEN** 指定的 bundle 名稱在 backend 中不存在
- **THEN** 輸出錯誤訊息至 stderr 並以非零退出碼結束

---

### Requirement: get — 取得指定 key 的 value
`mcp-gatekeeper-secret get <key>` SHALL 從 bundle 取得指定 key 的 value。預設顯示遮蔽值（`****`），加 `--reveal` 顯示明文。

#### Scenario: 成功取得 key（預設遮蔽）
- **WHEN** bundle 中存在指定 key，未加 `--reveal`
- **THEN** 輸出 `<key>: ****` 至 stdout

#### Scenario: 成功取得 key（--reveal）
- **WHEN** bundle 中存在指定 key，加上 `--reveal`
- **THEN** 輸出 `<key>: <value>` 至 stdout

#### Scenario: Key 不存在
- **WHEN** bundle 中不存在指定 key
- **THEN** 輸出含可用 key 列表的錯誤訊息至 stderr 並以非零退出碼結束

---

### Requirement: set — 新增或更新 bundle 中的 key
`mcp-gatekeeper-secret set <key> <value>` SHALL 在 bundle 中新增或更新指定 key 的 value，並將修改後的 bundle 寫回 backend。`--from-file=<path>` 可替代 positional value arg，用於多行內容。

#### Scenario: 以 positional arg 設定 value
- **WHEN** 使用者執行 `mcp-gatekeeper-secret set api_token abc123`
- **THEN** bundle 中 `api_token` 的值更新為 `abc123`，修改後 bundle 寫回 backend，輸出確認訊息至 stdout

#### Scenario: 以 --from-file 設定 value
- **WHEN** 使用者執行 `mcp-gatekeeper-secret set gcp_sa_key --from-file=./sa.json`
- **THEN** 讀取 `./sa.json` 全部內容，bundle 中 `gcp_sa_key` 的值更新為該內容，修改後 bundle 寫回 backend

#### Scenario: --from-file 優先於 positional arg
- **WHEN** 使用者同時提供 positional arg 與 `--from-file`
- **THEN** 使用 `--from-file` 的內容，忽略 positional arg

#### Scenario: 未提供 value 來源
- **WHEN** 使用者未提供 positional arg 也未指定 `--from-file`
- **THEN** 輸出錯誤訊息至 stderr 並以非零退出碼結束，不修改 bundle

#### Scenario: --from-file 指定的檔案不存在
- **WHEN** `--from-file` 指定路徑不存在
- **THEN** 輸出錯誤訊息至 stderr 並以非零退出碼結束，不修改 bundle

#### Scenario: Bundle 不存在時自動建立
- **WHEN** backend 中尚無對應 bundle，使用者執行 `set`
- **THEN** mcp-gatekeeper-secret SHALL 建立新 bundle（僅含該 key），寫入 backend

---

### Requirement: delete — 從 bundle 移除指定 key
`mcp-gatekeeper-secret delete <key>` SHALL 從 bundle 移除指定 key，將修改後的 bundle 寫回 backend。不刪除 backend 中的 bundle entry 本身。

#### Scenario: 成功刪除 key
- **WHEN** bundle 中存在指定 key
- **THEN** 該 key 從 bundle 移除，修改後 bundle 寫回 backend，輸出確認訊息至 stdout

#### Scenario: Key 不存在
- **WHEN** bundle 中不存在指定 key
- **THEN** 輸出錯誤訊息至 stderr 並以非零退出碼結束，不修改 bundle

---

### Requirement: --secret-source 與 --secret-source-name 為必要 global flags
所有 subcommand 均 SHALL 要求 `--secret-source` flag。`--secret-source-name` 為選填，default `mcp-gatekeeper`。

#### Scenario: 未指定 --secret-source
- **WHEN** 任意 subcommand 執行時未提供 `--secret-source`
- **THEN** 輸出錯誤訊息至 stderr 並以非零退出碼結束
