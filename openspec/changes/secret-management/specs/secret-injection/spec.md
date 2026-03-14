## ADDED Requirements

### Requirement: 以 env var 注入 secret
mcp-gatekeeper SHALL 支援 `--env=VAR={$secret.name}` 語法，將解析後的 secret 值設為子行程的環境變數。`{$secret.name}` 可出現在 value 的任意位置（支援部分替換，如 `--env=PREFIX_{$secret.name}_SUFFIX=value` 不適用；value 部分整體替換）。

#### Scenario: 注入單一 env var
- **WHEN** 使用者指定 `--env=API_TOKEN={$secret.my_token}`
- **THEN** 子行程 SHALL 以環境變數 `API_TOKEN=<secret值>` 啟動

#### Scenario: 注入多個 env var
- **WHEN** 使用者指定多個 `--env` flag
- **THEN** 子行程 SHALL 以所有指定的環境變數啟動

#### Scenario: 不含 secret 引用的 --env
- **WHEN** `--env=FOO=bar`（不含 `{$secret.*}`）
- **THEN** 子行程 SHALL 以環境變數 `FOO=bar` 啟動，不需要 `--secret-source`

---

### Requirement: 以 command arg 注入 secret
mcp-gatekeeper SHALL 在傳遞子行程 args 之前，將所有 arg token 中的 `{$secret.name}` 替換為對應 secret 值。

#### Scenario: arg 中含有 secret 引用
- **WHEN** 子行程 args 中有 token `--token={$secret.github_token}`
- **THEN** 子行程 SHALL 以 `--token=<secret值>` 啟動

#### Scenario: 多個 arg token 含 secret 引用
- **WHEN** 多個 arg token 分別含有不同 `{$secret.*}` 引用
- **THEN** 每個 token 均 SHALL 完成替換後傳遞給子行程

---

### Requirement: 以 temp file 注入 secret（env var 模式）
mcp-gatekeeper SHALL 支援 `--file=VAR={$secret.name}` 語法：將 secret 值寫入隨機 temp file（權限 `0600`），並將該檔案路徑設為子行程的環境變數 `VAR`。子行程結束後 SHALL 刪除該 temp file。

#### Scenario: 成功建立 temp file 並注入路徑
- **WHEN** 使用者指定 `--file=GOOGLE_APPLICATION_CREDENTIALS={$secret.gcp_sa_key}`
- **THEN** mcp-gatekeeper SHALL 將 secret 內容寫入 temp file，子行程以 `GOOGLE_APPLICATION_CREDENTIALS=<temp路徑>` 啟動

#### Scenario: 子行程結束後清理 temp file
- **WHEN** 子行程正常或異常結束
- **THEN** mcp-gatekeeper SHALL 刪除所有由 `--file=VAR=...` 建立的 temp file

#### Scenario: temp file 權限
- **WHEN** temp file 建立時
- **THEN** 檔案權限 SHALL 為 `0600`（僅 owner 可讀寫）

---

### Requirement: 以 fixed path file 注入 secret
mcp-gatekeeper SHALL 支援 `--file=/path/to/file={$secret.name}` 語法（左側以 `/` 或 `~` 開頭）：將 secret 值寫入指定路徑，不設定環境變數。

#### Scenario: 成功寫入 fixed path
- **WHEN** 使用者指定 `--file=/tmp/creds.json={$secret.gcp_sa_key}`
- **THEN** mcp-gatekeeper SHALL 將 secret 內容寫入 `/tmp/creds.json`，子行程不注入額外環境變數

#### Scenario: fixed path 無寫入權限
- **WHEN** 指定路徑無法寫入
- **THEN** mcp-gatekeeper SHALL 輸出錯誤訊息並以非零退出碼結束，不啟動子行程

---

### Requirement: 不含 secret 引用時不需要 --secret-source
若所有 `--env`、`--file` 值及子行程 args 中均不含 `{$secret.*}`，mcp-gatekeeper SHALL 不要求 `--secret-source` flag，行為與現有版本相同。

#### Scenario: 無 secret 引用的正常執行
- **WHEN** 未使用任何 `{$secret.*}` 語法，也未指定 `--secret-source`
- **THEN** mcp-gatekeeper SHALL 正常啟動，行為不變
