# Go API Client Multi-stage Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy the shared protobuf module first so the local replace works during build.
COPY go-shared-pb ./go-shared-pb
COPY go-api-client/go.mod go-api-client/go.sum* ./go-api-client/

WORKDIR /app/go-api-client
RUN go mod download

COPY go-api-client/. .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o api-client .

FROM alpine:3.18

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/go-api-client/api-client .

EXPOSE 8080

ENV HTTP_PORT=:8080
ENV GRPC_SERVER_ADDR=go-grpc-server:50051

CMD ["./api-client"]
