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

func TestResolveAllWithBackend(t *testing.T) {
	mock := &mockBackend{
		values: map[string]string{
			"db-password": "supersecret",
			"api-key":     "key123",
		},
	}

	result, err := resolveAllWithBackend(mock, []string{"db-password", "api-key"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["db-password"] != "supersecret" {
		t.Errorf("got %q, want %q", result["db-password"], "supersecret")
	}
	if result["api-key"] != "key123" {
		t.Errorf("got %q, want %q", result["api-key"], "key123")
	}
}

func TestResolveAllWithBackend_MissingSecret(t *testing.T) {
	mock := &mockBackend{values: map[string]string{}}

	_, err := resolveAllWithBackend(mock, []string{"missing"})
	if err == nil {
		t.Fatal("expected error for missing secret, got nil")
	}
}

func TestResolveAllWithBackend_Empty(t *testing.T) {
	mock := &mockBackend{values: map[string]string{}}

	result, err := resolveAllWithBackend(mock, []string{})
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
