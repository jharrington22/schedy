APP_NAME := resysched
BIN_DIR := bin

GO ?= go
IMAGE ?= localhost/$(APP_NAME):dev

.PHONY: help
help:
	@echo "Targets:"
	@echo "  make build           Build ./bin/resysched"
	@echo "  make test            Run go test"
	@echo "  make run             Run server locally (needs env vars)"
	@echo "  make keys            Generate cookie keys exports"
	@echo "  make user-add USERNAME=... PASSWORD=..."
	@echo "  make podman-up       podman-compose up --build"
	@echo "  make podman-down     podman-compose down -v"
	@echo "  make image           Build container image"
	@echo "  make deploy          kubectl apply -f deploy/k8s"
	@echo "  make deps            Fetch/tidy Go dependencies (writes go.sum)"

$(BIN_DIR):
	@mkdir -p $(BIN_DIR)

.PHONY: build
build: deps $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/$(APP_NAME) .	

.PHONY: test

test: deps
	$(GO) test ./...

.PHONY: run

run: deps
	$(GO) run . server --migrate=true

.PHONY: keys
keys: deps
	$(GO) run . keys

.PHONY: deps
deps:
	$(GO) mod tidy

.PHONY: user-add
user-add: build
	@if [ -z "$(USERNAME)" ] || [ -z "$(PASSWORD)" ]; then echo "USERNAME and PASSWORD required"; exit 1; fi
	./$(BIN_DIR)/$(APP_NAME) user add --username "$(USERNAME)" --password "$(PASSWORD)"

.PHONY: podman-up
podman-up:
	podman-compose up --build

.PHONY: podman-down
podman-down:
	podman-compose down -v

.PHONY: image
image:
	podman build -t $(IMAGE) -f Containerfile .

.PHONY: deploy
deploy:
	kubectl apply -f deploy/k8s

.PHONY: db/login
db/login:
	podman exec -it psql-cs bash -c "psql -h localhost -U $(USERNAME) resy"`
