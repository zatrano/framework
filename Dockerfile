# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS builder
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod ./
COPY go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/zatrano ./cmd/zatrano

FROM alpine:3.20
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata \
 && adduser -D -H -u 10001 zatrano
COPY --from=builder /out/zatrano /app/zatrano
COPY .env.example /app/.env.example
USER zatrano
ENTRYPOINT ["/app/zatrano"]
CMD ["list"]
