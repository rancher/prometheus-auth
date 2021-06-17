FROM alpine:3.12.7

MAINTAINER Frank Mai <frank@rancher.com>

RUN apk add --no-cache --update \
        curl openssl jq ca-certificates \
    ; \
    mkdir -p /data; \
    chown -R nobody:nogroup /data; \
    mkdir -p /run/cache

COPY prometheus-auth /usr/bin/

ENTRYPOINT ["/usr/bin/prometheus-auth"]