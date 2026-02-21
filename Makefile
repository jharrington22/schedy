APP_NAME := resysched
BIN_DIR := bin
GO := go

.PHONY: build clean test deps run keys

deps:
	$(GO) mod tidy

build: deps
	mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/$(APP_NAME) ./cmd/resysched

run: build
	./$(BIN_DIR)/$(APP_NAME) server

test:
	$(GO) test ./...

clean:
	rm -rf $(BIN_DIR)

keys: build
	./$(BIN_DIR)/$(APP_NAME) keys
