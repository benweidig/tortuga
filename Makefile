PROJECT := tortuga
BINARY  := tt
REPO    := github.com/benweidig/tortuga
HASH    := $(shell git rev-parse --short HEAD)
DATE    := $(shell date)
TAG     := $(shell git describe --tags --always --abbrev=0 --match="v[0-9]*.[0-9]*.[0-9]*" 2> /dev/null)
VERSION := $(shell echo "${TAG}" | sed 's/^.//')

BASE_BUILD_FOLDER := build
VERSION_FOLDER    := ${PROJECT}-${VERSION}
BUILD_FOLDER      := ${BASE_BUILD_FOLDER}/${VERSION_FOLDER}
RELEASE_FOLDER    := release/${PROJECT}-${VERSION}

LDFLAGS_DEV     := -ldflags "-X '${REPO}/version.CommitHash=${HASH}' -X '${REPO}/version.CompileDate=${DATE}'"
LDFLAGS_RELEASE := -ldflags "-X '${REPO}/version.Version=${VERSION}' -X '${REPO}/version.CommitHash=${HASH}' -X '${REPO}/version.CompileDate=${DATE}'"


.PHONY: all
all: clean fmt lint build test


.PHONY: clean
clean:
	#
	# ################################################################################
	# >>> TARGET: clean
	# ################################################################################
	#
	go clean
	rm -rf build
	rm -rf release


.PHONY: fmt
fmt:
	#
	# ################################################################################
	# >>> TARGET: fmt
	# ################################################################################
	#
	go fmt


.PHONY: lint
lint:
	#
	# ################################################################################
	# >>> TARGET: lint
	# ################################################################################
	#
	go get -u golang.org/x/lint/golint
	golint ./...


.PHONY: test
test:
	#
	# ################################################################################
	# >>> TARGET: test
	# ################################################################################
	#
	go test


.PHONY: build
build:
	#
	# ################################################################################
	# >>> TARGET: build
	# ################################################################################
	#
	go build ${LDFLAGS_DEV} -o build/${BINARY}


.PHONY: version
version:
	#
	# ################################################################################
	# >>> TARGET: version
	# ################################################################################
	#
	@echo "Version: ${VERSION}"


.PHONY: tag
tag:
	#
	# ################################################################################
	# >>> TARGET: tag
	# ################################################################################
	#
	@echo "Tag: ${TAG}"


.PHONY: prepare-release
prepare-release:
	#
	# ################################################################################
	# >>> TARGET: prepare-release
	# ################################################################################
	#
	mkdir -p ${BUILD_FOLDER}
	mkdir -p ${RELEASE_FOLDER}
	cp README.md ${BUILD_FOLDER}/
	cp LICENSE ${BUILD_FOLDER}/


.PHONY: release
release: clean fmt lint test prepare-release release-darwin release-linux release-windows

	#
	# ################################################################################
	# >>> RELEASE DONE
	# ################################################################################
	#
	@echo "Relase Done! Version: ${VERSION}"


