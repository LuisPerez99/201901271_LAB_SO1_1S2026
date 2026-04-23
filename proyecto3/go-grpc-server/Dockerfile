# Go gRPC Server Multi-stage Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy the shared protobuf module first so the local replace works during build.
COPY go-shared-pb ./go-shared-pb
COPY go-grpc-server/go.mod go-grpc-server/go.sum* ./go-grpc-server/

WORKDIR /app/go-grpc-server
RUN go mod download

COPY go-grpc-server/. .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o grpc-server .

FROM alpine:3.18

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/go-grpc-server/grpc-server .

EXPOSE 50051

ENV GRPC_PORT=:50051
ENV RABBITMQ_ADDR=rabbitmq:5672
ENV RABBITMQ_VHOST=/warreports
ENV RABBITMQ_USER=guest
ENV RABBITMQ_PASS=guest

CMD ["./grpc-server"]
