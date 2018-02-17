# Project Settings
NAME = tortuga
BINARY = tt
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
	gox -ldflags "-X '${REPO}/version.Version=${VERSION}' -X '${REPO}/version.CommitHash=${HASH}' -X '${REPO}/version.CompileDate=${DATE}'" --output="build/${NAME}-${VERSION}-{{.OS}}_{{.Arch}}"

.PHONY: compress
compress:
	for i in ./build/*; do gzip $$i; done

.PHONY: release
release: clean cross-compile compress deb

.PHONY: version
version:
	@echo $(VERSION)

.PHONY: tag
tag:
	@echo $(TAG)

.PHONY: deb
deb:
	# Prepare
	mkdir -p build/deb/usr/bin/ build/deb/DEBIAN/
	cp deb-control-template build/deb/DEBIAN/control
	sed -i 's/PKG_NAME/${BINARY}/g' build/deb/DEBIAN/control
	sed -i 's/PKG_VERSION/${VERSION}/g' build/deb/DEBIAN/control
	# i386
	sed -i 's/ARCH/i386/g' build/deb/DEBIAN/control
	GOOS=linux GOARCH=386 ${GOBUILD} -ldflags "-X '${REPO}/version.Version=${VERSION}' -X '${REPO}/version.CommitHash=${HASH}' -X '${REPO}/version.CompileDate=${DATE}'" -o build/deb/usr/bin/${BINARY}
	dpkg-deb --build build/deb build/${NAME}-${VERSION}-linux_386.deb
	# amd64
	sed -i 's/ARCH/amd64/g' build/deb/DEBIAN/control
	GOOS=linux GOARCH=amd64 ${GOBUILD} -ldflags "-X '${REPO}/version.Version=${VERSION}' -X '${REPO}/version.CommitHash=${HASH}' -X '${REPO}/version.CompileDate=${DATE}'" -o build/deb/usr/bin/${BINARY}
	dpkg-deb --build build/deb build/${NAME}-${VERSION}-linux_amd64.deb
	# Cleanup
	rm -r build/deb

