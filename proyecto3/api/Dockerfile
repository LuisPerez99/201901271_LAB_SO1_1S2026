# Rust API Multi-stage Dockerfile
FROM rust:1.75 as builder

WORKDIR /app

COPY Cargo.toml Cargo.lock* ./
COPY src ./src

RUN cargo build --release

FROM debian:bookworm-slim

WORKDIR /app

RUN apt-get update && apt-get install -y \
    ca-certificates \
    libssl3 \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/target/release/api /app/api

EXPOSE 3000

ENV RUST_LOG=info

CMD ["/app/api"]
