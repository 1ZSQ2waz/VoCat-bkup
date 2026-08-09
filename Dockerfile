# syntax=docker/dockerfile:1.7

# ---- Stage 1: build the web frontend ----
FROM node:20-alpine AS web-builder
WORKDIR /web
COPY web/package.json web/package-lock.json* ./
RUN npm ci
COPY web/ ./
RUN npm run build

# ---- Stage 2: build the Go binary ----
FROM golang:1.25-alpine AS go-builder
RUN apk add --no-cache git
WORKDIR /src

ARG VERSION=0.1.0-dev
ARG BUILD_TIME=""

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Overlay the freshly built frontend so go:embed web/dist picks it up.
COPY --from=web-builder /web/dist ./web/dist

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags "-s -w -X vocat/internal/buildinfo.Version=${VERSION} -X vocat/internal/buildinfo.BuildTime=${BUILD_TIME}" \
    -o /out/vocat \
    ./cmd/vocat

# ---- Stage 3: minimal runtime ----
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S -g 1000 vocat && \
    adduser -S -D -H -u 1000 -G vocat vocat

RUN mkdir -p /opt/vocat/bin /opt/vocat/data && \
    chown -R vocat:vocat /opt/vocat

COPY --from=go-builder /out/vocat /opt/vocat/bin/vocat

USER vocat
VOLUME ["/opt/vocat/data"]
EXPOSE 7575
ENV VOCAT_ADDR=0.0.0.0:7575 \
    VOCAT_DATABASE_PATH=/opt/vocat/data/vocat.db

ENTRYPOINT ["/opt/vocat/bin/vocat"]
