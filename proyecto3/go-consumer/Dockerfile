# Go Consumer Multi-stage Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o consumer .

FROM alpine:3.18

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/consumer .

ENV RABBITMQ_ADDR=rabbitmq:5672
ENV RABBITMQ_USER=guest
ENV RABBITMQ_PASS=guest
ENV RABBITMQ_VHOST=/warreports
ENV VALKEY_ADDR=valkey-svc:6379
ENV VALKEY_PASS=
ENV CONSUMER_COUNTRY=usa

CMD ["./consumer"]
