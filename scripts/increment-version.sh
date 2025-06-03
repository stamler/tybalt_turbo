#!/bin/bash

# Script to increment version numbers for Tybalt Turbo
# Usage: 
#   ./increment-version.sh build     # increment build number only
#   ./increment-version.sh minor     # increment minor version and reset build to 1
#   ./increment-version.sh major     # increment major version, reset minor to 0 and build to 1

set -e

VERSION_FILE="version.json"

if [ ! -f "$VERSION_FILE" ]; then
    echo "Error: $VERSION_FILE not found"
    exit 1
fi

# Read current version
CURRENT_VERSION=$(jq -r '.version' "$VERSION_FILE")
CURRENT_BUILD=$(jq -r '.build' "$VERSION_FILE")
NAME=$(jq -r '.name' "$VERSION_FILE")

# Parse semantic version (major.minor format)
IFS='.' read -r MAJOR MINOR <<< "$CURRENT_VERSION"

case "${1:-build}" in
    "major")
        MAJOR=$((MAJOR + 1))
        MINOR=0
        BUILD=1
        echo "Incrementing major version: $CURRENT_VERSION -> $MAJOR.$MINOR (build $BUILD)"
        ;;
    "minor")
        MINOR=$((MINOR + 1))
        BUILD=1
        echo "Incrementing minor version: $CURRENT_VERSION -> $MAJOR.$MINOR (build $BUILD)"
        ;;
    "build")
        BUILD=$((CURRENT_BUILD + 1))
        echo "Incrementing build number: $CURRENT_VERSION (build $CURRENT_BUILD -> build $BUILD)"
        ;;
    *)
        echo "Usage: $0 [major|minor|build]"
        echo "  major - increment major version (x.0.1)"
        echo "  minor - increment minor version (0.x.1)"
        echo "  build - increment build number (0.0.x) [default]"
        exit 1
        ;;
esac

NEW_VERSION="$MAJOR.$MINOR"

# Update version.json
jq --arg version "$NEW_VERSION" --argjson build "$BUILD" --arg name "$NAME" \
   '. | .version = $version | .build = $build | .name = $name' \
   "$VERSION_FILE" > "${VERSION_FILE}.tmp" && mv "${VERSION_FILE}.tmp" "$VERSION_FILE"

echo "Updated $VERSION_FILE:"
echo "  Version: $NEW_VERSION"
echo "  Build: $BUILD"
echo "  Full: $NEW_VERSION.$BUILD"

# If we're in a git repo, suggest tagging
if git rev-parse --git-dir > /dev/null 2>&1; then
    echo ""
    echo "Suggested git commands:"
    echo "  git add $VERSION_FILE"
    echo "  git commit -m \"Bump version to $NEW_VERSION.$BUILD\""
    echo "  git tag v$NEW_VERSION.$BUILD"
fi 