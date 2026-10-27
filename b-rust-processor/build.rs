// 빌드할 때 b-proto의 약속서(processor.proto)로 gRPC 서버 코드를 만든다.
// 만든 코드는 target/ 아래에 생기고, src/lib.rs의 tonic::include_proto!가 가져다 쓴다.
// protoc가 PATH에 있어야 한다.
fn main() -> Result<(), Box<dyn std::error::Error>> {
    tonic_prost_build::configure()
        // Rust는 서버만 만든다. 클라이언트(Go)는 b-go-end에서 만든다.
        .build_client(false)
        .compile_protos(&["../b-proto/kedp/v1/processor.proto"], &["../b-proto"])?;
    Ok(())
}
