package secret

import (
	"strings"
	"testing"
)

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
