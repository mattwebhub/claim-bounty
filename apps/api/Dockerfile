# syntax=docker/dockerfile:1.7
FROM golang:1.27.0-alpine3.23@sha256:3747dcba41c8b0db3211fda4db61638b980e17ac5bb3c94460a975a9cfe19395 AS build
WORKDIR /src/apps/api
RUN apk add --no-cache ca-certificates
COPY apps/api/go.mod apps/api/go.sum* ./
RUN go mod download
COPY apps/api/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w -buildid=" -o /out/api ./cmd/api

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build --chown=65532:65532 /out/api /api
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/api"]
