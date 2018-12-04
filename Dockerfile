#############
# phase one #
#############
FROM golang:1.11.1-alpine3.8 AS builder

RUN apk add --no-cache --update \
        build-base curl git

ARG VERSION=dev
ARG VCS_REF
ENV PROMETHEUS_AUTH_VER $VERSION
ENV PROMETHEUS_AUTH_HASH $VCS_REF

RUN echo "git cloning ..." \
    ; \
    git clone https://github.com/thxcode/prometheus-auth.git /go/src/github.com/rancher/prometheus-auth \
    ; \
    cd /go/src/github.com/rancher/prometheus-auth \
    ; \
    echo "go building version $PROMETHEUS_AUTH_VER ..." \
    ; \
    go build -i -tags k8s -ldflags "-X main.VER=$PROMETHEUS_AUTH_VER -X main.HASH=$PROMETHEUS_AUTH_HASH -s -w -extldflags -static" -o /build/bin/prometheus-auth ./cmd/main.go \
    ; \
    echo "completed"

#############
# phase two #
#############
FROM alpine:3.8

MAINTAINER Frank Mai <frank@rancher.com>

RUN apk add --no-cache --update \
        curl openssl jq ca-certificates \
    ; \
    mkdir -p /data; \
    chown -R nobody:nogroup /data; \
    mkdir -p /run/cache

COPY --from=builder /build/bin/prometheus-auth /usr/bin/prometheus-auth

USER    nobody
EXPOSE  9201

ENTRYPOINT ["/usr/bin/prometheus-auth"]