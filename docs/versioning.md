# Tybalt Turbo Versioning System

## Overview

Tybalt Turbo uses a comprehensive versioning system with automatic build incrementing, git commit tracking, and UI display. The system uses a simplified 2-tier approach:

- **Production**: Displays version from backend API (real git info, build numbers)
- **Development**: Displays version from `version.json` + ".dev" suffix

## Architecture

### Source of Truth: `version.json`

```json
{
  "name": "Tybalt Turbo",
  "version": "0.1",
  "build": 4
}
```

### Components

1. **Version Scripts**
   - `scripts/version.sh` - Multi-format version generation (JSON, Go, environment)
   - `scripts/increment-version.sh` - Version management (increment major/minor/build)

2. **Backend Integration**
   - `/api/version` endpoint serves version info with real git data
   - Go constants generated at build time from `version.sh go`

3. **UI Display**
   - `VersionInfo.svelte` component shows version in top-right corner
   - Production: Fetches from `/api/version`
   - Development: Reads `/version.json` and shows `v0.1.dev`

4. **Deployment Integration**
   - GitHub Actions automatically increments build number
   - Git tagging with `v0.1.4` format
   - Docker builds with version info baked in

## Development Workflow

### Local Development

```bash
# Start development - automatically copies version.json and shows "v0.1.dev"
npm run dev
```

The `npm run dev` command automatically:

1. Copies `version.json` to `ui/static/version.json`
2. Starts the development server
3. UI displays current version with ".dev" suffix

### Manual Version Changes

```bash
# Increment build number
./scripts/increment-version.sh build

# Increment minor version (resets build to 1)
./scripts/increment-version.sh minor

# Increment major version (resets minor and build)
./scripts/increment-version.sh major
```

### Deployment

GitHub Actions automatically:

1. Increments build number
2. Commits version changes
3. Tags release with `v0.1.4`
4. Pushes changes back to repo
5. Deploys to Fly.io with version info

## Version Display

**Format**: `v{version}.{build}` or `v{version}.dev`

**Examples**:

- Production: `v0.1.4` (shows build number)
- Development: `v0.1.dev` (shows dev suffix)

**UI Behavior**:

- Clickable button in top-right corner
- Expandable details popup with git info, build time
- Copy commit hash functionality
- Responsive positioning (mobile/desktop)

## File Structure

```
version.json                 # Source of truth
scripts/
  ├── version.sh            # Multi-format generation
  └── increment-version.sh  # Version management
ui/
  ├── static/version.json   # Auto-generated copy (ignored by git)
  └── src/lib/components/
      └── VersionInfo.svelte # UI component
app/
  ├── constants/version.go  # Generated Go constants
  └── handlers/version.go   # API endpoint
docs/versioning.md          # This documentation
```

## Key Design Principles

1. **Single Source of Truth**: All version info derives from `version.json`
2. **No Fake Fallbacks**: System shows errors rather than fake version numbers
3. **Environment-Specific**: Production uses API, development uses static file
4. **Automatic Deployment**: Build numbers increment automatically on deploy
5. **Git Integration**: Full commit tracking and branch information
