### Requirement: Backend interface 支援寫入
`Backend` interface SHALL 新增 `Set(ctx context.Context, name, value string) error`，語意為 upsert：secret 不存在時建立，存在時更新為新值。

#### Scenario: 寫入新 secret
- **WHEN** backend 中不存在名為 `name` 的 secret，呼叫 `Set`
- **THEN** backend SHALL 建立該 secret 並儲存 `value`

#### Scenario: 更新既有 secret
- **WHEN** backend 中已存在名為 `name` 的 secret，呼叫 `Set`
- **THEN** backend SHALL 將該 secret 更新為新的 `value`

---

### Requirement: GCP backend 實作 Set
GCP backend 的 `Set` SHALL 使用 `AddSecretVersion` 新增版本。若 secret 不存在，SHALL 先執行 `CreateSecret` 再 `AddSecretVersion`。

#### Scenario: Secret 已存在，新增版本
- **WHEN** GCP Secret Manager 中已存在指定 secret，呼叫 `Set`
- **THEN** 以新值新增一個 secret version（`AddSecretVersion`）

#### Scenario: Secret 不存在，自動建立
- **WHEN** GCP Secret Manager 中不存在指定 secret，呼叫 `Set`
- **THEN** 先建立 secret（`CreateSecret`），再新增版本（`AddSecretVersion`）

#### Scenario: 缺少必要 IAM 權限
- **WHEN** ADC 缺少 `secretmanager.secretVersions.add` 或 `secretmanager.secrets.create` 權限
- **THEN** `Set` SHALL 回傳含清楚描述的 error，不靜默失敗

---

### Requirement: AWS backend 實作 Set
AWS backend 的 `Set` SHALL 使用 `PutSecretValue` 更新值。若 secret 不存在，SHALL 先執行 `CreateSecret`。

#### Scenario: Secret 已存在，更新值
- **WHEN** AWS Secrets Manager 中已存在指定 secret，呼叫 `Set`
- **THEN** 使用 `PutSecretValue` 更新 SecretString

#### Scenario: Secret 不存在，自動建立
- **WHEN** AWS Secrets Manager 中不存在指定 secret，呼叫 `Set`
- **THEN** 先執行 `CreateSecret`，再執行 `PutSecretValue`

---

### Requirement: Keychain backend 實作 Set
Keychain backend 的 `Set` SHALL 使用 `keyring.Set(service, account, value)` 寫入，service name 固定為 `"mcp-gatekeeper"`，account name 為 `name` 參數。

#### Scenario: 成功寫入 Keychain
- **WHEN** 呼叫 `Set`，OS Keychain 可正常存取
- **THEN** `keyring.Set` 寫入指定 account 的值（upsert）

#### Scenario: Keychain 無法存取
- **WHEN** OS Keychain 不可用（如 headless Linux 無 Secret Service daemon）
- **THEN** `Set` SHALL 回傳含清楚描述的 error
