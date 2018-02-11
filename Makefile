# Project Settings
BINARY   = tortuga
VERSION ?= $(shell git describe --tags --always --abbrev=0 --match="[0-9]*.[0-9]*.[0-9]*" 2> /dev/null)

# Go parameters
GOCMD    = go
GOGET    = $(GOCMD) get
GOBUILD  = $(GOCMD) build
GOCLEAN  = $(GOCMD) clean
GOTEST   = $(GOCMD) test
GOFMT    = $(GOCMD) fmt

# Tasks
.PHONY: all
all: clean fmt test build

.PHONY: clean
clean:
	$(GOCLEAN)
	rm -f ${BINARY}
	rm -rf build

.PHONY: fmt
fmt:
	$(GOFMT)

.PHONY: test
test:
	$(GOTEST)

.PHONY: build
build:
	$(GOBUILD) -o ${BINARY}

.PHONY: release
release:
	$(GOGET) github.com/mitchellh/gox
	gox --output="build/${BINARY_NAME}_${VERSION}_{{.Dir}}_{{.OS}}_{{.Arch}}"

.PHONY: version
version:
	@echo $(VERSION)
