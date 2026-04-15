# Go API Client Multi-stage Dockerfile
FROM golang:1.21-alpine as builder

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o api-client .

FROM alpine:3.18

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/api-client .

EXPOSE 8080

ENV HTTP_PORT=:8080
ENV GRPC_SERVER_ADDR=go-grpc-server:50051

CMD ["./api-client"]
