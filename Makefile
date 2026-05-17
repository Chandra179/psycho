vendor:
	go mod tidy && go mod vendor

up:
	docker compose up -d