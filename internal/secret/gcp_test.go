package secret

import (
	"context"
	"fmt"
	"testing"

	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	gax "github.com/googleapis/gax-go/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockGCPClient struct {
	addVersionErr  error
	createErr      error
	accessResult   string
	accessErr      error
	addVersionCall int
	createCall     int
}

func (m *mockGCPClient) AddSecretVersion(_ context.Context, _ *secretmanagerpb.AddSecretVersionRequest, _ ...gax.CallOption) (*secretmanagerpb.SecretVersion, error) {
	m.addVersionCall++
	return &secretmanagerpb.SecretVersion{}, m.addVersionErr
}

func (m *mockGCPClient) CreateSecret(_ context.Context, _ *secretmanagerpb.CreateSecretRequest, _ ...gax.CallOption) (*secretmanagerpb.Secret, error) {
	m.createCall++
	return &secretmanagerpb.Secret{}, m.createErr
}

func (m *mockGCPClient) AccessSecretVersion(_ context.Context, _ *secretmanagerpb.AccessSecretVersionRequest, _ ...gax.CallOption) (*secretmanagerpb.AccessSecretVersionResponse, error) {
	if m.accessErr != nil {
		return nil, m.accessErr
	}
	return &secretmanagerpb.AccessSecretVersionResponse{
		Payload: &secretmanagerpb.SecretPayload{Data: []byte(m.accessResult)},
	}, nil
}

func (m *mockGCPClient) Close() error { return nil }

func newTestGCPBackend(client gcpSecretManagerClient) *gcpBackend {
	return &gcpBackend{
		project: "test-project",
		newClient: func(_ context.Context) (gcpSecretManagerClient, error) {
			return client, nil
		},
	}
}

func TestGCPBackend_Set_ExistingSecret(t *testing.T) {
	mock := &mockGCPClient{}
	b := newTestGCPBackend(mock)

	if err := b.Set(context.Background(), "my-bundle", "value"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.addVersionCall != 1 {
		t.Errorf("expected 1 AddSecretVersion call, got %d", mock.addVersionCall)
	}
	if mock.createCall != 0 {
		t.Errorf("expected 0 CreateSecret calls, got %d", mock.createCall)
	}
}

func TestGCPBackend_Set_SecretNotFound_CreatesAndAddsVersion(t *testing.T) {
	notFoundErr := status.Error(codes.NotFound, "secret not found")
	mock := &mockGCPClient{addVersionErr: notFoundErr}
	b := newTestGCPBackend(mock)

	// After create, second AddSecretVersion should succeed (reset error).
	b.newClient = func(_ context.Context) (gcpSecretManagerClient, error) {
		return &sequencedGCPClient{
			firstAddErr: notFoundErr,
		}, nil
	}

	if err := b.Set(context.Background(), "new-secret", "value"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGCPBackend_Set_CreateFails(t *testing.T) {
	notFoundErr := status.Error(codes.NotFound, "secret not found")
	mock := &sequencedGCPClient{
		firstAddErr: notFoundErr,
		createErr:   fmt.Errorf("permission denied"),
	}
	b := &gcpBackend{
		project: "test-project",
		newClient: func(_ context.Context) (gcpSecretManagerClient, error) {
			return mock, nil
		},
	}

	err := b.Set(context.Background(), "my-bundle", "value")
	if err == nil {
		t.Fatal("expected error when create fails")
	}
}

func TestGCPBackend_Get(t *testing.T) {
	mock := &mockGCPClient{accessResult: "bundle-yaml-content"}
	b := newTestGCPBackend(mock)

	val, err := b.Get(context.Background(), "my-bundle")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "bundle-yaml-content" {
		t.Errorf("got %q, want %q", val, "bundle-yaml-content")
	}
}

func TestGCPBackend_Get_Error(t *testing.T) {
	mock := &mockGCPClient{accessErr: fmt.Errorf("not found")}
	b := newTestGCPBackend(mock)

	_, err := b.Get(context.Background(), "my-bundle")
	if err == nil {
		t.Fatal("expected error")
	}
}

// sequencedGCPClient fails AddSecretVersion on the first call, succeeds on subsequent calls.
type sequencedGCPClient struct {
	firstAddErr error
	createErr   error
	addCalls    int
}

func (s *sequencedGCPClient) AddSecretVersion(_ context.Context, _ *secretmanagerpb.AddSecretVersionRequest, _ ...gax.CallOption) (*secretmanagerpb.SecretVersion, error) {
	s.addCalls++
	if s.addCalls == 1 {
		return nil, s.firstAddErr
	}
	return &secretmanagerpb.SecretVersion{}, nil
}

func (s *sequencedGCPClient) CreateSecret(_ context.Context, _ *secretmanagerpb.CreateSecretRequest, _ ...gax.CallOption) (*secretmanagerpb.Secret, error) {
	return &secretmanagerpb.Secret{}, s.createErr
}

func (s *sequencedGCPClient) AccessSecretVersion(_ context.Context, _ *secretmanagerpb.AccessSecretVersionRequest, _ ...gax.CallOption) (*secretmanagerpb.AccessSecretVersionResponse, error) {
	return nil, nil
}

func (s *sequencedGCPClient) Close() error { return nil }
