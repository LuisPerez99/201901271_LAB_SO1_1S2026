# syntax=docker/dockerfile:1.4
# Go API Client Multi-stage Dockerfile
FROM golang:1.26-alpine AS builder

RUN apk --no-cache add ca-certificates

WORKDIR /app/go-api-client

# Copy shared protobuf module from an extra build context named "sharedpb".
# This keeps replace ../go-shared-pb working without cloning from GitHub.
COPY --from=sharedpb . /app/go-shared-pb

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o api-client .

FROM alpine:3.18

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/go-api-client/api-client .

EXPOSE 8080

ENV HTTP_PORT=:8080
ENV GRPC_SERVER_ADDR=go-grpc-server:50051

CMD ["./api-client"]
