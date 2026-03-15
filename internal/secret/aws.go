package secret

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
)

// awsSecretsManagerClient is the interface used by awsBackend for testability.
type awsSecretsManagerClient interface {
	GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
	PutSecretValue(ctx context.Context, params *secretsmanager.PutSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.PutSecretValueOutput, error)
	CreateSecret(ctx context.Context, params *secretsmanager.CreateSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.CreateSecretOutput, error)
}

type awsBackend struct {
	newClient func(ctx context.Context) (awsSecretsManagerClient, error)
}

func newAWSBackend() (*awsBackend, error) {
	return &awsBackend{
		newClient: func(ctx context.Context) (awsSecretsManagerClient, error) {
			cfg, err := config.LoadDefaultConfig(ctx)
			if err != nil {
				return nil, fmt.Errorf("aws: failed to load config: %w", err)
			}
			if cfg.Region == "" {
				return nil, fmt.Errorf("aws: region is not set (set AWS_DEFAULT_REGION or AWS_REGION)")
			}
			return secretsmanager.NewFromConfig(cfg), nil
		},
	}, nil
}

func (b *awsBackend) Set(ctx context.Context, name, value string) error {
	client, err := b.newClient(ctx)
	if err != nil {
		return err
	}

	_, err = client.PutSecretValue(ctx, &secretsmanager.PutSecretValueInput{
		SecretId:     &name,
		SecretString: &value,
	})
	if err != nil {
		var notFound *types.ResourceNotFoundException
		if errors.As(err, &notFound) {
			// Secret doesn't exist — create it first.
			_, createErr := client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
				Name:         &name,
				SecretString: aws.String(value),
			})
			if createErr != nil {
				return fmt.Errorf("aws: failed to create secret %q: %w", name, createErr)
			}
			return nil
		}
		return fmt.Errorf("aws: failed to put secret value for %q: %w", name, err)
	}
	return nil
}

func (b *awsBackend) Get(ctx context.Context, name string) (string, error) {
	client, err := b.newClient(ctx)
	if err != nil {
		return "", err
	}

	result, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: &name,
	})
	if err != nil {
		return "", fmt.Errorf("aws: failed to get secret %q: %w", name, err)
	}
	if result.SecretString != nil {
		return *result.SecretString, nil
	}
	return "", fmt.Errorf("aws: secret %q has no string value", name)
}
