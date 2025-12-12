FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /api ./cmd/api

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    chromium \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /root/

COPY --from=builder /api .

ENV PORT=8087
ENV CHROMIUM_PATH=/usr/bin/chromium

EXPOSE 8087

CMD ["./api"]
