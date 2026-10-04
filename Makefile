assets:
	npm ci --ignore-scripts
	npm run build:assets

vendor:
	go mod tidy && go mod vendor

build:
	go build ./...

run:
	go run ./cmd/psycho/

test:
	go test ./... -v

# Offline experiment; corpus and model artifacts stay local and gitignored.
train:
	go run ./cmd/train -csv corpus-eval/essays.csv -out testresults/supervised

test-curl:
	@TMP=$$(mktemp); \
	curl -s -X POST http://localhost:8080/analyze-dir \
		-H "Content-Type: application/json" \
		-d '{"source_type": "file"}' > $$TMP; \
	jq . $$TMP; \
	ID=$$(jq -r .analysis_id $$TMP); \
	rm $$TMP; \
	echo ""; \
	echo "--> analysis_id: $$ID"; \
	echo "--> make pdf ID=$$ID"

# Usage: make pdf ID=<analysis_id>
pdf:
	@[ -n "$(ID)" ] || { echo "Usage: make pdf ID=<analysis_id>"; exit 1; }
	curl -s -o "profile-$(ID).pdf" \
		-X GET http://localhost:8080/analysis/$(ID)/pdf

# Runs test-curl, downloads the resulting PDF, and renders the single HTML
# report (profile-report.html) from the same analysis.
test-pdf:
	@TMP=$$(mktemp); \
	curl -s -X POST http://localhost:8080/analyze-dir \
		-H "Content-Type: application/json" \
		-d '{"source_type": "file"}' > $$TMP; \
	jq . $$TMP; \
	ID=$$(jq -r .analysis_id $$TMP); \
	if [ -z "$$ID" ] || [ "$$ID" = "null" ]; then \
		echo "--> analyze-dir failed, no analysis_id"; rm $$TMP; exit 1; \
	fi; \
	echo "--> analysis_id: $$ID"; \
	curl -s -o "profile-$$ID.pdf" -X GET http://localhost:8080/analysis/$$ID/pdf; \
	echo "--> wrote profile-$$ID.pdf"; \
	go run ./cmd/rendertemplates < $$TMP; \
	rm $$TMP; \
	echo "--> wrote profile-report.html"

image:
	podman build -t psycho .

up: image
	podman run -d --name psycho -p 8080:8080 psycho

down:
	-podman stop psycho
	-podman rm psycho
