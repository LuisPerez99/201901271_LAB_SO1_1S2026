# Go gRPC Server Multi-stage Dockerfile
FROM golang:1.21-alpine as builder

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o grpc-server .

FROM alpine:3.18

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/grpc-server .

EXPOSE 50051

ENV GRPC_PORT=:50051
ENV RABBITMQ_ADDR=rabbitmq:5672
ENV RABBITMQ_VHOST=/warreports
ENV RABBITMQ_USER=guest
ENV RABBITMQ_PASS=guest

CMD ["./grpc-server"]
