# Project Settings
BINARY   = tt
VERSION ?= $(shell git describe --tags --always --abbrev=0 --match="[0-9]*.[0-9]*.[0-9]*" 2> /dev/null)

# Go parameters
GOCMD    = go
GOCLEAN  = $(GOCMD) clean
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
	go get -t -v ./...

.PHONY: lint
lint:
	go get -u github.com/golang/lint/golint
	golint ./...

.PHONY: test
test:
	$(GOTEST)

.PHONY: build
build:
	$(GOBUILD) -o ${BINARY}

.PHONY: release
release:
	$(GOGET) github.com/mitchellh/gox
	gox --output="build/${BINARY}_${VERSION}_{{.OS}}_{{.Arch}}"

.PHONY: version
version:
	@echo $(VERSION)
