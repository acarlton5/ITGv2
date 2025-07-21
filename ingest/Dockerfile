FROM rust:latest as builder
WORKDIR /rust/src/
COPY . .
RUN cargo install --path .


FROM debian:buster-slim as itg-ingest
RUN useradd -M -s /bin/bash itg
WORKDIR /data
RUN chown itg:root /data
COPY --from=builder --chown=itg:itg /usr/local/cargo/bin/itg-ingest /usr/local/bin/itg-ingest

USER itg
CMD ["itg-ingest"]

EXPOSE 8084