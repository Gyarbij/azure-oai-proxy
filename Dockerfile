FROM golang:1.23.4 AS builder
WORKDIR /build
COPY . .
RUN go get github.com/joho/godotenv
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o azure-oai-proxy .

FROM debian:12.5-slim

RUN apt-get update && apt-get install -y ca-certificates openssh-client curl

RUN curl -sSL https://sdk.cloud.google.com | bash
ENV PATH="$PATH:/root/google-cloud-sdk/bin"

COPY --from=builder /build/azure-oai-proxy /

# Set the working directory for the service account key
WORKDIR /app

EXPOSE 11437
ENTRYPOINT ["/azure-oai-proxy"]