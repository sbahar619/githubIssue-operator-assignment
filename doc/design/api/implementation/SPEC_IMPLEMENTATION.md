# Spec Implementation

> **Context**: [Spec Design](../design/SPEC_DESIGN.md) | [Design](../../DESIGN.md)

## Steps

### 1. Add Validation Markers

**File**: `api/v1alpha1/githubissue_types.go`

```go
// Repo is the GitHub repository URL
// +kubebuilder:validation:Pattern=`^https://github\.com/[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+$`
// +kubebuilder:validation:MinLength=19
// +kubebuilder:validation:MaxLength=150
Repo string `json:"repo"`

// Title is the GitHub issue title that will be created or updated
// +kubebuilder:validation:MinLength=1
// +kubebuilder:validation:MaxLength=256
Title string `json:"title"`

// Description is the GitHub issue body content
// +kubebuilder:validation:MaxLength=65536
// +optional
Description *string `json:"description,omitempty"`
```

### 2. Current Status

✅ **FULLY IMPLEMENTED** - All spec fields and validation markers are in place.

### 3. Verification Commands

```bash
# Generate CRD
make manifests

# Verify validation rules in generated CRD
grep -A 10 "pattern\|minLength\|maxLength" config/crd/bases/github.shahaf.com_githubissues.yaml

# Quality checks
make lint
```

## Validation Rationale

- **Repo Pattern**: Ensures valid GitHub URLs, prevents API failures
- **Repo Length**: 19-150 chars (shortest: `https://github.com/a/b`)
- **Title Length**: 1-256 chars (GitHub requirement)
- **Description Length**: Max 65KB (reasonable size, well within API limits)

## Checklist
- [x] Add validation markers to all fields
- [x] Add GoDoc comments
- [x] Generate CRD (`make manifests`)
- [x] Verify validation rules in generated CRD
- [x] Run lint checks