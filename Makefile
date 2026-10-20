.PHONY: all build run test lint clean docker-up docker-down migrate-up migrate-down

BINARY_NAME=bottagond
BUILD_DIR=bin
GO_FILES=$(shell find . -name '*.go')

all: build

build:
	@mkdir -p $(BUILD_DIR)
	go build -v -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/bottagond

run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

test:
	go test -v -race -cover ./...

lint:
	golangci-lint run ./... || go vet ./...

clean:
	rm -rf $(BUILD_DIR)

docker-up:
	docker compose up -d

docker-down:
	docker compose down

migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down
