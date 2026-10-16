# AIProJsonMD
#
# Standard entry points: deps, build, install, test, lint, default, all.
# Run `make` or `make help` to list every target.

APP_NAME ?= AIProJsonMD
GO ?= go
GOFLAGS ?=
GOVULNCHECK_VERSION ?= v1.9.0
OUT_DIR ?= out_markdown
ASSETS_DIR ?= assets
FORMAT ?= md
CHANGELOG_REF ?= main

.DEFAULT_GOAL := help
.PHONY: help default all deps fmt fmt/check vet lint lint/shell test test/race test/cover vuln build install run clean changelog/status

## Build the binary (alias for build)
default: build

## Run the full local pipeline: format check, vet, lint, shell lint, race tests, build
all: fmt/check vet lint lint/shell test/race build

## Download and verify module dependencies
deps:
	$(GO) mod download
	$(GO) mod verify

## Format all Go source files in place
fmt:
	$(GO) fmt ./...

## Fail if any file is not gofmt-formatted
fmt/check:
	@test -z "$$(gofmt -l .)" || { gofmt -l .; echo "run 'make fmt' to fix" >&2; exit 1; }

## Run go vet
vet:
	$(GO) vet ./...

## Run golangci-lint
lint:
	golangci-lint run ./...

## Lint the shell scripts in scripts/ with shellcheck
lint/shell:
	@command -v shellcheck >/dev/null || { echo "shellcheck is not installed" >&2; exit 1; }
	shellcheck scripts/*.sh

## Show the PR number, merge time, and commits of each CHANGELOG.md entry, read from git (CHANGELOG_REF=main)
changelog/status:
	@scripts/changelog-status.sh -r "$(CHANGELOG_REF)" -t

## Run unit tests
test: deps
	$(GO) test ./...

## Run unit tests with the race detector
test/race: deps
	$(GO) test -race ./...

## Run tests with coverage and print the total
test/cover: deps
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out | tail -1

## Scan dependencies and the standard library with a pinned govulncheck (GOVULNCHECK_VERSION)
vuln:
	$(GO) run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...

## Build the binary
build: deps
	$(GO) build $(GOFLAGS) -o $(APP_NAME) .

## Install the binary into GOBIN (or GOPATH/bin)
install: deps
	$(GO) install $(GOFLAGS) .

## Run the converter: make run IN=<export.json> [FORMAT=md|html|both] [OUT_DIR=out_markdown] [ASSETS_DIR=assets]
run: deps
	@test -n "$(IN)" || { echo "Usage: make run IN=<export.json> [FORMAT=$(FORMAT)] [OUT_DIR=$(OUT_DIR)] [ASSETS_DIR=$(ASSETS_DIR)]" >&2; exit 1; }
	$(GO) run $(GOFLAGS) . -in "$(IN)" -out "$(OUT_DIR)" -assets "$(ASSETS_DIR)" -format "$(FORMAT)"

## Remove build artifacts
clean:
	rm -f $(APP_NAME) coverage.out

## This help screen
help:
	@printf "Available targets:\n\n"
	@awk '/^[a-zA-Z\-_0-9%:\\\/]+/ { \
		helpMessage = match(lastLine, /^## (.*)/); \
		if (helpMessage) { \
			helpCommand = $$1; \
			helpMessage = substr(lastLine, RSTART + 3, RLENGTH); \
			gsub("\\\\", "", helpCommand); \
			gsub(":+$$", "", helpCommand); \
			printf "  \x1b[32;01m%-35s\x1b[0m %s\n", helpCommand, helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST) | sort -u
	@printf "\n"
