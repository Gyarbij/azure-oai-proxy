# Build step
FROM golang:1.22.4 AS builder
RUN mkdir -p /build
WORKDIR /build
COPY . .
RUN go build

# Final step
FROM debian:bookworm-slim
RUN set -x && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    ca-certificates && \
    rm -rf /var/lib/apt/lists/* \

EXPOSE 11437
WORKDIR /app
COPY --from=builder /build/azure-oai-proxy /app/azure-oai-proxy
ENTRYPOINT ["/app/azure-oai-proxy"]