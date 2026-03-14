## 1. 專案初始化

- [x] 1.1 建立 Go module（`go mod init github.com/<user>/mcp-keeper`）
- [x] 1.2 建立主要目錄結構（`cmd/mcp-keeper/`, `internal/proxy/`, `internal/filter/`）
- [x] 1.3 建立 `main.go` 入口，確認可編譯

## 2. CLI 介面

- [x] 2.1 以 `flag` 套件實作 `--filter` flag（支援多次指定，收集為 string slice）
- [x] 2.2 以 `flag` 套件實作 `--exclude` flag（支援多次指定，收集為 string slice）
- [x] 2.3 `flag.Parse()` 後以 `flag.Args()` 取得子行程指令，無指令時印出 usage 並以非零退出碼結束

## 3. stdio proxy 核心

- [x] 3.1 以 `os/exec` 啟動子行程，設定 stdin pipe、stdout pipe，stderr 繼承（`cmd.Stderr = os.Stderr`）
- [x] 3.2 實作 `client → server` goroutine：`io.Copy(cmd.Stdin, os.Stdin)`
- [x] 3.3 實作 `server → client` goroutine：逐行讀取子行程 stdout（`bufio.Scanner`），交給過濾層處理後寫入 `os.Stdout`
- [x] 3.4 等待子行程結束，取得退出碼，以相同退出碼結束 `mcp-keeper`

## 4. tool-filtering 邏輯

- [x] 4.1 實作 JSON-RPC response 識別：判斷一行是否為 `tools/list` response（含 `result.tools` 陣列）
- [x] 4.2 實作 `path.Match` glob 比對函式，支援多個 pattern OR 邏輯
- [x] 4.3 實作 allowlist 過濾：有 `--filter` 時，只保留符合任一 pattern 的 tool
- [x] 4.4 實作 denylist 過濾：有 `--exclude` 時，移除符合任一 pattern 的 tool
- [x] 4.5 確認套用順序：先 allowlist 後 denylist
- [x] 4.6 無任何 flag 時，`tools/list` response 原封不動轉發

## 5. 單元測試

- [x] 5.1 測試 glob 比對函式（各種 pattern 組合）
- [x] 5.2 測試 `tools/list` response 識別（正確識別 / 非 tools/list 訊息不影響）
- [x] 5.3 測試 allowlist 過濾邏輯（單一 pattern、多 pattern、無 match）
- [x] 5.4 測試 denylist 過濾邏輯（單一 pattern、多 pattern）
- [x] 5.5 測試 allowlist + denylist 組合邏輯
- [x] 5.6 測試無 flag 時透明代理行為

## 6. 整合測試

- [x] 6.1 以 `mcp-server-time`（`uvx`）驗證透明代理（無 flag）
- [x] 6.2 以 `mcp-server-time` 驗證 `--filter="get_*"` 只回傳 `get_current_time`
- [x] 6.3 以 `mcp-server-filesystem`（`npx`）驗證 `--exclude="*_file"` 過濾多個 tool
- [x] 6.4 以 `server-memory`（`npx`）驗證 npx 型指令可正常代理

## 7. 發佈準備

- [x] 7.1 設定 `goreleaser`（或手動 Makefile），支援 linux/darwin/windows 三平台交叉編譯
- [x] 7.2 建立 GitHub Actions CI（build + test）
- [x] 7.3 建立 Homebrew tap formula（`homebrew/mcp-keeper.rb`，SHA256 已填入 v0.0.1）
- [x] 7.4 撰寫 README（安裝方式、使用範例、`--filter` / `--exclude` 說明）
- [ ] 7.5 建立 `chy168/homebrew-tap` GitHub repo（public），將 `homebrew/mcp-keeper.rb` 放到 `Formula/` 目錄並 push，讓使用者可以 `brew install chy168/tap/mcp-keeper`
