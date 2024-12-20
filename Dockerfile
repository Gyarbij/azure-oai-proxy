FROM golang:1.23.4 AS builder
WORKDIR /build
COPY . .
RUN go get github.com/joho/godotenv
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o azure-oai-proxy .

FROM debian:12.5-slim

RUN apt-get update && apt-get install -y ca-certificates openssh-client curl python3

# Install the Google Cloud SDK
RUN curl -sSL https://sdk.cloud.google.com | bash

# Add the Google Cloud SDK bin directory to the PATH
ENV PATH="$PATH:/root/google-cloud-sdk/bin"

# Copy the binary from the builder stage
COPY --from=builder /build/azure-oai-proxy /

# Set the working directory for the service account key
WORKDIR /app

EXPOSE 11437
ENTRYPOINT ["/azure-oai-proxy"]
