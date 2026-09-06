# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS builder
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod ./
COPY go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/app ./cmd/app

FROM alpine:3.20
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata \
 && adduser -D -H -u 10001 app
COPY --from=builder /out/app /app/app
COPY .env.example /app/.env.example
COPY views /app/views
COPY public /app/public
COPY database /app/database
RUN mkdir -p /app/storage/framework/sessions \
             /app/storage/framework/cache \
             /app/storage/framework/schedule \
             /app/storage/logs \
             /app/storage/app/public \
 && chown -R app:app /app
USER app
ENV APP_ENV=production \
    APP_URL=http://localhost:8080
EXPOSE 8080
CMD ["/app/app", "serve", "--port", "8080"]
