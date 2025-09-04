# API Spec Implementation Plan

## Overview
This document outlines the step-by-step implementation plan for updating `api/v1alpha1/githubissue_types.go` to align with the approved design requirements.

## 🎯 Implementation Steps

### Step 1: Update GithubIssueSpec Validation Markers

#### 1.1 Repo Field Validations
```go
// Add above Repo field:
// +kubebuilder:validation:Pattern=`^https://github\.com/[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+$`
// +kubebuilder:validation:MinLength=19
// +kubebuilder:validation:MaxLength=150
```

#### 1.2 Title Field Validations
```go
// Add above Title field:
// +kubebuilder:validation:MinLength=1
// +kubebuilder:validation:MaxLength=256
```

#### 1.3 Description Field Validations
```go
// Add above Description field:
// +kubebuilder:validation:MaxLength=65536
```

### Step 2: Add GoDoc Comments

#### 2.1 Repo Field Documentation
```go
// Repo is the GitHub repository URL (e.g., "https://github.com/octocat/Hello-World")
```

#### 2.2 Title Field Documentation
```go
// Title is the GitHub issue title that will be created or updated
```

#### 2.3 Description Field Documentation
```go
// Description is the GitHub issue body content
```

### Step 3: Code Cleanup

#### 3.1 Remove Scaffolding Comments
- Remove: `// EDIT THIS FILE! THIS IS SCAFFOLDING FOR YOU TO OWN!`
- Remove: `// NOTE: json tags are required...` comment

#### 3.2 Keep Required Elements
- ✅ License header
- ✅ Package declaration and imports
- ✅ Existing kubebuilder object markers
- ✅ JSON tags and existing +optional marker

### Step 4: Verification Tasks

#### 4.1 Generate CRD Manifests
```bash
make manifests
```

#### 4.2 Validate Generated CRD
- Check `config/crd/bases/github.shahaf.com_githubissues.yaml`
- Verify validation rules are present in OpenAPI schema
- Confirm pattern, minLength, maxLength constraints are applied

#### 4.3 Code Quality Checks
```bash
make lint
go vet ./api/...
```

## 🔍 Validation Justifications

### Repo Field Validations

#### Pattern: `^https://github\.com/[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+$`
**Purpose**: Ensures only valid GitHub repository URLs are accepted  
**Problem Solved**: Prevents GitHub API failures from malformed URLs  
**Example Valid**: `https://github.com/octocat/Hello-World`  
**Example Invalid**: `github.com/user/repo`, `https://gitlab.com/user/repo`

#### MinLength: 19
**Purpose**: Enforces minimum viable GitHub URL length  
**Problem Solved**: Catches obviously truncated or incomplete URLs  
**Rationale**: Shortest valid URL is `https://github.com/a/b` (19 characters)  
**Example Invalid**: `https://github.com/`, `github.com`

#### MaxLength: 150
**Purpose**: Prevents extremely long URLs that could cause issues  
**Problem Solved**: Avoids resource bloat and potential API server issues  
**Rationale**: Accommodates realistic GitHub URLs while preventing abuse  
**Safety Margin**: Well within Kubernetes API server limits

### Title Field Validations

#### MinLength: 1
**Purpose**: Prevents empty GitHub issue titles  
**Problem Solved**: GitHub API rejects empty titles, causing controller failures  
**Rationale**: GitHub requires non-empty issue titles (hard requirement)  
**Example Invalid**: `""` (empty string)

#### MaxLength: 256
**Purpose**: Enforces GitHub's issue title character limit  
**Problem Solved**: Prevents GitHub API rejections due to oversized titles  
**Rationale**: Based on GitHub's actual API constraints  
**Safety**: Aligns with GitHub platform limits

### Description Field Validations

#### MaxLength: 65536
**Purpose**: Prevents extremely large issue descriptions  
**Problem Solved**: Avoids resource bloat and performance issues  
**Rationale**: 64KB is reasonable for issue descriptions while staying well within API server limits  
**Safety Margin**: 65KB total CR size is ~6% of 1MB practical Kubernetes limit  
**No MinLength**: Empty descriptions are valid (optional field)

## 📋 Implementation Checklist

### Validation Markers
- [ ] Repo pattern validation for GitHub URL format
- [ ] Repo length constraints (19-150 characters)
- [ ] Title length constraints (1-256 characters)
- [ ] Description length constraint (max 65536 characters)

### Documentation
- [ ] Repo field GoDoc comment
- [ ] Title field GoDoc comment  
- [ ] Description field GoDoc comment

### Code Cleanup
- [ ] Remove scaffolding comments
- [ ] Verify JSON tags remain intact
- [ ] Confirm existing kubebuilder markers preserved

### Verification
- [ ] CRD generation succeeds (`make manifests`)
- [ ] Validation rules appear in generated CRD
- [ ] Lint checks pass
- [ ] No breaking changes to existing structure

## 🚨 Important Notes

### Breaking Changes
- New validation rules will be enforced at CRD level
- Invalid existing resources may be rejected after update
- Test with sample resources before deploying

### Dependencies
- No new imports required (all using existing metav1)
- No changes to field types or JSON tags
- Maintains backward compatibility for valid resources

### Expected Outcome
- CRD-level validation prevents invalid GithubIssue resources
- Clear field documentation for users
- Aligned with design requirements and coding standards

## ⏱️ Estimated Time
- Implementation: 30 minutes
- Testing and verification: 15 minutes
- **Total: 45 minutes**