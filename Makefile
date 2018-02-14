# Project Settings
BINARY = tt
REPO   = github.com/benweidig/tortuga
HASH  := $(shell git rev-parse --short HEAD)
DATE  := $(shell date)
TAG  ?= $(shell git describe --tags --always --abbrev=0 --match="[0-9]*.[0-9]*.[0-9]*" 2> /dev/null)

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
	rm -f ${BINARY}
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
	$(GOBUILD) -ldflags "-X '${REPO}/version.CommitHash=${HASH}' -X '${REPO}/version.CompileDate=${DATE}'" -o ${BINARY}

.PHONY: release
release:
	$(GOGET) github.com/mitchellh/gox
	gox --output="build/${BINARY}_${TAG}_{{.OS}}_{{.Arch}}"

.PHONY: version
version:
	@echo $(VERSION)
