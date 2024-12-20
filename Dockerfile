FROM golang:1.23.4 AS builder
WORKDIR /build
COPY . .
RUN go get github.com/joho/godotenv
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o azure-oai-proxy .

FROM gcr.io/distroless/base-debian12
COPY --from=builder /build/azure-oai-proxy /
RUN apt-get update && apt-get install -y ca-certificates && apt-get install -y openssh-client
RUN apt-get update && apt-get install -y curl
RUN curl -sSL https://sdk.cloud.google.com | bash
ENV PATH="$PATH:/opt/google-cloud-sdk/bin"
RUN gcloud init
EXPOSE 11437
ENTRYPOINT ["/azure-oai-proxy"]