.PHONY: build run test lint migrate sqlc

build:
	go build -o bin/api ./cmd/api
	go build -o bin/worker ./cmd/worker

run:
	go run ./cmd/api

test:
	go test ./... -v

test-coverage:
	go test ./... -v -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run ./...

migrate:
	psql "$$DATABASE_URL" -f db/migrations/001_initial.sql

sqlc:
	sqlc generate

setup: migrate sqlc build
