# Standalone build (docker build -t orcadub-mcp .) — release images are
# published to ghcr.io by goreleaser using Dockerfile.goreleaser instead.
FROM golang:1.25 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /orcadub-mcp ./cmd

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /orcadub-mcp /usr/local/bin/orcadub-mcp
ENTRYPOINT ["/usr/local/bin/orcadub-mcp"]
