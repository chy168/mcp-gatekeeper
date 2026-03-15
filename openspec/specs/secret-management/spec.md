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

---

### Requirement: 指定 secret bundle 名稱
mcp-gatekeeper SHALL 接受 `--secret-source-name=<name>` flag，指定從 backend 取得的 bundle entry 名稱。若未指定，SHALL 使用 default 值 `mcp-gatekeeper`。

#### Scenario: 指定自訂 bundle 名稱
- **WHEN** 使用者執行 `mcp-gatekeeper --secret-source=gcp --secret-source-name=my-bundle ...`
- **THEN** mcp-gatekeeper SHALL 從 GCP Secret Manager 取得名為 `my-bundle` 的 secret 作為 bundle

#### Scenario: 未指定 bundle 名稱
- **WHEN** 使用者未指定 `--secret-source-name`
- **THEN** mcp-gatekeeper SHALL 使用 `mcp-gatekeeper` 作為 bundle 名稱

---

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

---

### Requirement: Fail fast — 任何 secret 解析失敗即中止
mcp-gatekeeper SHALL 在啟動子行程之前解析所有 `{$secret.*}` 引用。任何一個解析失敗，SHALL 輸出錯誤訊息至 stderr 並以非零退出碼結束，不進行部分注入。

#### Scenario: 所有 secret 解析成功
- **WHEN** 所有 `{$secret.*}` 引用均成功取得值
- **THEN** mcp-gatekeeper SHALL 繼續執行 secret 注入並啟動子行程

#### Scenario: 部分 secret 解析失敗
- **WHEN** 有任意一個 `{$secret.*}` 解析失敗
- **THEN** mcp-gatekeeper SHALL 輸出錯誤訊息並以非零退出碼結束，不啟動子行程，不進行任何部分注入
