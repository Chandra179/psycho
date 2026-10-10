#!/bin/sh
# Assembles dist/ for Cloudflare Workers static assets. The wasm, script and
# stylesheet are copied under a name that contains a hash of their content
# (dist/a/<name>.<hash>.<ext>), so site/_headers can cache them for a year and a
# new deploy still reaches visitors: index.html is the only unhashed entry point
# and always revalidates.
set -eu

rm -rf dist
mkdir -p dist/a

GOOS=js GOARCH=wasm go build -ldflags="-s -w" -trimpath -o dist/a/psycho.wasm ./cmd/wasm
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" dist/a/wasm_exec.js
cp site/app.js site/index.html assets/app.css dist/a/ 2>/dev/null
mv dist/a/index.html dist/index.html
cp site/_headers site/og.png site/robots.txt site/sitemap.xml site/llms.txt dist/

# hashed NAME: renames dist/a/NAME to NAME-<first 10 hex of sha256>.EXT and prints the new name.
hashed() {
	base=${1%.*}
	ext=${1##*.}
	sum=$(sha256sum "dist/a/$1" | cut -c1-10)
	mv "dist/a/$1" "dist/a/$base-$sum.$ext"
	printf '%s-%s.%s' "$base" "$sum" "$ext"
}

wasm=$(hashed psycho.wasm)
exec_js=$(hashed wasm_exec.js)
css=$(hashed app.css)

# The wasm name goes into app.js before app.js itself is hashed.
sed -i "s|var WASM_URL = 'psycho.wasm';|var WASM_URL = 'a/$wasm';|" dist/a/app.js
app=$(hashed app.js)

sed -i \
	-e "s|href=\"app.css\"|href=\"a/$css\"|" \
	-e "s|src=\"wasm_exec.js\"|src=\"a/$exec_js\"|" \
	-e "s|src=\"app.js\"|src=\"a/$app\"|" \
	-e "s|href=\"psycho.wasm\"|href=\"a/$wasm\"|" \
	dist/index.html

grep -q "a/$wasm" dist/a/"$app" && grep -q "a/$wasm" dist/index.html || { echo "hash substitution failed" >&2; exit 1; }
