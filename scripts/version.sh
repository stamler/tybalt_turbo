#!/bin/sh

# Version generation script for Tybalt Turbo
# This script generates version information and outputs it in multiple formats

set -e

# Read version info from version.json
VERSION=$(jq -r '.version' version.json)
BUILD=$(jq -r '.build' version.json)
NAME=$(jq -r '.name' version.json)

# Get git information - prefer build args/env vars, fallback to git commands
GIT_COMMIT=${GIT_COMMIT:-$(git rev-parse HEAD 2>/dev/null || echo "unknown")}
GIT_COMMIT_SHORT=${GIT_COMMIT_SHORT:-$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")}
GIT_BRANCH=${GIT_BRANCH:-$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")}
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Create full version string
FULL_VERSION="${VERSION}.${BUILD}"

# Output version information in different formats based on first argument
case "${1:-all}" in
  "json")
    cat << EOF
{
  "name": "${NAME}",
  "version": "${VERSION}",
  "build": ${BUILD},
  "fullVersion": "${FULL_VERSION}",
  "gitCommit": "${GIT_COMMIT}",
  "gitCommitShort": "${GIT_COMMIT_SHORT}",
  "gitBranch": "${GIT_BRANCH}",
  "buildTime": "${BUILD_TIME}"
}
EOF
    ;;
  "env")
    echo "export TYBALT_VERSION='${VERSION}'"
    echo "export TYBALT_BUILD='${BUILD}'"
    echo "export TYBALT_FULL_VERSION='${FULL_VERSION}'"
    echo "export TYBALT_GIT_COMMIT='${GIT_COMMIT}'"
    echo "export TYBALT_GIT_COMMIT_SHORT='${GIT_COMMIT_SHORT}'"
    echo "export TYBALT_GIT_BRANCH='${GIT_BRANCH}'"
    echo "export TYBALT_BUILD_TIME='${BUILD_TIME}'"
    ;;
  "go")
    # Generate Go constants file
    cat << EOF
package constants

// Version information generated at build time
const (
	AppName        = "${NAME}"
	Version        = "${VERSION}"
	Build          = ${BUILD}
	FullVersion    = "${FULL_VERSION}"
	GitCommit      = "${GIT_COMMIT}"
	GitCommitShort = "${GIT_COMMIT_SHORT}"
	GitBranch      = "${GIT_BRANCH}"
	BuildTime      = "${BUILD_TIME}"
)
EOF
    ;;
  "ts")
    # Generate TypeScript constants file
    cat << EOF
// Version information generated at build time
export const VERSION_INFO = {
  name: '${NAME}',
  version: '${VERSION}',
  build: ${BUILD},
  fullVersion: '${FULL_VERSION}',
  gitCommit: '${GIT_COMMIT}',
  gitCommitShort: '${GIT_COMMIT_SHORT}',
  gitBranch: '${GIT_BRANCH}',
  buildTime: '${BUILD_TIME}'
} as const;

export type VersionInfo = typeof VERSION_INFO;
EOF
    ;;
  "short")
    echo "${FULL_VERSION}"
    ;;
  "full")
    echo "${FULL_VERSION} (${GIT_COMMIT_SHORT})"
    ;;
  *)
    echo "Tybalt Turbo v${FULL_VERSION}"
    echo "Build: ${BUILD}"
    echo "Git: ${GIT_COMMIT_SHORT} (${GIT_BRANCH})"
    echo "Built: ${BUILD_TIME}"
    ;;
esac 