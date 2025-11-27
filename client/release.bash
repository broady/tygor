#!/bin/bash
set -e

# Always run from the script's directory (client/)
cd "$(dirname "$0")"

# Check that an argument is provided
if [[ $# -eq 0 ]]; then
  echo "Error: No version bump type specified"
  echo "Usage: $0 <major|minor|patch>"
  exit 1
fi

VERSION_TYPE=$1

# Validate version type
if [[ ! "${VERSION_TYPE}" =~ ^(major|minor|patch)$ ]]; then
  echo "Error: Invalid version type '${VERSION_TYPE}'"
  echo "Usage: $0 <major|minor|patch>"
  exit 1
fi

# Check that working directory is clean
# shellcheck disable=SC2312
if [[ -n "$(git status --porcelain)" ]]; then
  echo "Error: Working directory is not clean"
  echo "Please commit or stash your changes before releasing"
  git status --short
  exit 1
fi

# Show current version
CURRENT_VERSION=$(jq -r .version package.json)
echo "Current version: ${CURRENT_VERSION}"

# Bump version
npm version "${VERSION_TYPE}" --no-git-tag-version

# Show new version
NEW_VERSION=$(jq -r .version package.json)
TAG="v${NEW_VERSION}"
echo "New version: ${NEW_VERSION}"

# Update example package.json files to use new version
echo "Updating example package.json files..."
for example_pkg in ../examples/*/client/package.json; do
  if [[ -f "${example_pkg}" ]]; then
    # Update @tygor/client dependency to ^NEW_VERSION
    jq --arg v "^${NEW_VERSION}" '.dependencies["@tygor/client"] = $v' "${example_pkg}" > "${example_pkg}.tmp"
    mv "${example_pkg}.tmp" "${example_pkg}"
    echo "  Updated ${example_pkg}"
  fi
done

# Confirmation
read -p "Ready to release ${TAG}? (y/N) " -n 1 -r
echo
if [[ ! ${REPLY} =~ ^[Yy]$ ]]; then
  echo "Aborted. Restoring package.json..."
  git checkout package.json
  exit 1
fi

# Commit, tag, and publish
git commit -a -m "Release ${TAG}"
git tag "${TAG}"

# Set browser to echo so it prints the URL instead of trying to open it
npm --browser=echo publish

echo "Successfully released ${TAG}. Run git push && git push --tags"
