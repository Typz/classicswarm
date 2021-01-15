ARG GO_VERSION=1.12.4

FROM golang:${GO_VERSION}-alpine AS builder

ARG GOOS
ARG GOARCH

COPY . /go/src/github.com/docker/swarm
WORKDIR /go/src/github.com/docker/swarm

RUN set -ex \
	&& apk add --no-cache --virtual .build-deps \
	git \
	&& GOARCH=$GOARCH GOOS=$GOOS CGO_ENABLED=0 go install -v -a -tags netgo -installsuffix netgo -ldflags "-w -X github.com/docker/swarm/version.GITCOMMIT=$(git rev-parse --short HEAD) -X github.com/docker/swarm/version.BUILDTIME=$(date -u +%FT%T%z)"  \
	&& apk del .build-deps

################################################################################
FROM debian:latest AS certificates

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates

################################################################################
FROM scratch

WORKDIR /tmp
WORKDIR /
COPY --from=certificates /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /go/bin/swarm /swarm

ENV SWARM_HOST :2375
EXPOSE 2375

VOLUME /.swarm

ENTRYPOINT ["/swarm"]
CMD ["--help"]
