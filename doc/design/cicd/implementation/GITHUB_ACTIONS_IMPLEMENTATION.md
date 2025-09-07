# GitHub Actions Implementation

## Files to Create

### `.github/actions/setup-go/action.yml`
```yaml
name: 'Setup Go Environment'
description: 'Sets up Go with caching'

inputs:
  go-version:
    description: 'Go version to use'
    required: true
    default: '1.24'

runs:
  using: 'composite'
  steps:
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: ${{ inputs.go-version }}
        cache: true
        cache-dependency-path: '**/go.sum'
```

### `.github/workflows/build.yml`
```yaml
name: Build

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main, develop ]

env:
  GO_VERSION: '1.24'

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
    - name: Checkout code
      uses: actions/checkout@v4
    
    - name: Setup Go
      uses: ./.github/actions/setup-go
      with:
        go-version: ${{ env.GO_VERSION }}

    - name: Build binary
      run: make build
```

### `.github/workflows/code-quality.yml`
```yaml
name: Code Quality

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main, develop ]

env:
  GO_VERSION: '1.24'

jobs:
  code-quality:
    runs-on: ubuntu-latest
    steps:
    - name: Checkout code
      uses: actions/checkout@v4
    
    - name: Setup Go
      uses: ./.github/actions/setup-go
      with:
        go-version: ${{ env.GO_VERSION }}

    - name: Format check
      run: |
        make fmt
        git diff --exit-code

    - name: Vet
      run: make vet

    - name: Lint
      uses: golangci/golangci-lint-action@v6
      with:
        version: v2.4.0
```

### `.github/workflows/unit-tests.yml`
```yaml
name: Unit Tests

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main, develop ]

env:
  GO_VERSION: '1.24'

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
    - name: Checkout code
      uses: actions/checkout@v4
    
    - name: Setup Go
      uses: ./.github/actions/setup-go
      with:
        go-version: ${{ env.GO_VERSION }}

    - name: Run tests
      run: make test
```

