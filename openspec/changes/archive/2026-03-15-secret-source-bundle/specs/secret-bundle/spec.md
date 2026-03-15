## ADDED Requirements

### Requirement: Secret bundle 以 YAML 格式儲存
Secret bundle SHALL 為合法的 YAML 文件，top-level 結構為 mapping（key-value pairs），所有 value SHALL 為 scalar（string）型別。mcp-gatekeeper 以此格式作為唯一支援的 bundle 格式。

#### Scenario: 合法的 YAML bundle
- **WHEN** secret bundle 內容為合法 YAML mapping，所有 value 為 scalar
- **THEN** mcp-gatekeeper SHALL 成功解析並建立 key → value 對應表

#### Scenario: 支援多行純量
- **WHEN** bundle 中某 key 的值使用 YAML block scalar（`|`），例如 private key 或 JSON credential
- **THEN** mcp-gatekeeper SHALL 正確解析並保留換行符號，取得完整多行字串

---

### Requirement: YAML parse 失敗時 fail fast 並提供範例
當 bundle 內容無法解析為合法 YAML 時，mcp-gatekeeper SHALL 輸出包含失敗原因與正確格式範例的錯誤訊息至 stderr，並以非零退出碼結束，不啟動子行程。

#### Scenario: 非合法 YAML 內容
- **WHEN** backend 回傳的 bundle 內容不是合法 YAML
- **THEN** mcp-gatekeeper SHALL 輸出錯誤訊息，格式如下，並以非零退出碼結束：
  ```
  error: failed to parse secret bundle "<bundle-name>": <yaml error detail>

  Expected format:
    api_token: your-token-value
    private_key: |
      -----BEGIN RSA PRIVATE KEY-----
      ...
      -----END RSA PRIVATE KEY-----
  ```

---

### Requirement: Non-scalar value 時 fail fast 並提供範例
當 `{$secret.key}` 引用的 key 存在於 bundle 但其值不是 scalar（例如 nested map 或 sequence），mcp-gatekeeper SHALL 輸出含原因與範例的錯誤訊息，並以非零退出碼結束，不啟動子行程。

#### Scenario: Key 的值為 nested map
- **WHEN** bundle 中 key `foo` 的值為 YAML mapping
- **THEN** mcp-gatekeeper SHALL 輸出以下格式的錯誤訊息並以非零退出碼結束：
  ```
  error: secret key "foo" in bundle "<bundle-name>" is not a string value

  Expected format:
    foo: your-string-value
    multiline_value: |
      line one
      line two
  ```

#### Scenario: Key 的值為 sequence
- **WHEN** bundle 中 key `foo` 的值為 YAML sequence（list）
- **THEN** mcp-gatekeeper SHALL 輸出相同格式的錯誤訊息並以非零退出碼結束

---

### Requirement: Key 不存在時 fail fast 並列出可用 key
當 `{$secret.key}` 引用的 key 不存在於 bundle，mcp-gatekeeper SHALL 輸出錯誤訊息，列出 bundle 中現有的所有 key，並以非零退出碼結束，不啟動子行程。

#### Scenario: 引用不存在的 key
- **WHEN** `{$secret.api_token}` 但 bundle 中無 `api_token` 這個 key
- **THEN** mcp-gatekeeper SHALL 輸出以下格式的錯誤訊息並以非零退出碼結束：
  ```
  error: secret key "api_token" not found in bundle "<bundle-name>"

  Available keys: jira_token, gcp_sa_key
  ```

#### Scenario: Bundle 為空
- **WHEN** bundle 內容為空或無任何 key
- **THEN** mcp-gatekeeper SHALL 輸出錯誤訊息，Available keys 顯示為空，並以非零退出碼結束
