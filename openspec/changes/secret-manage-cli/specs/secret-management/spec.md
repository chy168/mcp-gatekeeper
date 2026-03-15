## MODIFIED Requirements

### Requirement: 指定 secret backend
mcp-gatekeeper SHALL 接受 `--secret-source=<backend>` flag，支援值為 `gcp`、`aws`、`keychain`。每次執行只能指定一個 backend。此 flag 同樣適用於 `mcp-gatekeeper-secret` binary 的所有 subcommands。

#### Scenario: 指定有效 backend
- **WHEN** 使用者執行 `mcp-gatekeeper --secret-source=gcp ...` 或 `mcp-gatekeeper-secret --secret-source=gcp ...`
- **THEN** 該 binary SHALL 使用 GCP Secret Manager 作為 secret 來源

#### Scenario: 指定無效 backend
- **WHEN** 使用者指定不支援的 backend（如 `--secret-source=vault`）
- **THEN** 該 binary SHALL 輸出錯誤訊息並以非零退出碼結束，不執行任何操作

#### Scenario: 有 secret 引用但未指定 backend（mcp-gatekeeper）
- **WHEN** `mcp-gatekeeper` 任何 flag 中含有 `{$secret.*}` 但未指定 `--secret-source`
- **THEN** mcp-gatekeeper SHALL 輸出錯誤訊息並以非零退出碼結束，不啟動子行程

#### Scenario: mcp-gatekeeper-secret 未指定 backend
- **WHEN** `mcp-gatekeeper-secret` 任意 subcommand 未提供 `--secret-source`
- **THEN** mcp-gatekeeper-secret SHALL 輸出錯誤訊息並以非零退出碼結束
