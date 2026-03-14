package filter

import (
	"encoding/json"
	"path"
)

// Tool represents a single MCP tool entry.
type Tool struct {
	Name string `json:"name"`
}

// jsonRPCResponse is used to detect tools/list responses.
type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  *struct {
		Tools json.RawMessage `json:"tools"`
	} `json:"result"`
}

// IsToolsListResponse returns true if the line is a JSON-RPC response
// that contains a result.tools array.
func IsToolsListResponse(line []byte) bool {
	var r jsonRPCResponse
	if err := json.Unmarshal(line, &r); err != nil {
		return false
	}
	return r.Result != nil && r.Result.Tools != nil
}

// matchesAny returns true if name matches any of the given glob patterns.
func matchesAny(name string, patterns []string) bool {
	for _, p := range patterns {
		if ok, _ := path.Match(p, name); ok {
			return true
		}
	}
	return false
}

// FilterToolsListResponse applies allow/deny filtering to a tools/list response line.
// If allows and excludes are both empty, the line is returned unchanged.
// allows = allowlist (keep matching), excludes = denylist (remove matching).
func FilterToolsListResponse(line []byte, allows, excludes []string) ([]byte, error) {
	if len(allows) == 0 && len(excludes) == 0 {
		return line, nil
	}

	// Parse the full response to preserve all fields (id, jsonrpc, etc.)
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(line, &raw); err != nil {
		return line, nil
	}

	resultRaw, ok := raw["result"]
	if !ok {
		return line, nil
	}

	var result map[string]json.RawMessage
	if err := json.Unmarshal(resultRaw, &result); err != nil {
		return line, nil
	}

	toolsRaw, ok := result["tools"]
	if !ok {
		return line, nil
	}

	var tools []Tool
	if err := json.Unmarshal(toolsRaw, &tools); err != nil {
		return line, nil
	}

	// Apply allowlist
	if len(allows) > 0 {
		var kept []Tool
		for _, t := range tools {
			if matchesAny(t.Name, allows) {
				kept = append(kept, t)
			}
		}
		tools = kept
	}

	// Apply denylist
	if len(excludes) > 0 {
		var kept []Tool
		for _, t := range tools {
			if !matchesAny(t.Name, excludes) {
				kept = append(kept, t)
			}
		}
		tools = kept
	}

	if tools == nil {
		tools = []Tool{}
	}

	filteredToolsJSON, err := json.Marshal(tools)
	if err != nil {
		return line, nil
	}

	result["tools"] = filteredToolsJSON
	filteredResultJSON, err := json.Marshal(result)
	if err != nil {
		return line, nil
	}

	raw["result"] = filteredResultJSON
	out, err := json.Marshal(raw)
	if err != nil {
		return line, nil
	}
	return out, nil
}
