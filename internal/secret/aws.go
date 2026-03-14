package secret

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type awsBackend struct{}

func newAWSBackend() (*awsBackend, error) {
	return &awsBackend{}, nil
}

func (b *awsBackend) Get(ctx context.Context, name string) (string, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("aws: failed to load config: %w", err)
	}
	if cfg.Region == "" {
		return "", fmt.Errorf("aws: region is not set (set AWS_DEFAULT_REGION or AWS_REGION)")
	}

	client := secretsmanager.NewFromConfig(cfg)
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
