FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY . .

RUN go build -o main ./src

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/main .
COPY .env .env

EXPOSE 8083

CMD ["./main"]
