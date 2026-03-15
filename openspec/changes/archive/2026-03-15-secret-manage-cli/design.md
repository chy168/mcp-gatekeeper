## Context

`internal/secret` 目前只有 `Backend.Get`。`mcp-gatekeeper-secret` 的 `set` / `delete` 需要將修改後的 YAML bundle 寫回 backend，因此需擴充 interface 加入 `Set`。

三個 backend 的寫入語意各異：
- **GCP**：需先確認 secret 是否存在（`AddSecretVersion`），若不存在則先 `CreateSecret` 再 `AddSecretVersion`
- **AWS**：`PutSecretValue`（secret 不存在時需先 `CreateSecret`）
- **Keychain**：`keyring.Set(service, account, value)`（`go-keyring` 已支援 upsert）

`mcp-gatekeeper-secret` 是薄的 CLI wrapper，所有 bundle 邏輯（parse、lookup、modify、serialize）集中在 `internal/secret/bundle.go`（由 `secret-source-bundle` change 建立），`main.go` 只負責 flag parsing 與 subcommand dispatch。

## Goals / Non-Goals

**Goals:**
- `Backend` interface 新增 `Set(ctx, name, value string) error`
- 三個 backend 各自實作 `Set`（upsert 語意：不存在則建立，存在則更新）
- `mcp-gatekeeper-secret` binary 提供 `list`、`get`、`set`、`delete` subcommands
- `get` 預設遮蔽 value，`--reveal` 顯示明文
- `set` 支援直接傳值（arg）與 `--from-file=<path>`（多行內容）

**Non-Goals:**
- 不提供 bundle rename / copy 功能
- 不支援 backend 層的 secret 刪除（只刪除 bundle 內的 key，bundle entry 本身保留）
- 不提供互動式（TUI）介面

## Decisions

### 決策一：`delete` 只刪 bundle 內的 key，不刪 backend entry

`delete <key>` 從 YAML bundle 移除該 key 後寫回，bundle entry 本身仍存在於 backend。

**理由**：刪除 backend entry 是破壞性且不可逆的操作，超出 key-level 管理的範圍；使用者若需刪除整個 bundle，應直接用各 backend 原生工具。

---

### 決策二：Set 語意為 upsert

`Backend.Set` 在 secret 不存在時自動建立，存在時更新（新版本）。

**理由**：使用者首次建立 bundle 時不需要額外的 `init` 指令，`set` 一個 key 即可完成 bundle 初始化。

---

### 決策三：CLI 結構 — cobra + PersistentFlags

使用 cobra 管理 subcommand dispatch：

```
rootCmd  (mcp-gatekeeper-secret)
  PersistentFlags: --secret-source, --secret-source-name

  ├── list
  ├── get <key>      [--reveal]
  ├── set <key> [value]  [--from-file=<path>]
  └── delete <key>
```

`--secret-source` / `--secret-source-name` 定義在 root command 的 `PersistentFlags()`，自動繼承至所有 subcommand。`--reveal` / `--from-file` 定義在各自的 subcommand local flags。

**理由**：`mcp-gatekeeper-secret` 是純 subcommand CLI，沒有 subprocess 需求，正是 cobra 設計的使用情境。cobra 的 `PersistentFlags` 天然解決 global flag 繼承問題，不需手動 dispatch 或 flag forwarding。`mcp-gatekeeper` 使用 `pflag` 是因為需要 `SetInterspersed(false)` 捕捉 subprocess args，兩者情境不同。

---

### 決策四：`set` 的 value 來源優先序

1. `--from-file=<path>`（若指定，忽略 positional arg）
2. positional arg（`mcp-gatekeeper-secret set <key> <value>`）
3. 兩者皆無 → 錯誤

**理由**：多行內容（private key、JSON）無法舒適地傳 positional arg；`--from-file` 是明確的多行 UX，兩者並存讓單行值更方便。

---

### 決策五：GCP Set 實作使用 AddSecretVersion

對 GCP，`Set` 永遠新增一個版本（`AddSecretVersion`），若 secret 不存在則先 `CreateSecret`。不刪除舊版本。

**理由**：GCP Secret Manager 版本不可變，符合平台設計；版本管理（rotation、cleanup）由使用者自行處理。

## Risks / Trade-offs

**GCP Set 需要額外 IAM 權限**
→ `roles/secretmanager.secretVersionAdder`（現有 `secretAccessor` 只夠 Get）。需在文件中說明。

**Keychain 在 Linux headless 環境不可用**
→ 繼承自現有 `Get` 的限制，`Set` 同樣只在有 Secret Service daemon 的環境運作。文件已有說明。

**`--from-file` 讀取大檔案**
→ 目前不限制檔案大小，使用者若誤傳大型檔案會整份存入 backend。此為 edge case，暫不處理。

## Open Questions

- 無
