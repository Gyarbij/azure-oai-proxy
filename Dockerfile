FROM golang:1.22.5 AS builder
WORKDIR /build
COPY . .
RUN go get github.com/joho/godotenv
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o azure-oai-proxy .

FROM gcr.io/distroless/base-debian12
COPY --from=builder /build/azure-oai-proxy /
EXPOSE 11437
ENTRYPOINT ["/azure-oai-proxy"]