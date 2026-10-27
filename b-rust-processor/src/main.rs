//! 처리 서버 시작점.
//!
//! 1. .env 파일과 환경 변수에서 설정을 읽는다.
//! 2. DB 연결 pool을 만든다. 실제 접속은 처음 호출을 받을 때 한다.
//! 3. gRPC 서버를 열고, Ctrl+C(SIGINT)나 SIGTERM(docker stop)을 받으면 멈춘다.

use std::path::PathBuf;
use std::process::ExitCode;

use kedp_processor::config::Config;
use kedp_processor::db;
use kedp_processor::pb::processor_service_server::ProcessorServiceServer;
use kedp_processor::service::Processor;
use tokio::signal::unix::{SignalKind, signal};
use tonic::transport::Server;

#[tokio::main]
async fn main() -> ExitCode {
    match run().await {
        Ok(()) => ExitCode::SUCCESS,
        Err(e) => {
            eprintln!("error: {e}");
            ExitCode::FAILURE
        }
    }
}

async fn run() -> Result<(), String> {
    let cfg = Config::load(&env_path()?)?;
    let pool = db::connect(&cfg.db)?;

    eprintln!("kedp-processor listening on {}", cfg.addr);
    Server::builder()
        .add_service(ProcessorServiceServer::new(Processor::new(pool)))
        .serve_with_shutdown(cfg.addr, shutdown())
        .await
        .map_err(|e| format!("serve: {e}"))?;
    eprintln!("kedp-processor stopped");
    Ok(())
}

/// .env 파일 위치. 기본은 ../.env다(b-rust-processor에서 실행할 때 저장소 최상위). --env 경로로 바꾼다.
fn env_path() -> Result<PathBuf, String> {
    let args: Vec<String> = std::env::args().skip(1).collect();
    match args.as_slice() {
        [] => Ok(PathBuf::from("../.env")),
        [flag, path] if flag == "--env" => Ok(PathBuf::from(path)),
        _ => Err("usage: kedp-processor [--env PATH]".to_string()),
    }
}

/// 멈추라는 신호를 기다린다.
async fn shutdown() {
    let mut term = signal(SignalKind::terminate()).expect("listen SIGTERM");
    tokio::select! {
        _ = tokio::signal::ctrl_c() => {}
        _ = term.recv() => {}
    }
}