.PHONY: release-linux
release-linux:
	#
	# ################################################################################
	# >>> TARGET: release-linux
	# ################################################################################
	#

	#
	# >> PREPARE .deb-file
	#
	mkdir -p ${BUILD_FOLDER}/deb/usr/bin/ ${BUILD_FOLDER}/deb/DEBIAN/
	cp -rf deb-control-template ${BUILD_FOLDER}/deb/DEBIAN/control
	sed -i 's/PKG_NAME/${PROJECT}/g' ${BUILD_FOLDER}/deb/DEBIAN/control
	sed -i 's/PKG_VERSION/${VERSION}/g' ${BUILD_FOLDER}/deb/DEBIAN/control

	#
	# >> LINUX/386
	#
	# > build binary
	#
	GOOS=linux GOARCH=386 go build ${LDFLAGS_RELEASE} -o ${BUILD_FOLDER}/${BINARY}

	#
	# > tar.gz binary
	#
	tar --exclude ${VERSION_FOLDER}/deb -czf ${RELEASE_FOLDER}/${VERSION_FOLDER}_linux_386.tar.gz -C ${BASE_BUILD_FOLDER} ${VERSION_FOLDER}

	#
	# > prepare .deb-file
	#
	cp ${BUILD_FOLDER}/${BINARY} ${BUILD_FOLDER}/deb/usr/bin/
	cp -rf deb-control-template ${BUILD_FOLDER}/deb/DEBIAN/control
	sed -i 's/PKG_NAME/${PROJECT}/g' ${BUILD_FOLDER}/deb/DEBIAN/control
	sed -i 's/PKG_VERSION/${VERSION}/g' ${BUILD_FOLDER}/deb/DEBIAN/control
	sed -i 's/ARCH/i386/g' ${BUILD_FOLDER}/deb/DEBIAN/control

	#
	# > build .deb-file
	#
	dpkg-deb --build ${BUILD_FOLDER}/deb ${RELEASE_FOLDER}/${PROJECT}-${VERSION}_linux_386.deb

	#
	# > cleanup
	#
	rm -f ${BUILD_FOLDER}/${BINARY}
	rm -f ${BUILD_FOLDER}/deb/DEBIAN/control

	#
	# >> LINUX/AMD64
	#
	# > build binary
	#
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS_RELEASE} -o ${BUILD_FOLDER}/${BINARY}

	#
	# > tar.gz binary
	#
	tar --exclude ${VERSION_FOLDER}/deb -czf ${RELEASE_FOLDER}/${VERSION_FOLDER}_linux_amd64.tar.gz -C ${BASE_BUILD_FOLDER} ${VERSION_FOLDER}

	#
	# > prepare .deb-file
	#
	cp ${BUILD_FOLDER}/${BINARY} ${BUILD_FOLDER}/deb/usr/bin/
	cp -rf deb-control-template ${BUILD_FOLDER}/deb/DEBIAN/control
	sed -i 's/PKG_NAME/${PROJECT}/g' ${BUILD_FOLDER}/deb/DEBIAN/control
	sed -i 's/PKG_VERSION/${VERSION}/g' ${BUILD_FOLDER}/deb/DEBIAN/control
	sed -i 's/ARCH/amd64/g' ${BUILD_FOLDER}/deb/DEBIAN/control

	#
	# > build .deb-file
	#
	dpkg-deb --build ${BUILD_FOLDER}/deb ${RELEASE_FOLDER}/${PROJECT}-${VERSION}_linux_amd64.deb

	#
	# > cleanup
	#
	rm -f ${BUILD_FOLDER}/${BINARY}
	rm -f ${BUILD_FOLDER}/deb/DEBIAN/control

	#
	# >> LINUX/ARM
	#
	# > build binary
	#
	GOOS=linux GOARCH=arm go build ${LDFLAGS_RELEASE} -o ${BUILD_FOLDER}/${BINARY}

	#
	# >> LINUX/ARM64
	#
	# > build binary
	#
	GOOS=linux GOARCH=arm64 go build ${LDFLAGS_RELEASE} -o ${BUILD_FOLDER}/${BINARY}

	#
	# > tar.gz binary
	#
	tar --exclude ${VERSION_FOLDER}/deb -czf ${RELEASE_FOLDER}/${VERSION_FOLDER}_linux_arm.tar.gz -C ${BASE_BUILD_FOLDER} ${VERSION_FOLDER}

	#
	# > prepare .deb-file
	#
	cp ${BUILD_FOLDER}/${BINARY} ${BUILD_FOLDER}/deb/usr/bin/
	cp -rf deb-control-template ${BUILD_FOLDER}/deb/DEBIAN/control
	sed -i 's/PKG_NAME/${PROJECT}/g' ${BUILD_FOLDER}/deb/DEBIAN/control
	sed -i 's/PKG_VERSION/${VERSION}/g' ${BUILD_FOLDER}/deb/DEBIAN/control
	sed -i 's/ARCH/armhf/g' ${BUILD_FOLDER}/deb/DEBIAN/control

	#
	# > build .deb-file
	#
	dpkg-deb --build ${BUILD_FOLDER}/deb ${RELEASE_FOLDER}/${PROJECT}-${VERSION}_linux_armhf.deb

	#
	# > cleanup
	#
	rm ${BUILD_FOLDER}/${BINARY}
	rm -f ${BUILD_FOLDER}/deb/DEBIAN/control


	rm -r ${BUILD_FOLDER}/deb


.PHONY: release-windows
release-windows:
	# Window has some dependencies that won't be available by the usual way, so we
	# need to get them here
	go get -u github.com/inconshreveable/mousetrap
	go get -u github.com/mattn/go-isatty
	#
	# ################################################################################
	# >>> TARGET: release-windows
	# ################################################################################
	#
	# >> WINDOWS/386
	#
	GOOS=windows GOARCH=386 go build ${LDFLAGS_RELEASE} -o ${BUILD_FOLDER}/${BINARY}
	tar --exclude ${VERSION_FOLDER}/deb -czf ${RELEASE_FOLDER}/${VERSION_FOLDER}_windows_386.tar.gz -C ${BASE_BUILD_FOLDER} ${VERSION_FOLDER}
	rm ${BUILD_FOLDER}/${BINARY}

	#
	# >> WINDOWS/AMD64
	#
	GOOS=windows GOARCH=amd64 go build ${LDFLAGS_RELEASE} -o ${BUILD_FOLDER}/${BINARY}
	tar --exclude ${VERSION_FOLDER}/deb -czf ${RELEASE_FOLDER}/${VERSION_FOLDER}_windows_amd64.tar.gz -C ${BASE_BUILD_FOLDER} ${VERSION_FOLDER}
	rm ${BUILD_FOLDER}/${BINARY}


.PHONY: release-darwin
release-darwin:
	#
	# ################################################################################
	# >>> TARGET: release-darwin
	# ################################################################################
	#
	# >> DARWIN/AMD64
	#
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS_RELEASE} -o ${BUILD_FOLDER}/${BINARY}
	tar --exclude ${VERSION_FOLDER}/deb -czf ${RELEASE_FOLDER}/${VERSION_FOLDER}_darwin_amd64.tar.gz -C ${BASE_BUILD_FOLDER} ${VERSION_FOLDER}
	rm ${BUILD_FOLDER}/${BINARY}

	#
	# >> DARWIN/ARM64
	#
	GOOS=darwin GOARCH=arm64 go build ${LDFLAGS_RELEASE} -o ${BUILD_FOLDER}/${BINARY}
	tar --exclude ${VERSION_FOLDER}/deb -czf ${RELEASE_FOLDER}/${VERSION_FOLDER}_darwin_arm64.tar.gz -C ${BASE_BUILD_FOLDER} ${VERSION_FOLDER}
	rm ${BUILD_FOLDER}/${BINARY}
