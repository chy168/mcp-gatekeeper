package secret

import (
	"context"
	"fmt"

	"github.com/zalando/go-keyring"
)

const keychainService = "mcp-gatekeeper"

type keychainBackend struct{}

func newKeychainBackend() (*keychainBackend, error) {
	return &keychainBackend{}, nil
}

func (b *keychainBackend) Get(_ context.Context, name string) (string, error) {
	val, err := keyring.Get(keychainService, name)
	if err != nil {
		return "", fmt.Errorf("keychain: failed to get secret %q: %w", name, err)
	}
	return val, nil
}
