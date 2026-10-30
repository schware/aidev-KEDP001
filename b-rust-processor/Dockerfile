# syntax=docker/dockerfile:1
# Rust 처리 서버 image: kedp-rust. build는 deploy/build.sh로 한다(KEDP001-D023).
# build.rs가 b-proto의 processor.proto를 읽으므로 build context `proto`(b-proto 폴더)를 함께 준다:
#   docker build --build-context proto=../b-proto -t kedp-rust:<tag> .
# 실행 image에는 정적 실행 파일(musl)만 둔다. 시각은 UTC로 계산하므로 시간대 정보가 필요 없다.
ARG RUST_IMAGE=rust:1.99-alpine

FROM ${RUST_IMAGE} AS build
RUN apk add --no-cache musl-dev protoc protobuf-dev
WORKDIR /src/b-rust-processor
COPY --from=proto . /src/b-proto
COPY Cargo.toml Cargo.lock build.rs ./
COPY src ./src
# dell-debian 메모리가 작으므로 2개씩만 컴파일한다(.claude/environment.md).
ENV CARGO_BUILD_JOBS=2
RUN cargo build --release --locked && strip target/release/kedp-processor

FROM scratch
COPY --from=build /src/b-rust-processor/target/release/kedp-processor /kedp-processor
ENV TZ=Asia/Seoul
USER 65532:65532
EXPOSE 50051
ENTRYPOINT ["/kedp-processor"]
