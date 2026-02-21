APP_NAME := resysched
BIN_DIR := bin
GO := go

.PHONY: deps build test run clean podman-up podman-down user-add

deps:
	$(GO) mod tidy

build: deps
	mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/$(APP_NAME) ./cmd/resysched

test:
	$(GO) test ./...

run: build
	./$(BIN_DIR)/$(APP_NAME) server

user-add: build
	./$(BIN_DIR)/$(APP_NAME) user add --username "$(USERNAME)" --password "$(PASSWORD)"

podman-up:
	podman-compose up --build

podman-down:
	podman-compose down

clean:
	rm -rf $(BIN_DIR)
