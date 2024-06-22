FROM golang:1.22.4 AS builder
WORKDIR /build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o azure-oai-proxy .

FROM gcr.io/distroless/base-debian12
COPY --from=builder /build/azure-oai-proxy /
EXPOSE 11437
ENTRYPOINT ["/azure-oai-proxy"]