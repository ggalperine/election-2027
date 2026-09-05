# Build all Go service binaries in one image; docker-compose picks binary per service.
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/gateway          ./cmd/gateway          && \
    CGO_ENABLED=0 go build -o /out/aggregator       ./cmd/aggregator       && \
    CGO_ENABLED=0 go build -o /out/poll-ingester    ./cmd/poll-ingester    && \
    CGO_ENABLED=0 go build -o /out/results-ingester ./cmd/results-ingester && \
    CGO_ENABLED=0 go build -o /out/scheduler        ./cmd/scheduler

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/ /app/
