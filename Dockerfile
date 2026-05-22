# Build stage
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o fincontrol .

# Runtime stage
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/fincontrol .
COPY templates/ ./templates/
COPY static/    ./static/
ENV TZ=America/Sao_Paulo
EXPOSE 8080
CMD ["./fincontrol"]
