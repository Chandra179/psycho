#!/usr/bin/env bash
set -euo pipefail

usage() {
	echo "Usage: $0 <new-module-name>"
	echo "Example: $0 github.com/me/myproject"
	echo ""
	echo "Renames the Go module from 'brook' to <new-module-name>."
	echo "Updates go.mod and all import paths in .go files."
	exit 1
}

if [ $# -ne 1 ]; then
	usage
fi

old="brook"
new="$1"

if [ "$old" = "$new" ]; then
	echo "Old and new module names are identical. Nothing to do."
	exit 0
fi

echo "Renaming module '$old' -> '$new'"

# 1. Update go.mod module line
sed -i "s/^module $old$/module $new/" go.mod
echo "  updated go.mod"

# 2. Update import paths in all .go files
find . -name '*.go' -exec sed -i "s|\"$old/|\"$new/|g" {} +
echo "  updated import paths in .go files"

# 3. Also check .proto files if any
if find . -name '*.proto' -print0 | xargs -0 grep -l "$old/" 2>/dev/null; then
	find . -name '*.proto' -exec sed -i "s|$old/|$new/|g" {} +
	echo "  updated .proto files"
fi

echo "Done. Run 'go build ./...' to verify."
