.PHONY: help build run worker test test-coverage lint fmt clean docker-build docker-up docker-down

help:
	@echo "Available commands:"
	@echo "  make build        - Build binary"
	@echo "  make run          - Run server"
	@echo "  make worker       - Run outbox worker"
	@echo "  make test         - Run all tests"
	@echo "  make test-coverage- Run with coverage"
	@echo "  make fmt          - Format code"
	@echo "  make clean        - Clean artifacts"
	@echo "  make docker-up    - Start Docker"
	@echo "  make docker-down  - Stop Docker"

build:
	go build -o bin/vk_backend cmd/main.go

run:
	go run cmd/main.go

worker:
	go run cmd/worker/main.go

test:
	go test -v ./...

test-coverage:
	go test -cover -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

fmt:
	go fmt ./...

clean:
	rm -rf bin/ coverage.out coverage.html

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down
