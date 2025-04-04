FROM golang:latest AS builder
WORKDIR /app
RUN apt-get update && apt-get install -y gcc sqlite3 libsqlite3-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o out .

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    sqlite3 && \
    rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /app/out .
CMD ["./out"]