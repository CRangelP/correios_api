FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /api ./cmd/api

FROM alpine:latest

RUN apk --no-cache add ca-certificates chromium

WORKDIR /root/

COPY --from=builder /api .

ENV PORT=8087
ENV CHROMIUM_PATH=/usr/bin/chromium-browser

EXPOSE 8087

CMD ["./api"]
