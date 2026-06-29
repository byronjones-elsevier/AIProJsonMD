APP_NAME := aichatjsonmd
GO := go
OUT_DIR ?= out_markdown
ASSETS_DIR ?= assets

.PHONY: help fmt vet test build run clean

help:
	@echo "Targets:"
	@echo "  make fmt                      - Run gofmt on all Go files"
	@echo "  make vet                      - Run go vet"
	@echo "  make test                     - Run unit tests"
	@echo "  make build                    - Build the binary"
	@echo "  make run IN=<export.json>     - Run converter"
	@echo "  make clean                    - Remove binary"

fmt:
	@$(GO) fmt ./...

vet:
	@$(GO) vet ./...

test:
	@$(GO) test ./...

build:
	@$(GO) build -o $(APP_NAME) .

run:
	@if [ -z "$(IN)" ]; then \
		echo "Usage: make run IN=<export.json> [OUT_DIR=out_markdown] [ASSETS_DIR=assets]"; \
		exit 1; \
	fi
	@$(GO) run . -in "$(IN)" -out "$(OUT_DIR)" -assets "$(ASSETS_DIR)"

clean:
	@rm -f $(APP_NAME)
