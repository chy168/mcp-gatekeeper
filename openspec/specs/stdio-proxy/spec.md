## ADDED Requirements

### Requirement: CLI 接受子行程指令
`mcp-gatekeeper` SHALL 將所有非自身 flag（`--allow`、`--exclude`、`--secret-source`、`--env`、`--file`）之後的參數視為子行程指令及其參數，並在完成 secret 注入後執行。

#### Scenario: 基本啟動
- **WHEN** 使用者執行 `mcp-gatekeeper uvx mcp-server-time`
- **THEN** `mcp-gatekeeper` 啟動 `uvx mcp-server-time` 作為子行程

#### Scenario: 帶參數的子行程指令
- **WHEN** 使用者執行 `mcp-gatekeeper uvx mcp-server-filesystem /tmp`
- **THEN** `mcp-gatekeeper` 以 `uvx mcp-server-filesystem /tmp` 啟動子行程，參數完整傳遞

#### Scenario: 無子行程指令
- **WHEN** 使用者執行 `mcp-gatekeeper` 而不帶任何子行程指令
- **THEN** `mcp-gatekeeper` 輸出使用說明並以非零退出碼結束

#### Scenario: 含 secret 引用的子行程指令
- **WHEN** 使用者執行 `mcp-gatekeeper --secret-source=gcp --env=TOKEN={$secret.my_token} uvx mcp-server-foo`
- **THEN** `mcp-gatekeeper` SHALL 先解析所有 secret，注入完成後才啟動子行程

---

### Requirement: 雙向 stdio 代理
`mcp-keeper` SHALL 將 client 的 stdin 完整轉發至子行程的 stdin，並將子行程的 stdout 完整轉發至 client 的 stdout，除非訊息需被過濾處理。

#### Scenario: client 送出訊息
- **WHEN** client 透過 stdin 傳送任意 JSON-RPC 訊息
- **THEN** 該訊息 SHALL 原封不動地寫入子行程 stdin

#### Scenario: server 回傳非 tools/list 訊息
- **WHEN** 子行程 stdout 輸出任意非 `tools/list` response 的 JSON-RPC 訊息
- **THEN** 該訊息 SHALL 原封不動地寫入 `mcp-keeper` 的 stdout

---

### Requirement: stderr 繼承
`mcp-keeper` SHALL 將子行程的 stderr 直接繼承（不代理、不修改）。

#### Scenario: 子行程輸出錯誤訊息
- **WHEN** 子行程向 stderr 輸出任意內容
- **THEN** 該內容 SHALL 直接出現在 `mcp-keeper` 的 stderr，不做任何處理

---

### Requirement: 子行程結束時退出
`mcp-keeper` SHALL 在子行程結束後，以相同的退出碼結束自身行程。

#### Scenario: 子行程正常結束
- **WHEN** 子行程以退出碼 0 結束
- **THEN** `mcp-keeper` SHALL 以退出碼 0 結束

#### Scenario: 子行程異常結束
- **WHEN** 子行程以非零退出碼結束
- **THEN** `mcp-keeper` SHALL 以相同的非零退出碼結束
