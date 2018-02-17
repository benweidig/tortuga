# Project Settings
BINARY = tortuga
REPO   = github.com/benweidig/tortuga
HASH  := $(shell git rev-parse --short HEAD)
DATE  := $(shell date)
TAG  ?= $(shell git describe --tags --always --abbrev=0 --match="v[0-9]*.[0-9]*.[0-9]*" 2> /dev/null)
VERSION = $(shell sed 's/^.//' <<< "${TAG}")

# Go parameters
GOCMD    = $(shell which go)
GOCLEAN = $(GOCMD) clean
GOFMT    = $(GOCMD) fmt
GOGET    = $(GOCMD) get
GOLINT   = golint
GOTEST   = $(GOCMD) test
GOBUILD  = $(GOCMD) build

# Tasks
.PHONY: all
all: clean fmt deps lint test build

.PHONY: clean
clean:
	$(GOCLEAN)
	rm -rf build

.PHONY: fmt
fmt:
	$(GOFMT)

.PHONY: deps
deps:
	$(GOGET) -t -v ./...

.PHONY: lint
lint:
	$(GOGET) -u github.com/golang/lint/golint
	$(GOLINT) ./...

.PHONY: test
test:
	$(GOTEST)

.PHONY: build
build:
	$(GOBUILD) -ldflags "-X '${REPO}/version.CommitHash=${HASH}' -X '${REPO}/version.CompileDate=${DATE}'" -o build/${BINARY}

.PHONY: cross-compile
cross-compile:
	$(GOGET) github.com/mitchellh/gox
	gox -ldflags "-X '${REPO}/version.Version=${VERSION}' -X '${REPO}/version.CommitHash=${HASH}' -X '${REPO}/version.CompileDate=${DATE}'" --output="build/${BINARY}-${VERSION}-{{.OS}}_{{.Arch}}"

.PHONY: release
release: cross-compile
	echo "Test"

.PHONY: version
version:
	@echo $(VERSION)

.PHONY: tag
tag:
	@echo $(TAG)

