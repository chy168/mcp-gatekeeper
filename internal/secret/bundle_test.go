package secret

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// mockWritableBackend supports both Get and Set for testing bundle write ops.
type mockWritableBackend struct {
	values map[string]string
	setErr error
}

func (m *mockWritableBackend) Get(_ context.Context, name string) (string, error) {
	v, ok := m.values[name]
	if !ok {
		return "", fmt.Errorf("secret not found: %s", name)
	}
	return v, nil
}

func (m *mockWritableBackend) Set(_ context.Context, name, value string) error {
	if m.setErr != nil {
		return m.setErr
	}
	if m.values == nil {
		m.values = map[string]string{}
	}
	m.values[name] = value
	return nil
}

func TestParseBundle(t *testing.T) {
	t.Run("valid simple YAML", func(t *testing.T) {
		content := "api_token: abc123\njira_token: xyz789\n"
		bundle, err := ParseBundle("test-bundle", content)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if bundle["api_token"] != "abc123" {
			t.Errorf("got %q, want %q", bundle["api_token"], "abc123")
		}
		if bundle["jira_token"] != "xyz789" {
			t.Errorf("got %q, want %q", bundle["jira_token"], "xyz789")
		}
	})

	t.Run("multiline block scalar", func(t *testing.T) {
		content := "private_key: |\n  -----BEGIN RSA PRIVATE KEY-----\n  MIIEowIBAA\n  -----END RSA PRIVATE KEY-----\n"
		bundle, err := ParseBundle("test-bundle", content)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(bundle["private_key"], "BEGIN RSA") {
			t.Errorf("multiline value not preserved: %q", bundle["private_key"])
		}
		if !strings.Contains(bundle["private_key"], "\n") {
			t.Errorf("newlines not preserved in multiline value")
		}
	})

	t.Run("empty bundle", func(t *testing.T) {
		bundle, err := ParseBundle("test-bundle", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(bundle) != 0 {
			t.Errorf("expected empty bundle, got %v", bundle)
		}
	})

	t.Run("invalid YAML returns error with example", func(t *testing.T) {
		_, err := ParseBundle("test-bundle", "key: [unclosed")
		if err == nil {
			t.Fatal("expected error for invalid YAML")
		}
		if !strings.Contains(err.Error(), "test-bundle") {
			t.Errorf("error should mention bundle name: %v", err)
		}
		if !strings.Contains(err.Error(), "Expected format") {
			t.Errorf("error should include expected format example: %v", err)
		}
	})

	t.Run("non-scalar value returns error with example", func(t *testing.T) {
		_, err := ParseBundle("test-bundle", "nested:\n  foo: bar\n")
		if err == nil {
			t.Fatal("expected error for non-scalar value")
		}
		if !strings.Contains(err.Error(), "nested") {
			t.Errorf("error should mention key name: %v", err)
		}
		if !strings.Contains(err.Error(), "test-bundle") {
			t.Errorf("error should mention bundle name: %v", err)
		}
		if !strings.Contains(err.Error(), "not a string value") {
			t.Errorf("error should mention type issue: %v", err)
		}
	})

	t.Run("sequence value returns error", func(t *testing.T) {
		_, err := ParseBundle("test-bundle", "items:\n  - one\n  - two\n")
		if err == nil {
			t.Fatal("expected error for sequence value")
		}
		if !strings.Contains(err.Error(), "not a string value") {
			t.Errorf("error should mention type issue: %v", err)
		}
	})
}

func TestLookupBundle(t *testing.T) {
	bundle := map[string]string{
		"api_token":  "abc123",
		"jira_token": "xyz789",
	}

	t.Run("existing key", func(t *testing.T) {
		val, err := LookupBundle(bundle, "api_token", "test-bundle")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val != "abc123" {
			t.Errorf("got %q, want %q", val, "abc123")
		}
	})

	t.Run("missing key lists available keys", func(t *testing.T) {
		_, err := LookupBundle(bundle, "missing_key", "test-bundle")
		if err == nil {
			t.Fatal("expected error for missing key")
		}
		if !strings.Contains(err.Error(), "missing_key") {
			t.Errorf("error should mention requested key: %v", err)
		}
		if !strings.Contains(err.Error(), "test-bundle") {
			t.Errorf("error should mention bundle name: %v", err)
		}
		if !strings.Contains(err.Error(), "Available keys") {
			t.Errorf("error should list available keys: %v", err)
		}
		if !strings.Contains(err.Error(), "api_token") {
			t.Errorf("error should list api_token as available: %v", err)
		}
	})

	t.Run("empty bundle shows none", func(t *testing.T) {
		_, err := LookupBundle(map[string]string{}, "any_key", "test-bundle")
		if err == nil {
			t.Fatal("expected error for empty bundle")
		}
		if !strings.Contains(err.Error(), "(none)") {
			t.Errorf("empty bundle should show (none): %v", err)
		}
	})
}

func TestSerializeBundle(t *testing.T) {
	t.Run("normal map produces valid YAML", func(t *testing.T) {
		bundle := map[string]string{"api_token": "abc123", "jira_token": "xyz"}
		out, err := SerializeBundle(bundle)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Round-trip: parse back
		parsed, err := ParseBundle("test", out)
		if err != nil {
			t.Fatalf("round-trip parse failed: %v", err)
		}
		if parsed["api_token"] != "abc123" {
			t.Errorf("got %q, want %q", parsed["api_token"], "abc123")
		}
	})

	t.Run("multiline value preserved", func(t *testing.T) {
		bundle := map[string]string{"key": "line1\nline2\n"}
		out, err := SerializeBundle(bundle)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		parsed, err := ParseBundle("test", out)
		if err != nil {
			t.Fatalf("round-trip parse failed: %v", err)
		}
		if parsed["key"] != "line1\nline2\n" {
			t.Errorf("multiline not preserved: got %q", parsed["key"])
		}
	})

	t.Run("empty map produces valid YAML", func(t *testing.T) {
		out, err := SerializeBundle(map[string]string{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		parsed, err := ParseBundle("test", out)
		if err != nil {
			t.Fatalf("round-trip parse failed: %v", err)
		}
		if len(parsed) != 0 {
			t.Errorf("expected empty map, got %v", parsed)
		}
	})
}

func TestSetBundleKey(t *testing.T) {
	t.Run("add new key", func(t *testing.T) {
		m := &mockWritableBackend{
			values: map[string]string{"mcp-gatekeeper": "existing: value\n"},
		}
		if err := SetBundleKey(context.Background(), m, "mcp-gatekeeper", "new_key", "new_val"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		parsed, _ := ParseBundle("mcp-gatekeeper", m.values["mcp-gatekeeper"])
		if parsed["new_key"] != "new_val" {
			t.Errorf("new key not set: %v", parsed)
		}
		if parsed["existing"] != "value" {
			t.Errorf("existing key lost: %v", parsed)
		}
	})

	t.Run("update existing key", func(t *testing.T) {
		m := &mockWritableBackend{
			values: map[string]string{"mcp-gatekeeper": "api_token: old\n"},
		}
		if err := SetBundleKey(context.Background(), m, "mcp-gatekeeper", "api_token", "new"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		parsed, _ := ParseBundle("mcp-gatekeeper", m.values["mcp-gatekeeper"])
		if parsed["api_token"] != "new" {
			t.Errorf("key not updated: got %q", parsed["api_token"])
		}
	})

	t.Run("bundle not found starts from empty map", func(t *testing.T) {
		m := &mockWritableBackend{values: map[string]string{}}
		if err := SetBundleKey(context.Background(), m, "mcp-gatekeeper", "first_key", "val"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		parsed, _ := ParseBundle("mcp-gatekeeper", m.values["mcp-gatekeeper"])
		if parsed["first_key"] != "val" {
			t.Errorf("key not set in new bundle: %v", parsed)
		}
	})
}

func TestDeleteBundleKey(t *testing.T) {
	t.Run("delete existing key", func(t *testing.T) {
		m := &mockWritableBackend{
			values: map[string]string{"mcp-gatekeeper": "api_token: abc\njira_token: xyz\n"},
		}
		if err := DeleteBundleKey(context.Background(), m, "mcp-gatekeeper", "api_token"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		parsed, _ := ParseBundle("mcp-gatekeeper", m.values["mcp-gatekeeper"])
		if _, ok := parsed["api_token"]; ok {
			t.Error("key should have been deleted")
		}
		if parsed["jira_token"] != "xyz" {
			t.Error("other key should be preserved")
		}
	})

	t.Run("key not found returns error", func(t *testing.T) {
		m := &mockWritableBackend{
			values: map[string]string{"mcp-gatekeeper": "other_key: val\n"},
		}
		err := DeleteBundleKey(context.Background(), m, "mcp-gatekeeper", "missing")
		if err == nil {
			t.Fatal("expected error for missing key")
		}
		if !strings.Contains(err.Error(), "missing") {
			t.Errorf("error should mention key name: %v", err)
		}
	})
}
