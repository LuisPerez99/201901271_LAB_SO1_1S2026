# syntax=docker/dockerfile:1.4
# Go gRPC Server Multi-stage Dockerfile
FROM golang:1.26-alpine AS builder

RUN apk --no-cache add ca-certificates

WORKDIR /app/go-grpc-server

# Copy shared protobuf module from an extra build context named "sharedpb".
# This keeps replace ../go-shared-pb working without cloning from GitHub.
COPY --from=sharedpb . /app/go-shared-pb

COPY go.mod go.sum* ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o grpc-server .

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
