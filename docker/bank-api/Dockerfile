FROM golang:1.25.4-alpine3.21

ENV AIR_VERSION=v1.61.7 \
    GOLANGCI_LINT_VERSION=v1.62.2

RUN apk add --no-cache \
    git \
    make \
    curl

RUN go install github.com/air-verse/air@${AIR_VERSION} && \
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@${GOLANGCI_LINT_VERSION}

WORKDIR /app

COPY bank/go.mod bank/go.sum ./
RUN go mod download

COPY bank/ .

EXPOSE 8080

CMD ["air", "-c", ".air.toml"]
