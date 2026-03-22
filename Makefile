BINARY      := tt
REPO        := github.com/benweidig/tortuga
REPO_URL    := https://$(REPO)
PROJECT     := tortuga
DESC        := CLI tool for fetching/pushing/rebasing multiple git repositories at once
MAINTAINER  := Ben Weidig <ben+tortuga@netzgut.net>

TMPL_DIR    := packaging

HASH    := $(shell git rev-parse --short HEAD 2>/dev/null || echo "n/a")
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
TAG     := $(shell git describe --tags --abbrev=0 --match="v[0-9]*.[0-9]*.[0-9]*" 2>/dev/null)
VERSION := $(shell echo "$(TAG)" | sed 's/^v//')

BASE_BUILD_FOLDER := build
VERSION_FOLDER    := $(PROJECT)-$(VERSION)
BUILD_FOLDER      := $(BASE_BUILD_FOLDER)/$(VERSION_FOLDER)
RELEASE_FOLDER    := release/$(PROJECT)-$(VERSION)

LDFLAGS_DEV     := -ldflags "-X '$(REPO)/version.CommitHash=$(HASH)' -X '$(REPO)/version.CompileDate=$(DATE)'"
LDFLAGS_RELEASE := -ldflags "-X '$(REPO)/version.Version=$(VERSION)' -X '$(REPO)/version.CommitHash=$(HASH)' -X '$(REPO)/version.CompileDate=$(DATE)' -s -w"

# ##############################################################################

.PHONY: all
all: clean fmt vet test staticcheck build

.PHONY: clean
clean:
	go clean
	rm -rf build/ release/ $(BINARY)

# ##############################################################################

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: staticcheck
staticcheck:
	@if command -v staticcheck >/dev/null 2>&1; then \
		staticcheck ./...; \
	else \
		echo "staticcheck not found, skipping (install with: go install honnef.co/go/tools/cmd/staticcheck@latest)"; \
	fi

# ##############################################################################

.PHONY: test
test:
	go test ./...

.PHONY: test-verbose
test-verbose:
	go test -v ./...

.PHONY: test-race
test-race:
	go test -race ./...

# ##############################################################################

.PHONY: build
build:
	go build $(LDFLAGS_DEV) -o $(BINARY) .

.PHONY: install
install:
	go install $(LDFLAGS_DEV) .

.PHONY: version
version:
	@echo "Version: $(VERSION)"

.PHONY: tag
tag:
	@echo "Tag: $(TAG)"

# ##############################################################################

# build-tar: cross-compile and package one binary into a tar.gz.
# Usage: $(call build-tar,<GOOS>,<GOARCH>)
define build-tar
	CGO_ENABLED=0 GOOS=$(1) GOARCH=$(2) go build $(LDFLAGS_RELEASE) -o $(BUILD_FOLDER)/$(BINARY)
	tar --exclude $(VERSION_FOLDER)/deb \
	    -czf $(RELEASE_FOLDER)/$(VERSION_FOLDER)_$(1)_$(2).tar.gz \
	    -C $(BASE_BUILD_FOLDER) $(VERSION_FOLDER)
	rm $(BUILD_FOLDER)/$(BINARY)
endef

# build-deb: cross-compile and package one Linux binary into a tar.gz + .deb.
# Usage: $(call build-deb,<GOARCH>,<DEB_ARCH>)
define build-deb
	CGO_ENABLED=0 GOOS=linux GOARCH=$(1) go build $(LDFLAGS_RELEASE) -o $(BUILD_FOLDER)/$(BINARY)
	tar --exclude $(VERSION_FOLDER)/deb \
	    -czf $(RELEASE_FOLDER)/$(VERSION_FOLDER)_linux_$(1).tar.gz \
	    -C $(BASE_BUILD_FOLDER) $(VERSION_FOLDER)
	cp $(BUILD_FOLDER)/$(BINARY) $(BUILD_FOLDER)/deb/usr/bin/
	sed "s/PKG_NAME/$(PROJECT)/g; s/PKG_VERSION/$(VERSION)/g; s/ARCH/$(2)/g; s|DESCRIPTION|$(DESC)|g; s|MAINTAINER|$(MAINTAINER)|g" \
	    $(TMPL_DIR)/deb-control-template > $(BUILD_FOLDER)/deb/DEBIAN/control
	dpkg-deb --build $(BUILD_FOLDER)/deb $(RELEASE_FOLDER)/$(PROJECT)-$(VERSION)_linux_$(2).deb
	rm -f $(BUILD_FOLDER)/$(BINARY) $(BUILD_FOLDER)/deb/DEBIAN/control
endef

.PHONY: prepare-release
prepare-release:
	mkdir -p $(BUILD_FOLDER)
	mkdir -p $(RELEASE_FOLDER)
	$(if $(wildcard README.md), cp README.md $(BUILD_FOLDER)/)
	$(if $(wildcard LICENSE),   cp LICENSE   $(BUILD_FOLDER)/)

.PHONY: release-darwin
release-darwin:
	$(call build-tar,darwin,amd64)
	$(call build-tar,darwin,arm64)

.PHONY: release-linux
release-linux:
	@if ! command -v dpkg-deb >/dev/null 2>&1; then \
		echo "dpkg-deb not found (install with: brew install dpkg)"; exit 1; \
	fi
	mkdir -p $(BUILD_FOLDER)/deb/usr/bin/ $(BUILD_FOLDER)/deb/DEBIAN/
	$(call build-deb,amd64,amd64)
	$(call build-deb,arm64,arm64)
	rm -rf $(BUILD_FOLDER)/deb

.PHONY: release-formula
release-formula:
	@SUM_AMD64=$$(sha256sum $(RELEASE_FOLDER)/$(VERSION_FOLDER)_darwin_amd64.tar.gz | awk '{print $$1}'); \
	 SUM_ARM64=$$(sha256sum $(RELEASE_FOLDER)/$(VERSION_FOLDER)_darwin_arm64.tar.gz | awk '{print $$1}'); \
	 sed "s|REPO_URL|$(REPO_URL)|g; s/FORMULA_VERSION/$(VERSION)/g; s|DESCRIPTION|$(DESC)|g; s/SHA256_DARWIN_AMD64/$$SUM_AMD64/; s/SHA256_DARWIN_ARM64/$$SUM_ARM64/" \
	     $(TMPL_DIR)/formula-template.rb > $(RELEASE_FOLDER)/tortuga.rb

.PHONY: release-aur
release-aur:
	@SUM_AMD64=$$(sha256sum $(RELEASE_FOLDER)/$(VERSION_FOLDER)_linux_amd64.tar.gz | awk '{print $$1}'); \
	 SUM_ARM64=$$(sha256sum $(RELEASE_FOLDER)/$(VERSION_FOLDER)_linux_arm64.tar.gz | awk '{print $$1}'); \
	 sed "s|REPO_URL|$(REPO_URL)|g; s/PKG_NAME/$(PROJECT)-bin/g; s/PKG_VERSION/$(VERSION)/g; s|DESCRIPTION|$(DESC)|g; s|MAINTAINER|$(MAINTAINER)|g; s/SHA256_AMD64/$$SUM_AMD64/; s/SHA256_ARM64/$$SUM_ARM64/" \
	     $(TMPL_DIR)/pkgbuild-template > $(RELEASE_FOLDER)/PKGBUILD

.PHONY: release
release: clean fmt vet test prepare-release release-darwin release-formula release-linux release-aur
	@echo "Release done! Version: $(VERSION)"
