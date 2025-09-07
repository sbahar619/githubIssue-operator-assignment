package auth

import (
	"fmt"
	"os"
)

const GitHubTokenEnvVar = "GITHUB_TOKEN"

func GetGitHubToken() (string, error) {
	token := os.Getenv(GitHubTokenEnvVar)
	if token == "" {
		return "", fmt.Errorf("environment variable %s is not configured", GitHubTokenEnvVar)
	}
	return token, nil
}
