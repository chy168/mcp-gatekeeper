package secret

import (
	"context"
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"
)

// keychainService is the service name used for all mcp-gatekeeper secrets in the OS keychain.
// Platform support: macOS Keychain, Windows Credential Manager.
// Linux: requires a running Secret Service daemon (e.g. gnome-keyring or KWallet via D-Bus).
// Not suitable for headless/CI environments on Linux.
const keychainService = "mcp-gatekeeper"

type keychainBackend struct{}

func newKeychainBackend() (*keychainBackend, error) {
	return &keychainBackend{}, nil
}

func (b *keychainBackend) Get(_ context.Context, name string) (string, error) {
	val, err := keyring.Get(keychainService, name)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", fmt.Errorf("keychain: bundle %q not found in keychain: %w", name, ErrBundleNotFound)
		}
		return "", fmt.Errorf("keychain: failed to get secret %q: %w", name, err)
	}
	return val, nil
}

func (b *keychainBackend) Set(_ context.Context, name, value string) error {
	if err := keyring.Set(keychainService, name, value); err != nil {
		return fmt.Errorf("keychain: failed to set secret %q: %w", name, err)
	}
	return nil
}
