CGO_ENABLED=1
VERSION=development
COMMIT=$(shell git rev-parse --short HEAD)


.dockerimage: Dockerfile
	docker build -t csv2beancount-builder .
	touch .dockerimage

.PHONY: docker-go-%
docker-go-%: .dockerimage
	docker run \
	  --rm \
	  -it \
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
build-docker: docker-go-build

.PHONY: test
test:
	go \
	  test \
	  -cover \
	  -v \
	  github.com/cewood/csv2beancount/...

.PHONY: test-docker
test-docker: docker-go-test

.PHONY: reportcard
reportcard:
	goreportcard-cli -v

.PHONY: reportcard-docker
reportcard-docker: docker-go-reportcard

.PHONY: lint
lint:
	golangci-lint run --verbose

.PHONY: lint-docker
lint-docker: docker-go-lint
