FROM rust:1.90-bookworm AS build

WORKDIR /src

COPY collectors/Cargo.toml ./Cargo.toml
COPY collectors/common ./common
COPY collectors/discovery-collector ./discovery-collector
COPY collectors/dns-collector ./dns-collector
COPY collectors/flow-collector ./flow-collector

ARG BIN_NAME

RUN cargo build --release -p ${BIN_NAME} && cp target/release/${BIN_NAME} /tmp/collector

FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends iproute2 iputils-ping fping ca-certificates ieee-data \
    && rm -rf /var/lib/apt/lists/* \
    && useradd -r -u 10001 collector

COPY --from=build /tmp/collector /usr/local/bin/collector

USER collector

ENTRYPOINT ["/usr/local/bin/collector"]
