# syntax=docker/dockerfile:1
# Build stage
FROM golang:1.27-alpine AS builder
WORKDIR /app
# Dependencias em camada propria: so baixam de novo quando go.mod/go.sum mudam.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
# Cache de compilacao entre deploys: o servidor recompila so o que mudou.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o fincontrol .

# Runtime stage
FROM alpine:3.24
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/fincontrol .
COPY templates/ ./templates/
COPY static/    ./static/
ENV TZ=America/Sao_Paulo
EXPOSE 8080
CMD ["./fincontrol"]
