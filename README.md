# GitHub Issue Operator

Kubernetes operator that manages GitHub issues as custom resources, enabling GitOps workflows for issue lifecycle management.

## Features

- **Declarative issue management** - Define GitHub issues as Kubernetes CRs
- **Automatic synchronization** - Create, update, and close issues based on CR state  
- **Ownership tracking** - Label-based ownership prevents conflicts
- **Status reporting** - Real-time issue state in CR status

## Quick Start

### Prerequisites
- Kubernetes v1.11.3+
- GitHub token with repository permissions

### Installation

1. **Deploy the operator:**
```sh
make install deploy IMG=quay.io/sbahar/github-issue-operator:latest
```

2. **Create GitHub token secret:**
```sh
kubectl create secret generic github-token --from-literal=token=<your-github-token>
```

3. **Create an issue:**
```yaml
apiVersion: github.shahaf.com/v1alpha1
kind: GithubIssue
metadata:
  name: my-issue
spec:
  repo: https://github.com/owner/repo
  title: "Feature request from K8s"
  description: "This issue was created by the operator"
```

### Example Usage

```sh
# Create issue
kubectl apply -f config/samples/github_v1alpha1_githubissue.yaml

# Check status
kubectl get githubissue my-issue -o yaml

# Update issue (modify spec.title or spec.description)
kubectl edit githubissue my-issue

# Delete issue (closes on GitHub)
kubectl delete githubissue my-issue
```

## Development

```sh
# Run locally
make run

# Build and test
make build test

# Run E2E tests
make test-e2e
```

## License

Apache 2.0 - see [LICENSE](LICENSE) for details.