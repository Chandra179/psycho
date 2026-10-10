assets:
	npm ci --ignore-scripts
	npm run build:assets

vendor:
	go mod tidy && go mod vendor

.PHONY: wasm serve-wasm deploy
# Browser-only build: dist/ is a static site (psycho.wasm plus its page) that
# analyzes text on the visitor's device. Serve it over http, e.g. `make serve-wasm`.
wasm:
	mkdir -p dist
	GOOS=js GOARCH=wasm go build -ldflags="-s -w" -trimpath -o dist/psycho.wasm ./cmd/wasm
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" dist/wasm_exec.js
	cp site/index.html site/app.js site/_headers assets/app.css dist/

serve-wasm: wasm
	cd dist && python3 -m http.server 8081

# Publishes dist/ to Cloudflare Workers (static assets; see wrangler.jsonc). Needs `npx wrangler login` once.
deploy: wasm
	npx wrangler deploy

build:
	go build ./...

test:
	go test ./... -v

# Offline experiment; corpus and model artifacts stay local and gitignored.
train:
	go run ./cmd/train -csv corpus-eval/essays.csv -out testresults/supervised
