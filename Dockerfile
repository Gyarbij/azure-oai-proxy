FROM golang:1.23.4 AS builder
WORKDIR /build
COPY . .
RUN go get github.com/joho/godotenv
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o azure-oai-proxy .

# Use debian:12-slim instead of distroless for installing additional tools
FROM debian:12-slim
COPY --from=builder /build/azure-oai-proxy /

# Install required packages
RUN apt-get update && \
    apt-get install -y ca-certificates openssh-client curl && \
    curl -sSL https://sdk.cloud.google.com | bash && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

ENV PATH="$PATH:/root/google-cloud-sdk/bin"
RUN gcloud init

EXPOSE 11437
ENTRYPOINT ["/azure-oai-proxy"]