# Use a Debian-based Golang image as the builder
FROM golang:1.23-buster AS builder
WORKDIR /app

# Install gcc for CGO
RUN apt-get update && apt-get install -y gcc

ENV CGO_ENABLED=1
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o out .

# Minimal runtime image
FROM alpine:latest
RUN apk --no-cache add ca-certificates sqlite
WORKDIR /app
COPY --from=builder /app/out .
CMD ["./out"]
