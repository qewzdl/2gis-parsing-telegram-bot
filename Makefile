.PHONY: run build docker-up docker-down tidy

# Run locally
run:
	go run ./cmd/bot

# Build binary
build:
	CGO_ENABLED=1 go build -o bin/parser ./cmd/bot

# Download dependencies
tidy:
	go mod tidy

# Docker
docker-up:
	docker-compose up --build -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f bot
