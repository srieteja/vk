# Setup Guide

## Quick Start

1. Extract the ZIP
2. cd enterprise-api
3. cp .env.example .env
4. go mod download
5. docker-compose up -d postgres
6. sleep 5
7. psql -U enterprise_user -d enterprise_db -h localhost < migrations/init.sql
8. go run cmd/main.go

Server: http://localhost:8080

## Test

curl http://localhost:8080/health

## Run Tests

go test -v ./...