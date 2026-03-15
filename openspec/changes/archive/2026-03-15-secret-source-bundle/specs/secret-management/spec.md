## ADDED Requirements

### Requirement: 指定 secret bundle 名稱
mcp-gatekeeper SHALL 接受 `--secret-source-name=<name>` flag，指定從 backend 取得的 bundle entry 名稱。若未指定，SHALL 使用 default 值 `mcp-gatekeeper`。

#### Scenario: 指定自訂 bundle 名稱
- **WHEN** 使用者執行 `mcp-gatekeeper --secret-source=gcp --secret-source-name=my-bundle ...`
- **THEN** mcp-gatekeeper SHALL 從 GCP Secret Manager 取得名為 `my-bundle` 的 secret 作為 bundle

#### Scenario: 未指定 bundle 名稱
- **WHEN** 使用者未指定 `--secret-source-name`
- **THEN** mcp-gatekeeper SHALL 使用 `mcp-gatekeeper` 作為 bundle 名稱

---

## MODIFIED Requirements

### Requirement: 從 GCP Secret Manager 取得 secret
當 `--secret-source=gcp` 時，mcp-gatekeeper SHALL 從 GCP Secret Manager 取得名為 `--secret-source-name`（default: `mcp-gatekeeper`）的 secret，project 從環境變數 `GOOGLE_CLOUD_PROJECT` 讀取，永遠取 `latest` 版本，認證使用 Application Default Credentials（ADC）。取得的內容作為 YAML bundle 解析，各 `{$secret.key}` 從 bundle 中 lookup。

#### Scenario: 成功取得 bundle
- **WHEN** `GOOGLE_CLOUD_PROJECT` 已設定，bundle 名稱存在於 GCP Secret Manager，內容為合法 YAML
- **THEN** mcp-gatekeeper SHALL 解析 bundle 並以其內容解析所有 `{$secret.key}` 引用

#### Scenario: 缺少 GOOGLE_CLOUD_PROJECT
- **WHEN** 環境變數 `GOOGLE_CLOUD_PROJECT` 未設定
- **THEN** mcp-gatekeeper SHALL 輸出錯誤訊息並以非零退出碼結束，不啟動子行程

#### Scenario: bundle 不存在
- **WHEN** 指定的 bundle 名稱在 GCP Secret Manager 中不存在
- **THEN** mcp-gatekeeper SHALL 輸出錯誤訊息並以非零退出碼結束，不啟動子行程

---

### Requirement: 從 AWS Secrets Manager 取得 secret
當 `--secret-source=aws` 時，mcp-gatekeeper SHALL 從 AWS Secrets Manager 取得名為 `--secret-source-name`（default: `mcp-gatekeeper`）的 secret，region 從標準 AWS 環境變數（`AWS_DEFAULT_REGION` 或 `AWS_REGION`）讀取，認證使用標準 AWS credential chain。取得的內容作為 YAML bundle 解析。

#### Scenario: 成功取得 bundle
- **WHEN** AWS region 與認證已正確設定，bundle 名稱存在，內容為合法 YAML
- **THEN** mcp-gatekeeper SHALL 解析 bundle 並以其內容解析所有 `{$secret.key}` 引用

#### Scenario: 缺少 region 設定
- **WHEN** `AWS_DEFAULT_REGION` 與 `AWS_REGION` 皆未設定
- **THEN** mcp-gatekeeper SHALL 輸出錯誤訊息並以非零退出碼結束，不啟動子行程

#### Scenario: bundle 不存在
- **WHEN** 指定的 bundle 名稱在 AWS Secrets Manager 中不存在
- **THEN** mcp-gatekeeper SHALL 輸出錯誤訊息並以非零退出碼結束，不啟動子行程

---

### Requirement: 從 OS Keychain 取得 secret
當 `--secret-source=keychain` 時，mcp-gatekeeper SHALL 使用 OS Keychain 取得 secret bundle，service name 固定為 `"mcp-gatekeeper"`，account name 為 `--secret-source-name` 的值（default: `mcp-gatekeeper`）。取得的內容作為 YAML bundle 解析。支援平台：macOS Keychain、Linux Secret Service、Windows Credential Manager。

#### Scenario: 成功取得 bundle
- **WHEN** OS Keychain 中存在 service=`mcp-gatekeeper`、account=`<bundle-name>` 的項目，內容為合法 YAML
- **THEN** mcp-gatekeeper SHALL 解析 bundle 並以其內容解析所有 `{$secret.key}` 引用

#### Scenario: bundle 不存在於 Keychain
- **WHEN** OS Keychain 中找不到對應的 bundle 項目
- **THEN** mcp-gatekeeper SHALL 輸出錯誤訊息並以非零退出碼結束，不啟動子行程
