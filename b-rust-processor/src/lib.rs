//! KEDP Rust 처리 서버.
//!
//! Go(수집)가 제공처 응답을 페이지마다 gRPC로 넘기면(b-proto/kedp/v1/processor.proto)
//! 응답을 읽고(response), 기간을 바꾼 뒤(period), DB에 쓴다(db).
//! gRPC 호출을 받는 곳은 service, 설정을 읽는 곳은 config다.

pub mod config;
pub mod db;
pub mod period;
pub mod response;
pub mod service;

/// b-proto로 만든 코드(build.rs). 메시지 타입과 서버 trait가 들어 있다.
pub mod pb {
    tonic::include_proto!("kedp.v1");
}
