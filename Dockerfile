FROM golang:1.26.1-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /build/server ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /build/server .
COPY --from=builder /build/home-ticket-system.html /app/index.html

EXPOSE 8080
CMD ["./server"]
