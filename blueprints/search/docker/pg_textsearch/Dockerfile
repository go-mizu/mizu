# Dockerfile for pg_textsearch (Tiger Data BM25 extension)
# Builds the extension from source on PostgreSQL 18
#
# Note: pg_textsearch is in pre-release (v0.5.0-dev).
# GA expected February 2026.
#
# Usage:
#   docker build -t fineweb-pg-textsearch .
#   docker run -d -p 5437:5432 -e POSTGRES_PASSWORD=fineweb fineweb-pg-textsearch

FROM postgres:18

# Install build dependencies
RUN apt-get update && apt-get install -y \
    build-essential \
    git \
    postgresql-server-dev-18 \
    clang \
    llvm \
    && rm -rf /var/lib/apt/lists/*

# Clone and build pg_textsearch
WORKDIR /tmp
RUN git clone --depth 1 https://github.com/timescale/pg_textsearch.git \
    && cd pg_textsearch \
    && make \
    && make install \
    && cd .. \
    && rm -rf pg_textsearch

# Create initialization script to enable extension
RUN mkdir -p /docker-entrypoint-initdb.d
COPY init-pg_textsearch.sql /docker-entrypoint-initdb.d/

# Reset working directory
WORKDIR /

# Expose PostgreSQL port
EXPOSE 5432
