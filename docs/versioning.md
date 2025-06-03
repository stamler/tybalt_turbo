# Tybalt Turbo Versioning System

This document describes the comprehensive versioning system implemented for Tybalt Turbo, which tracks versions across both the UI (Svelte) and backend (Go/PocketBase) components.

## Version Format

The application uses a simplified hybrid versioning format: `v{major}.{minor}.{build}`

- **Major.Minor**: Semantic versioning for significant changes (e.g., `0.1`)
- **Build**: Auto-incrementing build number for all deployments (e.g., `123`)
- **Full Version**: Combined format (e.g., `v0.1.123`)

This format provides unique deployment tracking while maintaining semantic meaning for major releases and feature updates.

## Version Storage

The version information is centrally managed in `version.json`:

```json
{
  "version": "0.1",
  "build": 1,
  "name": "Tybalt Turbo"
}
```

## Implementation Components

### 1. Version Generation Script (`scripts/version.sh`)

Generates version information in multiple formats:

```bash
# Display all version info
./scripts/version.sh

# Generate TypeScript constants for UI
./scripts/version.sh ts > ui/src/lib/version.ts

# Generate Go constants for backend
./scripts/version.sh go > app/constants/version.go

# Get just the version number
./scripts/version.sh short  # outputs: 0.1.123

# Get version with git info
./scripts/version.sh full   # outputs: 0.1.123 (abc1234)
```

### 2. Version Increment Script (`scripts/increment-version.sh`)

Manages version updates:

```bash
# Increment build number only (most common - used for all deployments)
./scripts/increment-version.sh build

# Increment semantic version components
./scripts/increment-version.sh minor  # 0.1 -> 0.2 (build resets to 1)
./scripts/increment-version.sh major  # 0.1 -> 1.0 (build resets to 1)
```

### 3. Backend API Endpoint

**Endpoint**: `GET /api/version`

Returns comprehensive version information:

```json
{
  "name": "Tybalt Turbo",
  "version": "0.1",
  "build": 123,
  "fullVersion": "0.1.123",
  "gitCommit": "full-40-char-hash",
  "gitCommitShort": "abc1234",
  "gitBranch": "main",
  "buildTime": "2024-01-15T10:30:00Z"
}
```

### 4. UI Version Display

The version is displayed in the top-right corner of the application:

- **Desktop**: In the sidebar header
- **Mobile**: In the mobile header bar

Features:

- Compact display showing `v0.1.123`
- Click to expand detailed information
- Copy git commit hash functionality
- Fallback to build-time constants if API fails

## Deployment Integration

### Automatic Build Increment

The GitHub Actions workflow (`fly-deploy.yml`) automatically:

1. Increments the build number on each deployment
2. Commits the version change
3. Creates a git tag (e.g., `v0.1.123`)
4. Pushes the changes and tag

### Docker Build Process

The Dockerfile generates version files during build:

1. **UI Stage**: Generates `ui/src/lib/version.ts`
2. **Backend Stage**: Generates `app/constants/version.go`
3. Both include git commit information and build timestamp

## Manual Version Management

### For Feature Releases

```bash
# For new features (minor version)
./scripts/increment-version.sh minor
git add version.json
git commit -m "Release v0.2 - New features"
git tag v0.2.1
git push origin main --tags
```

### For Major Releases

```bash
# For breaking changes (major version)
./scripts/increment-version.sh major
git add version.json
git commit -m "Release v1.0 - Major release"
git tag v1.0.1
git push origin main --tags
```

### For Regular Deployments

```bash
# Regular deployments (automatic via GitHub Actions)
# Build number increments automatically: 0.1.123 -> 0.1.124
```

## Version Semantics

### **Major Version** (`X.y.build`)

- Breaking changes
- Major architectural changes
- API compatibility breaks

### **Minor Version** (`x.Y.build`)

- New features
- Enhancements
- Non-breaking changes
- Build resets to 1

### **Build Number** (`x.y.BUILD`)

- Every deployment
- Bug fixes
- Configuration changes
- Auto-incremented

## Version Tracking Benefits

1. **Deployment Tracking**: Each deployment has a unique build number
2. **Git Integration**: Full commit hash tracked for every version
3. **User Visibility**: Users can see exactly which version they're running
4. **Debugging**: Easy to correlate user reports with specific builds
5. **Simplified Workflow**: No confusion between patch vs build increments
6. **Clean Display**: Shorter version strings in UI

## Rollback Support

To rollback to a previous version:

```bash
# Find the desired version tag
git tag -l "v*"

# Reset to specific version
git checkout v0.1.120
# Update version.json to match
# Deploy normally
```

## Development vs Production

- **Development**: Version files are gitignored and generated locally
- **Production**: Version files generated during Docker build with actual git info
- **API Fallback**: UI gracefully handles API unavailability using build-time constants

## Monitoring

The version endpoint can be used for:

- Health checks
- Monitoring dashboards  
- Automated deployment verification
- User support (version identification)

## Future Enhancements

Potential additions:

- Changelog integration
- Release notes automation
- Version comparison APIs
- Deployment metrics correlation
