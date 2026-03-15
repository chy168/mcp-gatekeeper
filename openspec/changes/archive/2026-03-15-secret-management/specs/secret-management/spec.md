## ADDED Requirements

### Requirement: 指定 secret backend
mcp-gatekeeper SHALL 接受 `--secret-source=<backend>` flag，支援值為 `gcp`、`aws`、`keychain`。每次執行只能指定一個 backend。

#### Scenario: 指定有效 backend
- **WHEN** 使用者執行 `mcp-gatekeeper --secret-source=gcp ...`
- **THEN** mcp-gatekeeper SHALL 使用 GCP Secret Manager 作為 secret 來源

#### Scenario: 指定無效 backend
- **WHEN** 使用者指定不支援的 backend（如 `--secret-source=vault`）
- **THEN** mcp-gatekeeper SHALL 輸出錯誤訊息並以非零退出碼結束，不啟動子行程

#### Scenario: 有 secret 引用但未指定 backend
- **WHEN** 任何 flag 中含有 `{$secret.*}` 但未指定 `--secret-source`
- **THEN** mcp-gatekeeper SHALL 輸出錯誤訊息並以非零退出碼結束，不啟動子行程

---

### Requirement: 從 GCP Secret Manager 取得 secret
當 `--secret-source=gcp` 時，mcp-gatekeeper SHALL 從 GCP Secret Manager 取得 secret，project 從環境變數 `GOOGLE_CLOUD_PROJECT` 讀取，永遠取 `latest` 版本，認證使用 Application Default Credentials（ADC）。

#### Scenario: 成功取得 secret
- **WHEN** `GOOGLE_CLOUD_PROJECT` 已設定，secret 名稱存在於 GCP Secret Manager
- **THEN** mcp-gatekeeper SHALL 取得該 secret 的 latest 版本內容

#### Scenario: 缺少 GOOGLE_CLOUD_PROJECT
- **WHEN** 環境變數 `GOOGLE_CLOUD_PROJECT` 未設定
- **THEN** mcp-gatekeeper SHALL 輸出錯誤訊息並以非零退出碼結束，不啟動子行程

#### Scenario: secret 不存在
- **WHEN** 指定的 secret 名稱在 GCP Secret Manager 中不存在
- **THEN** mcp-gatekeeper SHALL 輸出錯誤訊息並以非零退出碼結束，不啟動子行程

---

### Requirement: 從 AWS Secrets Manager 取得 secret
當 `--secret-source=aws` 時，mcp-gatekeeper SHALL 從 AWS Secrets Manager 取得 secret，region 從標準 AWS 環境變數（`AWS_DEFAULT_REGION` 或 `AWS_REGION`）讀取，認證使用標準 AWS credential chain。

#### Scenario: 成功取得 secret
- **WHEN** AWS region 與認證已正確設定，secret 名稱存在
- **THEN** mcp-gatekeeper SHALL 取得該 secret 的當前版本內容

#### Scenario: 缺少 region 設定
- **WHEN** `AWS_DEFAULT_REGION` 與 `AWS_REGION` 皆未設定
- **THEN** mcp-gatekeeper SHALL 輸出錯誤訊息並以非零退出碼結束，不啟動子行程

---

### Requirement: 從 OS Keychain 取得 secret
當 `--secret-source=keychain` 時，mcp-gatekeeper SHALL 使用 OS Keychain 取得 secret，service name 固定為 `"mcp-gatekeeper"`，account name 為 `{$secret.name}` 中的 `name`。支援平台：macOS Keychain、Linux Secret Service、Windows Credential Manager。

#### Scenario: 成功取得 secret
- **WHEN** OS Keychain 中存在 service=`mcp-gatekeeper`、account=`<name>` 的項目
- **THEN** mcp-gatekeeper SHALL 取得對應的 secret 值

#### Scenario: secret 不存在於 Keychain
- **WHEN** OS Keychain 中找不到對應項目
- **THEN** mcp-gatekeeper SHALL 輸出錯誤訊息並以非零退出碼結束，不啟動子行程

---

### Requirement: Fail fast — 任何 secret 解析失敗即中止
mcp-gatekeeper SHALL 在啟動子行程之前解析所有 `{$secret.*}` 引用。任何一個解析失敗，SHALL 輸出錯誤訊息至 stderr 並以非零退出碼結束，不進行部分注入。

#### Scenario: 所有 secret 解析成功
- **WHEN** 所有 `{$secret.*}` 引用均成功取得值
- **THEN** mcp-gatekeeper SHALL 繼續執行 secret 注入並啟動子行程

#### Scenario: 部分 secret 解析失敗
- **WHEN** 有任意一個 `{$secret.*}` 解析失敗
- **THEN** mcp-gatekeeper SHALL 輸出錯誤訊息並以非零退出碼結束，不啟動子行程，不進行任何部分注入
