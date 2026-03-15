package secret

import (
	"context"
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

// SerializeBundle serializes a map[string]string to a YAML string.
// Keys are sorted for deterministic output.
func SerializeBundle(bundle map[string]string) (string, error) {
	// Build an ordered yaml.Node to preserve key sort order.
	keys := make([]string, 0, len(bundle))
	for k := range bundle {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Use yaml.Marshal on a sorted slice of structs isn't easy; marshal map directly.
	// Sort by constructing a yaml.Node manually.
	mapping := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	for _, k := range keys {
		mapping.Content = append(mapping.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: k},
			&yaml.Node{Kind: yaml.ScalarNode, Value: bundle[k]},
		)
	}
	doc := &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{mapping}}
	out, err := yaml.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("failed to serialize bundle: %w", err)
	}
	return string(out), nil
}

// SetBundleKey fetches the bundle, upserts the key, and writes it back.
// If the bundle does not exist in the backend, starts from an empty map.
func SetBundleKey(ctx context.Context, backend Backend, bundleName, key, value string) error {
	bundle := map[string]string{}

	content, err := backend.Get(ctx, bundleName)
	if err == nil {
		parsed, parseErr := ParseBundle(bundleName, content)
		if parseErr != nil {
			return parseErr
		}
		bundle = parsed
	}
	// If Get returned an error we treat the bundle as non-existent and start fresh.

	bundle[key] = value

	serialized, err := SerializeBundle(bundle)
	if err != nil {
		return err
	}
	return backend.Set(ctx, bundleName, serialized)
}

// DeleteBundleKey fetches the bundle, removes the key, and writes it back.
// Returns an error if the key does not exist.
func DeleteBundleKey(ctx context.Context, backend Backend, bundleName, key string) error {
	content, err := backend.Get(ctx, bundleName)
	if err != nil {
		return err
	}

	bundle, err := ParseBundle(bundleName, content)
	if err != nil {
		return err
	}

	if _, ok := bundle[key]; !ok {
		return fmt.Errorf("secret key %q not found in bundle %q", key, bundleName)
	}

	delete(bundle, key)

	serialized, err := SerializeBundle(bundle)
	if err != nil {
		return err
	}
	return backend.Set(ctx, bundleName, serialized)
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
