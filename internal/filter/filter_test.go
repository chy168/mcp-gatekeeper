package filter

import (
	"encoding/json"
	"testing"
)

// helpers

func toolsListResponse(tools []string) []byte {
	type toolEntry struct {
		Name string `json:"name"`
	}
	entries := make([]toolEntry, len(tools))
	for i, n := range tools {
		entries[i] = toolEntry{Name: n}
	}
	toolsJSON, _ := json.Marshal(entries)
	raw := map[string]json.RawMessage{
		"jsonrpc": json.RawMessage(`"2.0"`),
		"id":      json.RawMessage(`1`),
		"result":  json.RawMessage(`{"tools":` + string(toolsJSON) + `}`),
	}
	b, _ := json.Marshal(raw)
	return b
}

func toolNames(line []byte) []string {
	var r struct {
		Result struct {
			Tools []struct{ Name string } `json:"tools"`
		} `json:"result"`
	}
	json.Unmarshal(line, &r)
	var names []string
	for _, t := range r.Result.Tools {
		names = append(names, t.Name)
	}
	return names
}

// Task 5.1 — glob matching

func TestMatchesAny(t *testing.T) {
	cases := []struct {
		name     string
		patterns []string
		want     bool
	}{
		{"get_current_time", []string{"get_*"}, true},
		{"convert_time", []string{"get_*"}, false},
		{"convert_time", []string{"get_*", "convert_*"}, true},
		{"delete_file", []string{"delete_*"}, true},
		{"read_file", []string{"delete_*"}, false},
		{"anything", []string{}, false},
		{"exact", []string{"exact"}, true},
		{"exact_extra", []string{"exact"}, false},
	}
	for _, c := range cases {
		got := matchesAny(c.name, c.patterns)
		if got != c.want {
			t.Errorf("matchesAny(%q, %v) = %v, want %v", c.name, c.patterns, got, c.want)
		}
	}
}

// Task 5.2 — tools/list response identification

func TestIsToolsListResponse(t *testing.T) {
	validResponse := toolsListResponse([]string{"get_current_time", "convert_time"})
	if !IsToolsListResponse(validResponse) {
		t.Error("expected true for tools/list response")
	}

	// tools/call response (no tools array)
	toolsCall := []byte(`{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"hello"}]}}`)
	if IsToolsListResponse(toolsCall) {
		t.Error("expected false for non-tools/list response")
	}

	// request (not a response)
	request := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`)
	if IsToolsListResponse(request) {
		t.Error("expected false for request")
	}

	// invalid JSON
	if IsToolsListResponse([]byte(`not json`)) {
		t.Error("expected false for invalid JSON")
	}
}

// Task 5.3 — allowlist filtering

func TestAllowlistFilter(t *testing.T) {
	// single pattern
	line := toolsListResponse([]string{"get_current_time", "convert_time"})
	out, _ := FilterToolsListResponse(line, []string{"get_*"}, nil)
	names := toolNames(out)
	if len(names) != 1 || names[0] != "get_current_time" {
		t.Errorf("single allow: got %v", names)
	}

	// multiple patterns (OR)
	line = toolsListResponse([]string{"get_current_time", "convert_time", "delete_event"})
	out, _ = FilterToolsListResponse(line, []string{"get_*", "convert_*"}, nil)
	names = toolNames(out)
	if len(names) != 2 {
		t.Errorf("multi allow: expected 2, got %v", names)
	}

	// no match → empty
	line = toolsListResponse([]string{"get_current_time", "convert_time"})
	out, _ = FilterToolsListResponse(line, []string{"write_*"}, nil)
	names = toolNames(out)
	if len(names) != 0 {
		t.Errorf("no match: expected empty, got %v", names)
	}
}

// Task 5.4 — denylist filtering

func TestDenylistFilter(t *testing.T) {
	// single pattern
	line := toolsListResponse([]string{"read_file", "write_file", "delete_file"})
	out, _ := FilterToolsListResponse(line, nil, []string{"delete_*"})
	names := toolNames(out)
	if len(names) != 2 {
		t.Errorf("single exclude: expected 2, got %v", names)
	}
	for _, n := range names {
		if n == "delete_file" {
			t.Error("delete_file should be excluded")
		}
	}

	// multiple patterns (OR)
	line = toolsListResponse([]string{"read_file", "write_file", "delete_file"})
	out, _ = FilterToolsListResponse(line, nil, []string{"delete_*", "write_*"})
	names = toolNames(out)
	if len(names) != 1 || names[0] != "read_file" {
		t.Errorf("multi exclude: got %v", names)
	}
}

// Task 5.5 — allowlist + denylist combined

func TestCombinedFilter(t *testing.T) {
	line := toolsListResponse([]string{"read_file", "write_file", "delete_file", "list_directory"})
	out, _ := FilterToolsListResponse(line, []string{"*_file"}, []string{"delete_*"})
	names := toolNames(out)
	if len(names) != 2 {
		t.Errorf("combined: expected 2, got %v", names)
	}
	for _, n := range names {
		if n == "delete_file" || n == "list_directory" {
			t.Errorf("combined: unexpected tool %q", n)
		}
	}
}

// Task 5.6 — transparent proxy (no flags)

func TestTransparentProxy(t *testing.T) {
	tools := []string{"get_current_time", "convert_time"}
	line := toolsListResponse(tools)
	out, _ := FilterToolsListResponse(line, nil, nil)
	// should be byte-identical
	if string(out) != string(line) {
		t.Errorf("transparent: got %s, want %s", out, line)
	}
}
