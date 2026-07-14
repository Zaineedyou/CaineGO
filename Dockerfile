# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o caine .

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# ca-certificates untuk HTTPS ke Discord/Groq API
RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/caine .

# Jalankan sebagai non-root user
RUN addgroup -S botgroup && adduser -S botuser -G botgroup
USER botuser

CMD ["./caine"]
