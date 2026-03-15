package secret

import (
	"context"
	"fmt"
	"os"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	gax "github.com/googleapis/gax-go/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// gcpSecretManagerClient is the interface used by gcpBackend for testability.
type gcpSecretManagerClient interface {
	AddSecretVersion(ctx context.Context, req *secretmanagerpb.AddSecretVersionRequest, opts ...gax.CallOption) (*secretmanagerpb.SecretVersion, error)
	CreateSecret(ctx context.Context, req *secretmanagerpb.CreateSecretRequest, opts ...gax.CallOption) (*secretmanagerpb.Secret, error)
	AccessSecretVersion(ctx context.Context, req *secretmanagerpb.AccessSecretVersionRequest, opts ...gax.CallOption) (*secretmanagerpb.AccessSecretVersionResponse, error)
	Close() error
}

type gcpBackend struct {
	project   string
	newClient func(ctx context.Context) (gcpSecretManagerClient, error)
}

func newGCPBackend() (*gcpBackend, error) {
	project := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if project == "" {
		return nil, fmt.Errorf("GOOGLE_CLOUD_PROJECT is not set")
	}
	return &gcpBackend{
		project: project,
		newClient: func(ctx context.Context) (gcpSecretManagerClient, error) {
			return secretmanager.NewClient(ctx)
		},
	}, nil
}

func (b *gcpBackend) Set(ctx context.Context, name, value string) error {
	client, err := b.newClient(ctx)
	if err != nil {
		return fmt.Errorf("gcp: failed to create client: %w", err)
	}
	defer client.Close()

	secretName := fmt.Sprintf("projects/%s/secrets/%s", b.project, name)

	// Try to add a new version; create the secret first if it doesn't exist.
	_, err = client.AddSecretVersion(ctx, &secretmanagerpb.AddSecretVersionRequest{
		Parent:  secretName,
		Payload: &secretmanagerpb.SecretPayload{Data: []byte(value)},
	})
	if err != nil {
		if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
			// Secret doesn't exist — create it, then add the version.
			_, createErr := client.CreateSecret(ctx, &secretmanagerpb.CreateSecretRequest{
				Parent:   fmt.Sprintf("projects/%s", b.project),
				SecretId: name,
				Secret: &secretmanagerpb.Secret{
					Replication: &secretmanagerpb.Replication{
						Replication: &secretmanagerpb.Replication_Automatic_{
							Automatic: &secretmanagerpb.Replication_Automatic{},
						},
					},
				},
			})
			if createErr != nil {
				return fmt.Errorf("gcp: failed to create secret %q (ensure the ADC principal has roles/secretmanager.admin or secretmanager.secrets.create): %w", name, createErr)
			}
			_, err = client.AddSecretVersion(ctx, &secretmanagerpb.AddSecretVersionRequest{
				Parent:  secretName,
				Payload: &secretmanagerpb.SecretPayload{Data: []byte(value)},
			})
			if err != nil {
				return fmt.Errorf("gcp: failed to add secret version for %q: %w", name, err)
			}
			return nil
		}
		return fmt.Errorf("gcp: failed to add secret version for %q (ensure the ADC principal has roles/secretmanager.secretVersionAdder): %w", name, err)
	}
	return nil
}

func (b *gcpBackend) Get(ctx context.Context, name string) (string, error) {
	client, err := b.newClient(ctx)
	if err != nil {
		return "", fmt.Errorf("gcp: failed to create client: %w", err)
	}
	defer client.Close()

	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/latest", b.project, name),
	}
	result, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
			return "", fmt.Errorf("gcp: bundle %q not found in GCP Secret Manager: %w", name, ErrBundleNotFound)
		}
		return "", fmt.Errorf("gcp: failed to access secret %q: %w", name, err)
	}
	return string(result.Payload.Data), nil
}
