FROM golang:1.16.0

LABEL maintainer="Bollohz <github.com/bollohz>"

ENV GO111MODULE=on
ENV CGO_ENABLED=0

ARG IMAGE_NAME=bollohz/exporters_webhook
ARG IMAGE_TAG=latest
ARG BINARY_PATH=/usr/local/bin/exporter_webhook_server


WORKDIR /src
COPY . .
RUN GOOS=linux GOARCH=amd64 go build -o $BINARY_PATH ./src

ENV USER_UID=1001
ENTRYPOINT ["$BINARY_PATH"]

# switch to non-root user
USER root
