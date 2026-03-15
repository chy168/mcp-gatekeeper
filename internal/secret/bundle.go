package secret

import (
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

var bundleExample = `
Expected format:
  api_token: your-token-value
  private_key: |
    -----BEGIN RSA PRIVATE KEY-----
    ...
    -----END RSA PRIVATE KEY-----`

// ParseBundle parses a YAML string into a flat map[string]string.
// Returns an error (with example) if the content is not valid YAML or if any
// top-level value is not a scalar.
func ParseBundle(bundleName, content string) (map[string]string, error) {
	var raw map[string]interface{}
	if err := yaml.Unmarshal([]byte(content), &raw); err != nil {
		return nil, fmt.Errorf("failed to parse secret bundle %q: %w\n%s", bundleName, err, bundleExample)
	}

	out := make(map[string]string, len(raw))
	for k, v := range raw {
		switch val := v.(type) {
		case string:
			out[k] = val
		case int, int64, float64, bool:
			// Accept scalar non-string types by converting to string
			out[k] = fmt.Sprintf("%v", val)
		default:
			return nil, fmt.Errorf(
				"secret key %q in bundle %q is not a string value\n\nExpected format:\n  %s: your-string-value\n  multiline_value: |\n    line one\n    line two",
				k, bundleName, k,
			)
		}
	}
	return out, nil
}

// LookupBundle returns the value for key from bundle.
// Returns an error listing available keys if the key is not found.
func LookupBundle(bundle map[string]string, key, bundleName string) (string, error) {
	val, ok := bundle[key]
	if !ok {
		keys := make([]string, 0, len(bundle))
		for k := range bundle {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		available := strings.Join(keys, ", ")
		if available == "" {
			available = "(none)"
		}
		return "", fmt.Errorf("secret key %q not found in bundle %q\n\nAvailable keys: %s", key, bundleName, available)
	}
	return val, nil
}
