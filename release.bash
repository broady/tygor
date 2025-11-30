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

# Show current versions
CLIENT_VERSION=$(jq -r .version client/package.json)
VITE_PLUGIN_VERSION=$(jq -r .version vite-plugin/package.json)
echo "Current versions:"
echo "  @tygor/client:      ${CLIENT_VERSION}"
echo "  @tygor/vite-plugin: ${VITE_PLUGIN_VERSION}"

if [[ "${CLIENT_VERSION}" != "${VITE_PLUGIN_VERSION}" ]]; then
  echo ""
  echo "Warning: Versions are out of sync. They will be synchronized."
fi

# Bump version in client (this is the source of truth)
cd client
npm version "${VERSION_TYPE}" --no-git-tag-version
NEW_VERSION=$(jq -r .version package.json)
cd ..

TAG="v${NEW_VERSION}"
echo ""
echo "New version: ${NEW_VERSION}"

# Update vite-plugin version and @tygor/client dependency
echo ""
echo "Updating vite-plugin/package.json..."
jq --arg v "${NEW_VERSION}" --arg dep "^${NEW_VERSION}" \
  '.version = $v | .dependencies["@tygor/client"] = $dep' \
  vite-plugin/package.json > vite-plugin/package.json.tmp
mv vite-plugin/package.json.tmp vite-plugin/package.json

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

# Confirmation
echo ""
read -p "Ready to release ${TAG}? (y/N) " -n 1 -r
echo
if [[ ! ${REPLY} =~ ^[Yy]$ ]]; then
  echo "Aborted. Restoring changes..."
  git checkout client/package.json vite-plugin/package.json examples/
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
bun publish
cd ..

echo ""
echo "Successfully released ${TAG}. Run: git push && git push --tags"
