package secret

import (
	"context"
	"fmt"
)

// Backend fetches a secret value by name.
type Backend interface {
	Get(ctx context.Context, name string) (string, error)
}

// NewBackend returns a Backend for the given source name.
// Returns an error for unknown sources.
func NewBackend(source string) (Backend, error) {
	switch source {
	case "gcp":
		return newGCPBackend()
	case "aws":
		return newAWSBackend()
	case "keychain":
		return newKeychainBackend()
	default:
		return nil, fmt.Errorf("unsupported secret source %q: must be gcp, aws, or keychain", source)
	}
}
