FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -ldflags="-s -w" -o server cmd/server/main.go

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/server .
COPY internal/db/schema.sql internal/db/schema.sql
EXPOSE 8080
CMD ["./server"]
