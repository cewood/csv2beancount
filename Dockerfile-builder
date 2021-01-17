FROM golang:1.14.2 AS base
FROM goreleaser/goreleaser:v0.133 AS goreleaser
FROM golangci/golangci-lint:v1.26.0 AS golangci-lint


FROM base AS build

RUN apt-get update && apt-get install -y \
  make \
  gcc \
  && rm -rf /var/lib/apt/lists/*

COPY --from=goreleaser /bin/goreleaser /bin/
COPY --from=golangci-lint /usr/bin/golangci-lint /bin/

RUN GO111MODULE=on go get github.com/gojp/goreportcard/cmd/goreportcard-cli@59167b5 \
  && mv /go/bin/goreportcard-cli /bin/

RUN wget -qO- https://github.com/alecthomas/gometalinter/releases/download/v3.0.0/gometalinter-3.0.0-linux-amd64.tar.gz | tar -xzf - --strip-components=1 -C /bin
