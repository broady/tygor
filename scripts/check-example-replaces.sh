#!/bin/bash
# Checks that example go.mod files:
# 1. Don't contain replace directives (break when cloned standalone)
# 2. Use the current version from VERSION file

set -e

cd "$(git rev-parse --show-toplevel)"

VERSION=$(cat VERSION)
errors=0

for modfile in $(find examples -name go.mod); do
    # Check for replace directives
    if grep -q '^replace ' "$modfile" 2>/dev/null; then
        echo "ERROR: $modfile contains a replace directive"
        grep '^replace ' "$modfile" | sed 's/^/  /'
        errors=$((errors + 1))
    fi

    # Check version matches VERSION file
    if ! grep -q "github.com/broady/tygor v$VERSION" "$modfile" 2>/dev/null; then
        actual=$(grep 'github.com/broady/tygor v' "$modfile" | head -1 | grep -o 'v[0-9.]*' || echo "not found")
        echo "ERROR: $modfile has tygor $actual, expected v$VERSION"
        errors=$((errors + 1))
    fi
done

if [ $errors -gt 0 ]; then
    echo ""
    echo "Examples must use published version v$VERSION (from VERSION file)."
    echo "Run: go mod edit -require github.com/broady/tygor@v$VERSION"
    exit 1
fi
