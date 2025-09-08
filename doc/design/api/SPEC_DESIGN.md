# API Spec Design

## Goal
Define user-facing API for declaring desired GitHub issue state.

## Spec Fields
- **Repo** (`string`): GitHub repository URL (`https://github.com/owner/repo`)
- **Title** (`string`): GitHub issue title (1-256 characters)
- **Description** (`*string`): Optional GitHub issue body (max 65536 characters)

## Validation Strategy
- **CRD-Level**: Pattern, length, format validation via OpenAPI schema
- **Controller-Level**: Business logic validation during reconciliation

## Field Design Decisions
- **Required Fields**: `repo`, `title` (essential for GitHub issue identification)
- **Optional Fields**: `description` (GitHub allows empty issue bodies)
- **String Types**: Simple user intent representation

## Repository URL Validation
- **Pattern**: `^https://github\.com/[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+$`
- **Length**: 19-150 characters
- **Format**: Full GitHub URL for unambiguous identification
