FROM golang:1.27 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o shortener ./cmd/main.go

FROM ubuntu:latest

WORKDIR /app

COPY --from=builder /app/shortener .

EXPOSE 8080

CMD ["./shortener"]