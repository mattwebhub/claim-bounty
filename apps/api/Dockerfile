# syntax=docker/dockerfile:1.7
FROM --platform=$BUILDPLATFORM golang:1.26.6-alpine3.23@sha256:e57c41c1d5864341031181b0db34b9a537bb5773eb6428e4e5bdaea0f9135406 AS build
ARG TARGETOS=linux
ARG TARGETARCH=amd64
WORKDIR /src/apps/api
RUN apk add --no-cache ca-certificates
COPY apps/api/go.mod apps/api/go.sum* ./
RUN go mod download
COPY apps/api/ ./
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w -buildid=" -o /out/api ./cmd/api

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build --chown=65532:65532 /out/api /api
USER 65532:65532
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=6 CMD ["/api", "healthcheck"]
ENTRYPOINT ["/api"]
