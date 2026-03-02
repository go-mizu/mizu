FROM ubuntu:24.04 AS builder
RUN apt-get update && apt-get install -y curl build-essential pkg-config && rm -rf /var/lib/apt/lists/*
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y --default-toolchain stable
ENV PATH="/root/.cargo/bin:$PATH"
ENV RUSTFLAGS="-C target-cpu=x86-64-v3"
WORKDIR /build
COPY . .
RUN cargo build --release
