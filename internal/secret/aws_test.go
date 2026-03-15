package secret

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
)

type mockAWSClient struct {
	putErr    error
	createErr error
	getResult string
	getErr    error
	putCalls  int
	createCalls int
}

func (m *mockAWSClient) PutSecretValue(_ context.Context, _ *secretsmanager.PutSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.PutSecretValueOutput, error) {
	m.putCalls++
	return &secretsmanager.PutSecretValueOutput{}, m.putErr
}

func (m *mockAWSClient) CreateSecret(_ context.Context, _ *secretsmanager.CreateSecretInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.CreateSecretOutput, error) {
	m.createCalls++
	return &secretsmanager.CreateSecretOutput{}, m.createErr
}

func (m *mockAWSClient) GetSecretValue(_ context.Context, _ *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return &secretsmanager.GetSecretValueOutput{SecretString: &m.getResult}, nil
}

func newTestAWSBackend(client awsSecretsManagerClient) *awsBackend {
	return &awsBackend{
		newClient: func(_ context.Context) (awsSecretsManagerClient, error) {
			return client, nil
		},
	}
}

func TestAWSBackend_Set_ExistingSecret(t *testing.T) {
	mock := &mockAWSClient{}
	b := newTestAWSBackend(mock)

	if err := b.Set(context.Background(), "my-bundle", "value"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.putCalls != 1 {
		t.Errorf("expected 1 PutSecretValue call, got %d", mock.putCalls)
	}
	if mock.createCalls != 0 {
		t.Errorf("expected 0 CreateSecret calls, got %d", mock.createCalls)
	}
}

func TestAWSBackend_Set_SecretNotFound_Creates(t *testing.T) {
	notFoundErr := &types.ResourceNotFoundException{Message: strPtr("not found")}
	mock := &mockAWSClient{putErr: notFoundErr}
	b := newTestAWSBackend(mock)

	if err := b.Set(context.Background(), "new-bundle", "value"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.createCalls != 1 {
		t.Errorf("expected 1 CreateSecret call, got %d", mock.createCalls)
	}
}

func TestAWSBackend_Set_CreateFails(t *testing.T) {
	notFoundErr := &types.ResourceNotFoundException{Message: strPtr("not found")}
	mock := &mockAWSClient{putErr: notFoundErr, createErr: fmt.Errorf("access denied")}
	b := newTestAWSBackend(mock)

	err := b.Set(context.Background(), "my-bundle", "value")
	if err == nil {
		t.Fatal("expected error when create fails")
	}
}

func TestAWSBackend_Set_PutFails_NonNotFound(t *testing.T) {
	mock := &mockAWSClient{putErr: fmt.Errorf("throttled")}
	b := newTestAWSBackend(mock)

	err := b.Set(context.Background(), "my-bundle", "value")
	if err == nil {
		t.Fatal("expected error on non-NotFound put failure")
	}
}

func TestAWSBackend_Get(t *testing.T) {
	mock := &mockAWSClient{getResult: "bundle-yaml"}
	b := newTestAWSBackend(mock)

	val, err := b.Get(context.Background(), "my-bundle")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "bundle-yaml" {
		t.Errorf("got %q, want %q", val, "bundle-yaml")
	}
}

func TestAWSBackend_Get_Error(t *testing.T) {
	mock := &mockAWSClient{getErr: fmt.Errorf("not found")}
	b := newTestAWSBackend(mock)

	_, err := b.Get(context.Background(), "my-bundle")
	if err == nil {
		t.Fatal("expected error")
	}
}

func strPtr(s string) *string { return &s }
