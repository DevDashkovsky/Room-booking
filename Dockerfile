FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /room-booking ./cmd/api

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /room-booking /app/room-booking
COPY --from=builder /app/migrations /app/migrations

EXPOSE 8080
CMD ["/app/room-booking"]
