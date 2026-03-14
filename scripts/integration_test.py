#!/usr/bin/env python3
# /// script
# requires-python = ">=3.12"
# ///
"""
Integration tests for mcp-keeper.

Run with:  uv run scripts/integration_test.py

Tests cover:
- tools/list filtering (allowlist, denylist, transparent)
- tools/call end-to-end: tools are actually invocable through the proxy
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


# ── MCP session helper ────────────────────────────────────────────────────────
def mcp_session(
    keeper_args: list[str],
    server_cmd: list[str],
    requests: list[dict],
    want_ids: set[int],
    timeout: int = 30,
) -> dict[int, dict]:
    """
    Run a full MCP session through mcp-keeper.

    Sends: initialize → initialized → *requests (in order)
    Collects responses whose 'id' is in want_ids.
    Returns {id: response_msg} for each collected response.
    """
    messages = [
        {"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {
            "protocolVersion": "2024-11-05",
            "capabilities": {},
            "clientInfo": {"name": "integration-test", "version": "0.1"},
        }},
        {"jsonrpc": "2.0", "method": "notifications/initialized", "params": {}},
        *requests,
    ]
    payload = ("\n".join(json.dumps(m) for m in messages) + "\n").encode()

    cmd = [BINARY] + keeper_args + server_cmd
    proc = subprocess.Popen(
        cmd,
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.DEVNULL,
        cwd=PROJECT_ROOT,
    )

    collected: dict[int, dict] = {}

    def reader():
        try:
            for raw_line in proc.stdout:
                line = raw_line.strip()
                if not line:
                    continue
                try:
                    msg = json.loads(line)
                    msg_id = msg.get("id")
                    if msg_id in want_ids:
                        collected[msg_id] = msg
                        if collected.keys() == want_ids:
                            return
                except json.JSONDecodeError:
                    pass
        except Exception:
            pass

    t = threading.Thread(target=reader, daemon=True)
    try:
        proc.stdin.write(payload)
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

    return collected


# ── Convenience wrappers ──────────────────────────────────────────────────────
def list_tools(keeper_args, server_cmd, timeout=30) -> list[str] | None:
    resp = mcp_session(
        keeper_args, server_cmd,
        requests=[{"jsonrpc": "2.0", "id": 2, "method": "tools/list", "params": {}}],
        want_ids={2},
        timeout=timeout,
    )
    msg = resp.get(2)
    if not msg:
        return None
    return [t["name"] for t in msg.get("result", {}).get("tools", [])]


def call_tool(keeper_args, server_cmd, tool_name, arguments, timeout=30) -> dict | None:
    """Returns the result dict from tools/call response, or None on timeout."""
    resp = mcp_session(
        keeper_args, server_cmd,
        requests=[
            {"jsonrpc": "2.0", "id": 2, "method": "tools/list", "params": {}},
            {"jsonrpc": "2.0", "id": 3, "method": "tools/call",
             "params": {"name": tool_name, "arguments": arguments}},
        ],
        want_ids={3},
        timeout=timeout,
    )
    msg = resp.get(3)
    if not msg:
        return None
    return msg.get("result")


# ── Test runner ───────────────────────────────────────────────────────────────
def run_test(name, keeper_args, server_cmd, fn, timeout=30) -> bool | None:
    flag_str = (" ".join(keeper_args) + " ") if keeper_args else ""
    cmd_str = " ".join(server_cmd)
    print(f"{'─' * 60}")
    print(f"  {name}")
    print(f"  $ mcp-keeper {flag_str}{cmd_str}")

    if not shutil.which(server_cmd[0]):
        print(f"  {SKIP}: '{server_cmd[0]}' not found in PATH")
        return None

    try:
        ok, detail = fn(keeper_args, server_cmd, timeout)
        print(f"  {PASS if ok else FAIL}: {detail}")
        return ok
    except Exception as exc:
        print(f"  {FAIL}: {exc}")
        return False


# ── Test functions ────────────────────────────────────────────────────────────
def test_transparent_list(keeper_args, server_cmd, timeout):
    tools = list_tools(keeper_args, server_cmd, timeout)
    if tools is None:
        return False, "No tools/list response"
    ok = "get_current_time" in tools and "convert_time" in tools
    return ok, f"tools: {tools}"

def test_filter_list(keeper_args, server_cmd, timeout):
    tools = list_tools(keeper_args, server_cmd, timeout)
    if tools is None:
        return False, "No tools/list response"
    ok = tools == ["get_current_time"]
    return ok, f"expected ['get_current_time'], got {tools}"

def test_exclude_list(keeper_args, server_cmd, timeout):
    tools = list_tools(keeper_args, server_cmd, timeout)
    if tools is None:
        return False, "No tools/list response"
    leaked = [t for t in tools if t.endswith("_file")]
    ok = len(tools) > 0 and not leaked
    return ok, f"remaining tools: {tools}" if ok else f"*_file tools still present: {leaked}"

def test_npx_memory_list(keeper_args, server_cmd, timeout):
    tools = list_tools(keeper_args, server_cmd, timeout)
    if tools is None:
        return False, "No tools/list response"
    return len(tools) > 0, f"{len(tools)} tools: {tools}"

def test_call_get_current_time(keeper_args, server_cmd, timeout):
    result = call_tool(keeper_args, server_cmd, "get_current_time", {"timezone": "UTC"}, timeout)
    if result is None:
        return False, "No tools/call response"
    content = result.get("content", [])
    text = content[0].get("text", "") if content else ""
    ok = bool(text)
    return ok, f"got response: {text!r}"

def test_call_after_filter(keeper_args, server_cmd, timeout):
    """After --allow=get_*, get_current_time should still be callable."""
    result = call_tool(keeper_args, server_cmd, "get_current_time", {"timezone": "UTC"}, timeout)
    if result is None:
        return False, "No tools/call response"
    content = result.get("content", [])
    text = content[0].get("text", "") if content else ""
    ok = bool(text)
    return ok, f"tool callable after filter, got: {text!r}"

def test_call_filesystem(keeper_args, server_cmd, timeout):
    """list_directory should work through the proxy after --exclude=*_file."""
    result = call_tool(keeper_args, server_cmd, "list_directory", {"path": "/tmp"}, timeout)
    if result is None:
        return False, "No tools/call response"
    content = result.get("content", [])
    ok = len(content) > 0
    return ok, f"list_directory returned {len(content)} content item(s)"

def test_call_memory_read_graph(keeper_args, server_cmd, timeout):
    """read_graph should work through the proxy (transparent)."""
    result = call_tool(keeper_args, server_cmd, "read_graph", {}, timeout)
    if result is None:
        return False, "No tools/call response"
    content = result.get("content", [])
    ok = len(content) > 0
    return ok, f"read_graph returned {len(content)} content item(s)"

def test_call_memory_after_filter(keeper_args, server_cmd, timeout):
    """create_entities + read_graph should work after --allow=read_*."""
    result = call_tool(keeper_args, server_cmd, "read_graph", {}, timeout)
    if result is None:
        return False, "No tools/call response"
    content = result.get("content", [])
    ok = len(content) > 0
    return ok, f"read_graph callable after --allow=read_*, got {len(content)} content item(s)"


# ── Test cases ────────────────────────────────────────────────────────────────
def main():
    os.chdir(PROJECT_ROOT)
    build()

    results = []

    # ── tools/list tests ──────────────────────────────────────────────────────
    results.append(run_test(
        "6.1  tools/list — transparent proxy (mcp-server-time, no flags)",
        keeper_args=[], server_cmd=["uvx", "mcp-server-time"],
        fn=test_transparent_list,
    ))
    results.append(run_test(
        "6.2  tools/list — --allow=get_* (mcp-server-time)",
        keeper_args=["--allow=get_*"], server_cmd=["uvx", "mcp-server-time"],
        fn=test_filter_list,
    ))
    results.append(run_test(
        "6.3  tools/list — --exclude=*_file (server-filesystem)",
        keeper_args=["--exclude=*_file"],
        server_cmd=["npx", "-y", "@modelcontextprotocol/server-filesystem", "/tmp"],
        fn=test_exclude_list, timeout=60,
    ))
    results.append(run_test(
        "6.4  tools/list — transparent proxy (server-memory)",
        keeper_args=[],
        server_cmd=["npx", "-y", "@modelcontextprotocol/server-memory"],
        fn=test_npx_memory_list, timeout=60,
    ))

    # ── tools/call tests ──────────────────────────────────────────────────────
    results.append(run_test(
        "6.5  tools/call — get_current_time works (transparent proxy)",
        keeper_args=[], server_cmd=["uvx", "mcp-server-time"],
        fn=test_call_get_current_time,
    ))
    results.append(run_test(
        "6.6  tools/call — get_current_time works after --allow=get_*",
        keeper_args=["--allow=get_*"], server_cmd=["uvx", "mcp-server-time"],
        fn=test_call_after_filter,
    ))
    results.append(run_test(
        "6.7  tools/call — list_directory works after --exclude=*_file",
        keeper_args=["--exclude=*_file"],
        server_cmd=["npx", "-y", "@modelcontextprotocol/server-filesystem", "/tmp"],
        fn=test_call_filesystem, timeout=60,
    ))
    results.append(run_test(
        "6.8  tools/call — read_graph works (server-memory, transparent proxy)",
        keeper_args=[],
        server_cmd=["npx", "-y", "@modelcontextprotocol/server-memory"],
        fn=test_call_memory_read_graph, timeout=60,
    ))
    results.append(run_test(
        "6.9  tools/call — read_graph works after --allow=read_*",
        keeper_args=["--allow=read_*"],
        server_cmd=["npx", "-y", "@modelcontextprotocol/server-memory"],
        fn=test_call_memory_after_filter, timeout=60,
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
