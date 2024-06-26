export VERSION:=$(shell cat ../VERSION)

build/stamp: $(shell find charts -type f)
	mkdir -p build
	rm -rf build/*.tgz
	helm package --destination build charts/determined
	cd build/ && ln -s determined-*.tgz determined-latest.tgz && cd ..
	touch $@

.PHONY: build
build: build/stamp

fmt:
	helm lint charts/determined/

.PHONY: clean
clean:
	rm -rf build/

.PHONY: release-gh
release-gh: export GORELEASER_CURRENT_TAG := $(VERSION)
release-gh: export GORELEASER_PREVIOUS_TAG := $(shell git tag --sort=-creatordate | grep -E '^[0-9.]+$$' | grep "$(VERSION)" -A1 | sed -n '2 p')
release-gh:
	go install github.com/goreleaser/goreleaser@v1.14.1
	git clean -df
	goreleaser --rm-dist

.PHONY: release-gh-ee
release-gh-ee: export GORELEASER_CURRENT_TAG := $(VERSION)-ee
release-gh-ee: export GORELEASER_PREVIOUS_TAG := $(shell git tag --sort=-creatordate | grep -E '^[0-9.]+-ee$$' | grep "$(VERSION)-ee" -A1 | sed -n '2 p')
release-gh-ee:
	go install github.com/goreleaser/goreleaser@v1.14.1
	git clean -df
	goreleaser --rm-dist
