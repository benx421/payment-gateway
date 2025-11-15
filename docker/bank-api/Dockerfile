FROM golang:1.25.4-alpine3.21

ENV AIR_VERSION=v1.63.0 \
    GOLANGCI_LINT_VERSION=2.6.2

RUN apk add --no-cache \
    git \
    make \
    curl

RUN go install github.com/air-verse/air@${AIR_VERSION}

RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | \
    sh -s -- -b /go/bin v${GOLANGCI_LINT_VERSION}

WORKDIR /app

COPY bank/go.mod bank/go.sum ./
RUN go mod download

COPY bank/ .

EXPOSE 8080

CMD ["air", "-c", ".air.toml"]
