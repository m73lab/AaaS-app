# Multi-stage build: compile the proxy, then ship a minimal distroless image.
FROM golang:1.27 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/aaas-proxy ./cmd/proxy

FROM gcr.io/distroless/static-debian12 AS runtime
WORKDIR /app
COPY --from=build /out/aaas-proxy /app/aaas-proxy
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/aaas-proxy"]
CMD ["-config", "/app/config.yaml"]
