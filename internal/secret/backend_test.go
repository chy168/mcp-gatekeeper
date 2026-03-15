package secret

import (
	"context"
	"fmt"
	"testing"
)

type mockBackend struct {
	values map[string]string
}

func (m *mockBackend) Get(_ context.Context, name string) (string, error) {
	v, ok := m.values[name]
	if !ok {
		return "", fmt.Errorf("secret not found: %s", name)
	}
	return v, nil
}

func TestResolveAllBundle(t *testing.T) {
	bundleYAML := "db_password: supersecret\napi_key: key123\n"
	mock := &mockBackend{
		values: map[string]string{
			"mcp-gatekeeper": bundleYAML,
		},
	}

	result, err := resolveAllWithBackend(mock, "mcp-gatekeeper", []string{"db_password", "api_key"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["db_password"] != "supersecret" {
		t.Errorf("got %q, want %q", result["db_password"], "supersecret")
	}
	if result["api_key"] != "key123" {
		t.Errorf("got %q, want %q", result["api_key"], "key123")
	}
}

func TestResolveAllBundle_MissingKey(t *testing.T) {
	mock := &mockBackend{
		values: map[string]string{
			"mcp-gatekeeper": "other_key: value\n",
		},
	}

	_, err := resolveAllWithBackend(mock, "mcp-gatekeeper", []string{"missing"})
	if err == nil {
		t.Fatal("expected error for missing key, got nil")
	}
}

func TestResolveAllBundle_BundleNotFound(t *testing.T) {
	mock := &mockBackend{values: map[string]string{}}

	_, err := resolveAllWithBackend(mock, "mcp-gatekeeper", []string{"any_key"})
	if err == nil {
		t.Fatal("expected error when bundle not found")
	}
}

func TestResolveAllBundle_NoRefs(t *testing.T) {
	mock := &mockBackend{values: map[string]string{}}

	result, err := resolveAllWithBackend(mock, "mcp-gatekeeper", []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result, got %v", result)
	}
}

func TestNewBackend_UnsupportedSource(t *testing.T) {
	_, err := NewBackend("vault")
	if err == nil {
		t.Fatal("expected error for unsupported source")
	}
}
