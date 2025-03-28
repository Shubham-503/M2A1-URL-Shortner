FROM golang:latest AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o out .

FROM debian:bookworm-slim
RUN apk --no-cache add ca-certificates sqlite
WORKDIR /app
COPY --from=builder /app/out .
CMD ["./out"]