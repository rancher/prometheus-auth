FROM alpine:3.8

MAINTAINER Frank Mai <frank@rancher.com>

COPY prometheus-auth /usr/bin/

USER    nobody
EXPOSE  9201

ENTRYPOINT ["/usr/bin/prometheus-auth"]
