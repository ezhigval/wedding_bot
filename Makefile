GO ?= go
NPM ?= npm
WEBAPP_DIR := webapp-react
GO_PACKAGES := ./cmd/... ./internal/...

.PHONY: deps frontend-deps vet test test-race lint build frontend-build verify run docker-build

deps:
	$(GO) mod download
	$(MAKE) frontend-deps

frontend-deps:
	cd $(WEBAPP_DIR) && $(NPM) ci

vet:
	$(GO) vet $(GO_PACKAGES)

test:
	$(GO) test $(GO_PACKAGES)

test-race:
	$(GO) test -race $(GO_PACKAGES)

lint:
	cd $(WEBAPP_DIR) && $(NPM) run lint

build:
	$(GO) build ./cmd/server

frontend-build:
	cd $(WEBAPP_DIR) && $(NPM) run build

verify: vet test lint build frontend-build

run:
	$(GO) run ./cmd/server

docker-build:
	docker build -t wedding-bot .
