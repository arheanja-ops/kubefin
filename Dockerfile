# syntax=docker/dockerfile:1

# --- build stage ---
FROM golang:1.27 AS build
WORKDIR /src

# Cachea las dependencias antes de copiar el resto del código.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -ldflags "-s -w -X main.version=${VERSION}" -o /out/kubefin ./cmd/kubefin

# --- runtime stage ---
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/kubefin /usr/local/bin/kubefin
USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/kubefin"]
# El servidor MCP (`mcp`) llega en la Fase 3; hasta entonces el default es --help.
CMD ["--help"]
