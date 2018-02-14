# Project Settings
BINARY   = tt
PATH     = github.com/benweidig/tortuga
HASH    := $(shell git rev-parse --short HEAD)
DATE    := $(shell date)

# Go parameters
GOCMD    = $(shell which go)
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
	$(GOBUILD) -ldflags "-X '${PATH}/version.CommitHash=${HASH}' -X '${PATH}/version.CompileDate=${DATE}'" -o ${BINARY}

.PHONY: release
release:
	$(GOGET) github.com/mitchellh/gox
	gox --output="build/${BINARY}_${TAG}_{{.OS}}_{{.Arch}}"

.PHONY: version
version:
	@echo $(VERSION)
