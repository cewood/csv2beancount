CGO_ENABLED        = 0
DOCKER_INTERACTIVE = $(if $(shell printenv GITHUB_ACTIONS),-t,-it)
GIT_REVISION       = $(or $(shell printenv GIT_REVISION), $(shell git describe --match= --always --abbrev=7 --dirty))
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

.dockerimage: Dockerfile
	docker build -t csv2beancount-builder .
	touch .dockerimage

.PHONY: docker-do-%
docker-do-%: .dockerimage
	docker run \
	  --rm \
	  ${DOCKER_INTERACTIVE} \
	  -e CGO_ENABLED=0 \
	  -v ${PWD}:/code \
	  -w /code \
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
	  -v \
	  github.com/cewood/csv2beancount/...

.PHONY: test-docker
test-docker: docker-do-test

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
