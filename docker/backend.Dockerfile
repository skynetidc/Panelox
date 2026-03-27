FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o /panelox \
    ./cmd/panelox

# ---

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

RUN addgroup -S panelox && adduser -S panelox -G panelox

COPY --from=builder /panelox /usr/local/bin/panelox
COPY backend/migrations /opt/panelox/migrations

RUN chown -R panelox:panelox /opt/panelox

USER panelox
WORKDIR /opt/panelox

EXPOSE 8080

ENTRYPOINT ["panelox"]
