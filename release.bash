#!/bin/bash
set -e

# Always run from the script's directory (repo root)
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

# Read current version from VERSION file (single source of truth)
CURRENT_VERSION=$(tr -d '[:space:]' < VERSION)
echo "Current version: ${CURRENT_VERSION}"

# Calculate new version
IFS='.' read -r MAJOR MINOR PATCH <<< "${CURRENT_VERSION}"
case "${VERSION_TYPE}" in
  major) NEW_VERSION="$((MAJOR + 1)).0.0" ;;
  minor) NEW_VERSION="${MAJOR}.$((MINOR + 1)).0" ;;
  patch) NEW_VERSION="${MAJOR}.${MINOR}.$((PATCH + 1))" ;;
esac

TAG="v${NEW_VERSION}"
echo "New version: ${NEW_VERSION}"

# Update VERSION file
echo "${NEW_VERSION}" > VERSION

# Update degit version in README.md
sed -i "s|broady/tygor/examples/react#v[0-9.]*|broady/tygor/examples/react#v${NEW_VERSION}|g" README.md
echo "  Updated README.md degit version"

# Update all package.json files
echo ""
echo "Updating package versions..."

# client/package.json
jq --arg v "${NEW_VERSION}" '.version = $v' \
  client/package.json > client/package.json.tmp
mv client/package.json.tmp client/package.json
echo "  Updated client/package.json"

# vite-plugin/package.json (version only, keep workspace:* for local dev)
jq --arg v "${NEW_VERSION}" '.version = $v' \
  vite-plugin/package.json > vite-plugin/package.json.tmp
mv vite-plugin/package.json.tmp vite-plugin/package.json
echo "  Updated vite-plugin/package.json"

# Update example package.json files
echo "Updating example package.json files..."
for example_pkg in examples/*/client/package.json; do
  if [[ -f "${example_pkg}" ]]; then
    updated=false
    tmp_file="${example_pkg}.tmp"
    cp "${example_pkg}" "${tmp_file}"

    # Update @tygor/client if present (and not a file: reference)
    if jq -e '.dependencies["@tygor/client"] // empty | startswith("^")' "${example_pkg}" > /dev/null 2>&1; then
      jq --arg v "^${NEW_VERSION}" '.dependencies["@tygor/client"] = $v' "${tmp_file}" > "${tmp_file}.2"
      mv "${tmp_file}.2" "${tmp_file}"
      updated=true
    fi

    # Update @tygor/vite-plugin if present (and not a file: reference)
    if jq -e '.devDependencies["@tygor/vite-plugin"] // empty | startswith("^")' "${example_pkg}" > /dev/null 2>&1; then
      jq --arg v "^${NEW_VERSION}" '.devDependencies["@tygor/vite-plugin"] = $v' "${tmp_file}" > "${tmp_file}.2"
      mv "${tmp_file}.2" "${tmp_file}"
      updated=true
    fi

    if [[ "${updated}" == "true" ]]; then
      mv "${tmp_file}" "${example_pkg}"
      echo "  Updated ${example_pkg}"
    else
      rm "${tmp_file}"
    fi
  fi
done

# Dry-run builds to catch errors before committing
echo ""
echo "Running dry-run builds..."
(cd client && bun publish --dry-run)
(cd vite-plugin && bun install && bun publish --dry-run)
echo "Dry-run builds passed."

# Confirmation
echo ""
read -p "Ready to release ${TAG}? (y/N) " -n 1 -r
echo
if [[ ! ${REPLY} =~ ^[Yy]$ ]]; then
  echo "Aborted. Restoring changes..."
  git checkout VERSION client/package.json vite-plugin/package.json examples/
  exit 1
fi

# Commit and tag
git commit -a -m "Release ${TAG}"
git tag "${TAG}"

# Publish both packages (use bun to resolve workspace:* dependencies)
echo ""
echo "Publishing @tygor/client..."
cd client
bun publish
cd ..

echo ""
echo "Publishing @tygor/vite-plugin..."
cd vite-plugin
bun install
bun publish
cd ..

echo ""
echo "Successfully released ${TAG}. Run: git push && git push --tags"
