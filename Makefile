ARCH               = $(or $(shell printenv ARCH),$(shell echo linux/amd64,linux/arm64))
BUILD_FLAGS        = $(or $(shell printenv BUILD_FLAGS),--pull)
CGO_ENABLED        = 1
CODECOV_TOKEN      = $(or $(shell printenv CODECOV_TOKEN),"UNSET")
CREATED            = $(or $(shell printenv CREATED),$(shell date --rfc-3339=seconds))
DOCKER_INTERACTIVE = $(if $(shell printenv GITHUB_ACTIONS),-t,-it)
GIT_REVISION       = $(or $(shell printenv GIT_REVISION), $(shell git describe --match= --always --abbrev=7 --dirty))
IMAGE              = $(or $(shell printenv IMAGE),cewood/csv2beancount)
IMAGE_TAG          = $(or $(shell printenv IMAGE_TAG),${TAG_REVISION})
TAG_REVISION       = $(or $(shell printenv TAG_REVISION),${GIT_REVISION})
VERSION            = snapshot


.PHONY: all-docker
all-docker: docker-do-all

.PHONY: all
all:
	make build \
	  && make test \
	  && make lint \
	  && make reportcard \
	  && make release-snapshot

.dockerimage-builder: Dockerfile-builder
	docker \
	  build \
	  --tag csv2beancount-builder \
	  --file Dockerfile-builder \
	  .
	touch .dockerimage-builder

.PHONY: docker-do-%
docker-do-%: .dockerimage-builder
	docker run \
	  --rm \
	  ${DOCKER_INTERACTIVE} \
	  -e CGO_ENABLED=${CGO_ENABLED} \
	  -e CODECOV_TOKEN=${CODECOV_TOKEN} \
	  -v ${PWD}:/workdir \
	  -w /workdir \
	  csv2beancount-builder \
	  make $*

.PHONY: build
build:
	CGO_ENABLED=${CGO_ENABLED} \
	go \
	  build

.PHONY: build-docker
build-docker: docker-do-build

.PHONY: test
test:
	go \
	  test \
	  -cover \
	  -covermode=atomic \
	  -coverprofile=coverage.txt \
	  -race \
	  -v \
	  github.com/cewood/csv2beancount/...

.PHONY: test-docker
test-docker: docker-do-test

.PHONY: test-ci-docker
test-ci-docker: docker-do-test docker-do-codecov-upload

.PHONY: reportcard
reportcard:
	goreportcard-cli -v

.PHONY: reportcard-docker
reportcard-docker: docker-do-reportcard

.PHONY: lint
lint:
	golangci-lint run --verbose

.PHONY: lint-docker
lint-docker: docker-do-lint

.PHONY: release-snapshot
release-snapshot:
	CGO_ENABLED=${CGO_ENABLED} \
	GIT_REVISION=${GIT_REVISION} \
	VERSION=${VERSION} \
	goreleaser \
	  --snapshot \
	  --skip-publish \
	  --rm-dist

.PHONY: release-snapshot-docker
release-snapshot-docker: docker-do-release-snapshot

.PHONY: release
release:
	CGO_ENABLED=${CGO_ENABLED} \
	GIT_REVISION=${GIT_REVISION} \
	GITHUB_TOKEN=${GITHUB_TOKEN} \
	VERSION=${VERSION} \
	goreleaser \
	  release \
	  --rm-dist

.PHONY: release-docker
release-docker: docker-do-release

.PHONY: codecov-upload
codecov-upload: SHELL := /bin/bash
codecov-upload:
	bash <(curl -s https://codecov.io/bash) || echo 'Codecov failed to upload'

.PHONY: load
load:
	$(MAKE) build-dockerimages BUILD_FLAGS=--load

.PHONY: inspect
inspect:
	docker inspect ${IMAGE}:${IMAGE_TAG}

.PHONY: binfmt-setup
binfmt-setup:
	docker \
	  run \
	  --rm \
	  --privileged \
	  docker/binfmt:66f9012c56a8316f9244ffd7622d7c21c1f6f28d

.PHONY: buildx-setup
buildx-setup:
	DOCKER_CLI_EXPERIMENTAL=enabled \
	docker \
	  buildx \
	  create \
	  --use \
	  --name multiarch

.PHONY: dive
dive:
	docker run --rm -it \
	  -e CI=true \
	  -v /var/run/docker.sock:/var/run/docker.sock \
	  wagoodman/dive:v0.9.2 ${IMAGE}:${IMAGE_TAG}

.PHONY: build-dockerimages
build-dockerimages:
	DOCKER_CLI_EXPERIMENTAL=enabled \
	docker \
	  buildx build \
	  ${BUILD_FLAGS} \
	  --build-arg CREATED="${CREATED}" \
	  --build-arg REVISION="${GIT_REVISION}" \
	  --platform ${ARCH} \
	  --tag ${IMAGE}:${IMAGE_TAG} \
	  -f Dockerfile \
	  .

.PHONY: build-and-push-dockerimages
build-and-push-dockerimages:
	$(MAKE) build-dockerimages BUILD_FLAGS=$(if $(findstring tags,${GITHUB_REF}),--push,--pull)
	$(MAKE) build-dockerimages BUILD_FLAGS=$(if $(findstring tags,${GITHUB_REF}),--push,--pull) TAG_REVISION=latest
