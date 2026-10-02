FROM golang:1.27 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o server ./cmd/server
RUN CGO_ENABLED=0 go build -o migrate ./cmd/migrate


FROM ubuntu:latest

WORKDIR /app

COPY --from=builder /app/server .
COPY --from=builder /app/migrate .
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/web ./web

EXPOSE 8080

CMD ["./server"]