#!/usr/bin/env python3
# /// script
# requires-python = ">=3.12"
# ///
"""
Integration tests for mcp-keeper.

Run with:  uv run scripts/integration_test.py

Each test:
1. Builds mcp-keeper (once, via gvm go1.24)
2. Spawns: mcp-keeper [flags] <mcp-server-command>
3. Sends MCP initialize → initialized → tools/list via stdin
4. Reads the tools/list JSON-RPC response from stdout (with timeout)
5. Asserts the tool list matches expectations
"""

import json
import os
import shutil
import subprocess
import sys
import threading

# ── ANSI colours ──────────────────────────────────────────────────────────────
PASS = "\033[32mPASS\033[0m"
FAIL = "\033[31mFAIL\033[0m"
SKIP = "\033[33mSKIP\033[0m"

PROJECT_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
BINARY = os.path.join(PROJECT_ROOT, "bin", "mcp-keeper")

# ── MCP JSON-RPC handshake messages ──────────────────────────────────────────
INIT_REQUEST = json.dumps({
    "jsonrpc": "2.0",
    "id": 1,
    "method": "initialize",
    "params": {
        "protocolVersion": "2024-11-05",
        "capabilities": {},
        "clientInfo": {"name": "integration-test", "version": "0.1"},
    },
})
INITIALIZED_NOTIF = json.dumps({
    "jsonrpc": "2.0",
    "method": "notifications/initialized",
    "params": {},
})
TOOLS_LIST_REQUEST = json.dumps({
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/list",
    "params": {},
})
MCP_HANDSHAKE = (INIT_REQUEST + "\n" + INITIALIZED_NOTIF + "\n" + TOOLS_LIST_REQUEST + "\n").encode()


# ── Build ─────────────────────────────────────────────────────────────────────
def build():
    print("Building mcp-keeper...")
    os.makedirs(os.path.join(PROJECT_ROOT, "bin"), exist_ok=True)
    result = subprocess.run(
        "source ~/.gvm/scripts/gvm && gvm use go1.24 2>/dev/null && "
        f"go build -o {BINARY} ./cmd/mcp-keeper",
        shell=True,
        executable="/bin/bash",
        cwd=PROJECT_ROOT,
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        print(f"Build failed:\n{result.stderr}")
        sys.exit(1)
    print(f"  Built: {BINARY}\n")


# ── Core helper ───────────────────────────────────────────────────────────────
def list_tools(keeper_args: list[str], server_cmd: list[str], timeout: int = 30) -> list[str] | None:
    """
    Spawn mcp-keeper with given flags + server command, run the MCP handshake,
    and return the tool names from the tools/list response.
    Returns None if no valid response is received within `timeout` seconds.
    """
    cmd = [BINARY] + keeper_args + server_cmd
    proc = subprocess.Popen(
        cmd,
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.DEVNULL,
        cwd=PROJECT_ROOT,
    )

    found: list[str] | None = None

    def reader():
        nonlocal found
        try:
            for raw_line in proc.stdout:
                line = raw_line.strip()
                if not line:
                    continue
                try:
                    msg = json.loads(line)
                    result = msg.get("result")
                    if isinstance(result, dict) and "tools" in result:
                        found = [t["name"] for t in result["tools"]]
                        return
                except json.JSONDecodeError:
                    pass
        except Exception:
            pass

    t = threading.Thread(target=reader, daemon=True)

    try:
        proc.stdin.write(MCP_HANDSHAKE)
        proc.stdin.flush()
        proc.stdin.close()
    except BrokenPipeError:
        pass

    t.start()
    t.join(timeout=timeout)

    proc.terminate()
    try:
        proc.wait(timeout=5)
    except subprocess.TimeoutExpired:
        proc.kill()

    return found


# ── Test runner ───────────────────────────────────────────────────────────────
def run_test(
    name: str,
    keeper_args: list[str],
    server_cmd: list[str],
    check,
    timeout: int = 30,
) -> bool | None:
    flag_str = (" ".join(keeper_args) + " ") if keeper_args else ""
    cmd_str = " ".join(server_cmd)
    print(f"{'─' * 60}")
    print(f"  {name}")
    print(f"  $ mcp-keeper {flag_str}{cmd_str}")

    # Skip if the underlying binary isn't available
    if not shutil.which(server_cmd[0]):
        print(f"  {SKIP}: '{server_cmd[0]}' not found in PATH")
        return None

    try:
        tools = list_tools(keeper_args, server_cmd, timeout=timeout)
        if tools is None:
            print(f"  {FAIL}: No tools/list response received within {timeout}s")
            return False
        print(f"  Tools ({len(tools)}): {tools}")
        ok, msg = check(tools)
        print(f"  {PASS if ok else FAIL}: {msg}")
        return ok
    except Exception as exc:
        print(f"  {FAIL}: {exc}")
        return False


# ── Test cases ────────────────────────────────────────────────────────────────
def main():
    os.chdir(PROJECT_ROOT)
    build()

    results = []

    # 6.1 ── Transparent proxy: no flags, all tools pass through
    results.append(run_test(
        "6.1  transparent proxy — uvx mcp-server-time, no flags",
        keeper_args=[],
        server_cmd=["uvx", "mcp-server-time"],
        check=lambda tools: (
            "get_current_time" in tools and "convert_time" in tools,
            f"expected both get_current_time & convert_time, got {tools}",
        ),
    ))

    # 6.2 ── Allowlist: --filter=get_* keeps only get_current_time
    results.append(run_test(
        "6.2  --filter=get_*  — uvx mcp-server-time",
        keeper_args=["--filter=get_*"],
        server_cmd=["uvx", "mcp-server-time"],
        check=lambda tools: (
            tools == ["get_current_time"],
            f"expected ['get_current_time'], got {tools}",
        ),
    ))

    # 6.3 ── Denylist: --exclude=*_file removes *_file tools
    #         filesystem server is npm: npx @modelcontextprotocol/server-filesystem
    results.append(run_test(
        "6.3  --exclude=*_file  — npx @modelcontextprotocol/server-filesystem /tmp",
        keeper_args=["--exclude=*_file"],
        server_cmd=["npx", "-y", "@modelcontextprotocol/server-filesystem", "/tmp"],
        check=lambda tools: (
            len(tools) > 0 and not any(t.endswith("_file") for t in tools),
            (
                f"no *_file tools expected in result; got {tools}"
                if any(t.endswith("_file") for t in tools)
                else f"OK — remaining tools: {tools}"
            ),
        ),
        timeout=60,  # npx may need extra time to download
    ))

    # 6.4 ── npx server-memory: transparent proxy
    results.append(run_test(
        "6.4  transparent proxy — npx @modelcontextprotocol/server-memory",
        keeper_args=[],
        server_cmd=["npx", "-y", "@modelcontextprotocol/server-memory"],
        check=lambda tools: (
            len(tools) > 0,
            f"expected ≥1 tool, got {tools}",
        ),
        timeout=60,
    ))

    # ── Summary ───────────────────────────────────────────────────────────────
    print(f"\n{'═' * 60}")
    skipped = sum(1 for r in results if r is None)
    definitive = [r for r in results if r is not None]
    passed = sum(1 for r in definitive if r)
    total = len(definitive)
    parts = [f"{passed}/{total} passed"]
    if skipped:
        parts.append(f"{skipped} skipped")
    print("Results: " + ", ".join(parts))
    sys.exit(0 if passed == total else 1)


if __name__ == "__main__":
    main()
