package auth

import (
	"context"
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const DefaultSecretName = "github-token-secret"

// TokenRetriever handles GitHub token retrieval from Kubernetes secrets
type TokenRetriever struct {
	client client.Client
}

// NewTokenRetriever creates a new token retriever instance
func NewTokenRetriever(k8sClient client.Client) *TokenRetriever {
	return &TokenRetriever{client: k8sClient}
}

// GetGitHubToken retrieves GitHub token from secret in the given namespace
func (tr *TokenRetriever) GetGitHubToken(ctx context.Context, namespace string) (string, error) {
	secretName := tr.getSecretName()

	secret := &corev1.Secret{}
	key := types.NamespacedName{
		Namespace: namespace,
		Name:      secretName,
	}

	if err := tr.client.Get(ctx, key, secret); err != nil {
		return "", fmt.Errorf("failed to get secret %s/%s: %w", namespace, secretName, err)
	}

	tokenBytes, exists := secret.Data["token"]
	if !exists {
		return "", fmt.Errorf("token key not found in secret %s/%s", namespace, secretName)
	}

	token := string(tokenBytes)
	if token == "" {
		return "", fmt.Errorf("empty token in secret %s/%s", namespace, secretName)
	}

	return token, nil
}

func (tr *TokenRetriever) getSecretName() string {
	if name := os.Getenv("GITHUB_TOKEN_SECRET_NAME"); name != "" {
		return name
	}
	return DefaultSecretName
}
