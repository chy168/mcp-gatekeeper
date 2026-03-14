## Why

MCP clients（如 Claude Desktop、Cursor）直接啟動 MCP server 時，無法控制哪些 tool 對 LLM 可見，容易造成 tool list 過長或暴露不必要的操作。`mcp-keeper` 作為一個輕量的 stdio proxy，插在 client 與 server 之間，讓使用者可以在不修改 MCP server 原始碼的情況下，過濾可用的 tool。

## What Changes

- 新增 `mcp-keeper` CLI 可執行檔
- 使用方式：將原本的啟動指令前綴加上 `mcp-keeper`
  - 原本：`uvx mcp-server-time`
  - 改為：`mcp-keeper uvx mcp-server-time`
- `mcp-keeper` 啟動後會：
  1. 將 `--filter` 以外的參數作為子行程指令執行（spawn subprocess）
  2. 透過 **stdio**（stdin/stdout）雙向代理所有 MCP JSON-RPC 訊息
  3. 攔截 `tools/list` 回應，以 glob pattern 過濾 tool 名稱後再回傳給 client
- 以 CLI flag 指定過濾規則：
  - `--filter=<glob>`：allowlist，只保留符合的 tool（例如 `--filter="get_*"`）
  - `--exclude=<glob>`：denylist，移除符合的 tool（例如 `--exclude="delete_*"`）
  - 兩者可同時使用，支援多次指定（OR 邏輯）
- 範例：`mcp-keeper --filter="get_*" uvx mcp-server-time` → client 只看到 `get_current_time`

## Capabilities

### New Capabilities

- `stdio-proxy`: 啟動子行程並雙向代理 MCP JSON-RPC stdio 訊息，對 client 完全透明
- `tool-filtering`: 攔截 `tools/list` response，依設定檔套用 allowlist 或 denylist 過濾 tool 清單

### Modified Capabilities

（無）

## Impact

- 新增一個獨立的 CLI binary `mcp-keeper`（初期以 Python 或 Node.js 實作）
- 不需修改任何現有 MCP server
- MCP client 設定檔（如 `claude_desktop_config.json`）只需在 command 前加上 `mcp-keeper`
- 依賴：Go stdlib（`os/exec`, `bufio`, `encoding/json`, `path`），無需額外依賴
