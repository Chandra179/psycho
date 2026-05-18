vendor:
	go mod tidy && go mod vendor

build:
	go build ./...

run:
	go run ./cmd/psycho/

test:
	go test ./... -v

test-curl:
	curl -s -X POST http://localhost:8080/analyze-dir \
		-H "Content-Type: application/json" \
		-d '{"source_type": "file"}' | jq .

up:
	docker compose up -d