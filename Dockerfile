FROM golang:1.21 AS build-env

WORKDIR /app
COPY . .

RUN go mod tidy && \
    CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o bin/prometheus-auth cmd/main.go


FROM alpine:latest
RUN echo 'promauth:x:1000:1000::/home/promauth:/bin/bash' >> /etc/passwd && \
    echo 'promauth:x:1000:' >> /etc/group && \
    mkdir /home/promauth && \
    chown -R promauth:promauth /home/promauth

COPY --from=build-env /app/bin/prometheus-auth /usr/bin/
USER promauth
CMD ["prometheus-auth"]
