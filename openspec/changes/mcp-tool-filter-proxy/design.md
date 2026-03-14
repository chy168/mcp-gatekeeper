## Context

MCP client（Claude Desktop、Cursor 等）透過設定檔指定啟動指令來啟動 MCP server，雙方以 stdio（stdin/stdout）交換 JSON-RPC 2.0 訊息。目前 client 看到的 tool 清單完全由 server 決定，無法在 client 端或中介層過濾。

`mcp-keeper` 插入 client 與 server 之間，成為一個透明的 stdio proxy，僅對 `tools/list` 回應動手腳，其他訊息原封不動轉發。

## Goals / Non-Goals

**Goals:**
- 作為 stdio proxy 啟動並代理任意 stdio-based MCP server 子行程
- 支援 `--filter=<glob>` CLI flag，以 glob pattern 對 `tools/list` 的 tool 名稱做 allowlist 過濾
- 對 client 完全透明（除了被過濾的 tool 外，行為與直接連 server 相同）

**Non-Goals:**
- 僅支援 stdio transport；不支援 SSE、HTTP、WebSocket 等其他 transport
- 不支援同時代理多個 MCP server
- 不做 tool 參數的過濾或修改
- 不持久化設定（設定皆透過 CLI flag 傳入）

## Decisions

### D1: 實作語言 — Go

**選擇**: Go（標準函式庫，無需額外依賴，編譯為 single binary）

**理由**: 編譯產物為單一 binary，無 runtime 依賴，可直接放入 `/usr/local/bin`，在 Claude Desktop 等 GUI 工具的乾淨 PATH 環境下最可靠；天生的 goroutine 模型非常適合雙向 stdio proxy；可輕鬆發佈到 Homebrew、Scoop、winget 等套件管理工具。

**替代方案**: Python（需要 runtime，PATH 在 GUI 環境不穩定）、Node.js（同樣需要 runtime）。

---

### D2: Proxy 架構 — goroutine 雙向管道

```
MCP Client (Claude Desktop)
    │ stdin / stdout
    ▼
mcp-keeper (this process)
    │ subprocess stdin / stdout
    ▼
MCP Server (uvx mcp-server-time)
```

用 `os/exec` 啟動子行程，再用兩個 goroutine 分別做：
- `client → server`：從 `os.Stdin` 讀取，原封不動寫入子行程 stdin
- `server → client`：從子行程 stdout 讀取，若為 `tools/list` response 則過濾後再寫入 `os.Stdout`

**理由**: goroutine 天生並發，兩個方向互不阻塞；`io.Copy` 處理透明轉發，`bufio.Scanner` 處理需過濾的方向。

---

### D3: 訊息邊界 — newline-delimited JSON

MCP stdio transport 以 `\n` 作為訊息分隔符（每條 JSON-RPC 訊息獨立一行）。

**選擇**: 以行為單位讀取（`bufio.Scanner`），`encoding/json` parse，判斷是否為 `tools/list` response 後決定是否修改。

---

### D4: Tool 過濾邏輯 — path.Match glob，支援 allowlist 與 denylist

**選擇**: 使用 Go stdlib `path.Match(pattern, toolName)` 做 glob 比對，提供兩個互補的 flag：

| Flag | 語意 | 範例 |
|---|---|---|
| `--filter=<glob>` | allowlist：只保留符合的 tool | `--filter="get_*"` |
| `--exclude=<glob>` | denylist：移除符合的 tool | `--exclude="delete_*"` |

**行為規則**（依序套用）:
1. 未指定任何 flag：所有 tool 通過（透明模式）
2. 有 `--filter`：先以 allowlist 縮減清單（多個 pattern 為 OR）
3. 有 `--exclude`：再從剩餘清單移除符合的 tool（多個 pattern 為 OR）
4. 兩者可同時使用

---

### D5: CLI 介面 — flag 套件

```
mcp-keeper [--filter=<glob>]... [--exclude=<glob>]... <command> [args...]
```

使用 Go stdlib `flag` 套件，`flag.Parse()` 後以 `flag.Args()` 取得子行程指令及其參數。

## Testing Strategy

**兩層測試：**

```
unit test      → mock subprocess stdout，直接測 JSON 過濾邏輯，無外部依賴
integration    → 真實啟動 MCP server，end-to-end 驗證 stdio proxy 行為
```

**Integration test 使用的 server：**

| Server | 啟動方式 | 工具數 | 測試重點 |
|---|---|---|---|
| `mcp-server-time` | `uvx mcp-server-time` | 2 | 基本 happy path、`--filter="get_*"` |
| `mcp-server-filesystem` | `uvx mcp-server-filesystem <path>` | 8+ | 多 tool 過濾、安全情境（擋 write/delete） |
| `server-memory` | `npx @modelcontextprotocol/server-memory` | 5 | npx 型指令、不同命名風格 |

## Risks / Trade-offs

- **JSON 訊息橫跨多行**: 若有 MCP server 輸出非 newline-delimited 的 JSON，目前解析會失敗 → 僅支援符合 MCP stdio spec 的 server。
- **子行程 stderr**: 子行程的 stderr 直接繼承（不代理），錯誤訊息會出現在 client 的 log 中，行為與直接啟動 server 相同，可接受。

## Migration Plan

1. 安裝：`brew install <tap>/mcp-keeper`，或直接下載 GitHub Releases binary
2. 修改 MCP client 設定，在原有 command 前加上 `mcp-keeper`（及 `--filter` 選項）
3. Rollback：移除 `mcp-keeper` 前綴即恢復原狀

## Open Questions

（無）
