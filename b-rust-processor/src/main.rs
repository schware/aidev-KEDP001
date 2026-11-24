//! 처리 서버 시작점.
//!
//! 1. 로그 level(KEDP_LOG_LEVEL)을 읽는다(KEDP001-D026).
//! 2. .env 파일과 환경 변수에서 설정을 읽는다.
//! 3. DB 연결 pool을 만든다. 실제 접속은 처음 호출을 받을 때 한다.
//! 4. gRPC 서버를 열고, Ctrl+C(SIGINT)나 SIGTERM(docker stop)을 받으면 멈춘다.

use std::path::{Path, PathBuf};
use std::process::ExitCode;

use kedp_processor::config::{self, Config};
use kedp_processor::db;
use kedp_processor::log::{self, Level};
use kedp_processor::pb::processor_service_server::ProcessorServiceServer;
use kedp_processor::service::Processor;
use kedp_processor::{log_error, log_info, log_warn};
use tokio::signal::unix::{SignalKind, signal};
use tonic::transport::Server;

#[tokio::main]
async fn main() -> ExitCode {
    let result = match env_path() {
        Ok(path) => {
            setup_log(&path);
            run(&path).await
        }
        Err(e) => Err(e),
    };
    match result {
        Ok(reason) => {
            log_info!("server.stop", "reason" => reason);
            ExitCode::SUCCESS
        }
        Err(e) => {
            // 멈추게 한 오류를 ERROR 한 줄로 남기고, 이어서 server.stop을 남긴다(D026 3장).
            log_error!("server.error", "err" => e);
            log_info!("server.stop", "reason" => "error");
            ExitCode::FAILURE
        }
    }
}

/// KEDP_LOG_LEVEL을 읽어 로그를 준비한다. 값이 틀리면 INFO로 두고 config.log_level(WARN)을 남긴다.
/// .env를 읽지 못하면 INFO로 둔다(그 오류는 Config::load에서 다시 나온다).
fn setup_log(path: &Path) {
    let raw = config::lookup(path, log::LEVEL_VAR)
        .ok()
        .flatten()
        .unwrap_or_default();
    match log::parse_level(&raw) {
        Some(lv) => log::set_level(lv),
        None => {
            log::set_level(Level::Info);
            log_warn!("config.log_level", "value" => raw, "use" => Level::Info.as_str());
        }
    }
}

/// 서버를 돌리고, 멈춘 이유를 돌려준다.
async fn run(path: &Path) -> Result<&'static str, String> {
    let cfg = Config::load(path)?;
    let pool = db::connect(&cfg.db)?;

    log_info!("server.start", "addr" => cfg.addr, "log_level" => log::level().as_str());
    Server::builder()
        .add_service(ProcessorServiceServer::new(Processor::new(pool)))
        .serve_with_shutdown(cfg.addr, shutdown())
        .await
        .map_err(|e| format!("serve: {e}"))?;
    Ok("signal")
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
