package secret

import (
	"context"
	"testing"

	"github.com/zalando/go-keyring"
)

func TestKeychainBackend_SetAndGet(t *testing.T) {
	keyring.MockInit()

	b := &keychainBackend{}
	ctx := context.Background()

	if err := b.Set(ctx, "test-bundle", "api_token: abc123\n"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	val, err := b.Get(ctx, "test-bundle")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "api_token: abc123\n" {
		t.Errorf("got %q, want %q", val, "api_token: abc123\n")
	}
}

func TestKeychainBackend_Set_Overwrite(t *testing.T) {
	keyring.MockInit()

	b := &keychainBackend{}
	ctx := context.Background()

	if err := b.Set(ctx, "my-bundle", "first"); err != nil {
		t.Fatalf("first Set failed: %v", err)
	}
	if err := b.Set(ctx, "my-bundle", "second"); err != nil {
		t.Fatalf("second Set failed: %v", err)
	}

	val, err := b.Get(ctx, "my-bundle")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "second" {
		t.Errorf("got %q, want %q", val, "second")
	}
}

func TestKeychainBackend_Get_NotFound(t *testing.T) {
	keyring.MockInit()

	b := &keychainBackend{}
	_, err := b.Get(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing key")
	}
}
