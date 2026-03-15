## MODIFIED Requirements

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
