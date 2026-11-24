//! gRPC 서비스. b-proto/kedp/v1/processor.proto의 ProcessorService를 구현한다.
//! 흐름과 결과는 b-proto/README.md를 따른다. 응답 단위(indicator.source_unit)도 함께 쓴다(Issue #76).
//! 로그(KEDP001-D026): grpc.check, process.page, process.step, source_unit.changed

use std::time::Instant;

use chrono::{DateTime, Utc};
use deadpool_postgres::Pool;
use tonic::{Request, Response, Status};

use crate::db::{self, DbError, RawResponse};
use crate::log::Level;
use crate::pb::processor_service_server::ProcessorService;
use crate::pb::{
    CheckRequest, CheckResponse, ProcessPageRequest, ProcessPageResponse, ProcessStatus,
};
use crate::response::{self, Reading};

/// 처리 서버. DB 연결 pool만 가지고 있다.
pub struct Processor {
    pool: Pool,
}

impl Processor {
    pub fn new(pool: Pool) -> Self {
        Processor { pool }
    }
}

#[tonic::async_trait]
impl ProcessorService for Processor {
    /// 상태 확인. DB에 접속해 indicator를 읽을 수 있으면 OK, 아니면 UNAVAILABLE이다.
    /// Go는 시작할 때 부르고 실패하면 멈춘다(KEDP001-D020). DB에 아무것도 쓰지 않는다.
    async fn check(
        &self,
        _request: Request<CheckRequest>,
    ) -> Result<Response<CheckResponse>, Status> {
        // 성공은 Go가 processor.check(INFO)로 남기므로 여기서는 DEBUG다.
        db::check(&self.pool).await.map_err(|e| {
            let status = to_status(e);
            log_error!("grpc.check", "ok" => false, "grpc_code" => code_name(&status), "err" => status.message());
            status
        })?;
        log_debug!("grpc.check", "ok" => true);
        Ok(Response::new(CheckResponse {}))
    }

    /// 제공처 응답 하나(한 페이지)를 처리한다. 결과를 process.page 한 줄로 남긴다(본문과 주소는 남기지 않는다).
    /// SUCCESS, NO_DATA는 INFO, 제공처 오류 판정(ERROR)은 WARN, gRPC 오류는 ERROR다.
    async fn process_page(
        &self,
        request: Request<ProcessPageRequest>,
    ) -> Result<Response<ProcessPageResponse>, Status> {
        let req = request.into_inner();
        let started = Instant::now();
        let result = self.process(&req).await;
        let ms = started.elapsed().as_millis();
        match result {
            Ok(reply) => {
                let lv = match reply.status() {
                    ProcessStatus::Success | ProcessStatus::NoData => Level::Info,
                    _ => Level::Warn,
                };
                log_at!(lv, "process.page",
                    "run" => req.run_id,
                    "indicator" => &req.indicator_id,
                    "page" => req.page,
                    "status" => reply.status().as_str_name().trim_start_matches("PROCESS_STATUS_"),
                    "inserted" => reply.inserted_count,
                    "updated" => reply.updated_count,
                    "unchanged" => reply.unchanged_count,
                    "skipped_empty" => reply.skipped_empty_count,
                    "code" => &reply.error_code,
                    "ms" => ms,
                );
                Ok(Response::new(reply))
            }
            Err(status) => {
                log_error!("process.page",
                    "run" => req.run_id,
                    "indicator" => &req.indicator_id,
                    "page" => req.page,
                    "grpc_code" => code_name(&status),
                    "ms" => ms,
                    "err" => status.message(),
                );
                Err(status)
            }
        }
    }
}

impl Processor {
    async fn process(&self, req: &ProcessPageRequest) -> Result<ProcessPageResponse, Status> {
        // 단계마다 걸린 시간을 process.step(DEBUG)으로 남긴다. 실패한 단계는 process.page(ERROR)가 남긴다.
        let mut started = Instant::now();
        let mut step = |name: &str| {
            log_debug!("process.step",
                "run" => req.run_id,
                "indicator" => &req.indicator_id,
                "page" => req.page,
                "step" => name,
                "ms" => started.elapsed().as_millis(),
            );
            started = Instant::now();
        };

        // 1. 요청 값을 확인한다. 잘못됐으면 DB에 아무것도 쓰지 않고 INVALID_ARGUMENT다.
        let received_at = validate(req)?;
        step("validate");

        // 2. 지표 정의(제공처, 코드, 주기, value_field)를 indicator 테이블에서 읽는다. 없으면 NOT_FOUND다.
        let mut client = db::client(&self.pool).await.map_err(to_status)?;
        let indicator = db::load_indicator(&client, &req.indicator_id)
            .await
            .map_err(to_status)?
            .ok_or_else(|| {
                Status::not_found(format!(
                    "indicator {:?} is not in indicator",
                    req.indicator_id
                ))
            })?;

        step("load_indicator");

        // 3. 응답을 읽는다: 제공처 오류 판정, 기간 변환, 빈 값 건너뛰기. 여기서는 DB를 쓰지 않는다.
        let reading = response::read(&indicator, req.http_status, &req.body);
        step("read");

        // 4. 원본 응답은 언제나, 관측값은 정상 응답일 때만 쓴다(한 transaction).
        //    여기서 실패하면 gRPC 오류이고 DB에는 아무것도 남지 않는다.
        let raw = RawResponse {
            run_id: req.run_id,
            received_at,
            request_masked: &req.request_masked,
            http_status: req.http_status,
            body: &req.body,
        };
        let (rows, unit) = match &reading {
            Reading::Success { rows, unit, .. } => (rows.as_slice(), unit.as_deref()),
            _ => (&[][..], None),
        };
        let counts = db::save(&mut client, &indicator, &raw, rows, unit)
            .await
            .map_err(to_status)?;
        step("save");
        // 응답 단위가 바뀌었으면(처음 채움, 기준연도 개편 등) 지표 id, 이전 단위, 새 단위를 남긴다.
        // 이전 단위가 없으면(처음 채움) old는 ""다.
        if let Some((old, new)) = &counts.source_unit {
            log_warn!("source_unit.changed",
                "run" => req.run_id,
                "indicator" => &indicator.id,
                "old" => old.as_deref().unwrap_or(""),
                "new" => new,
            );
        }

        // 5. 결과를 돌려준다. 오류 메시지에는 페이지 번호를 붙인다.
        Ok(match reading {
            Reading::Success { skipped_empty, .. } => ProcessPageResponse {
                status: ProcessStatus::Success.into(),
                inserted_count: counts.inserted,
                updated_count: counts.updated,
                unchanged_count: counts.unchanged,
                skipped_empty_count: skipped_empty,
                ..Default::default()
            },
            Reading::NoData { code, message } => {
                failed(ProcessStatus::NoData, code, message, req.page)
            }
            Reading::Error { code, message } => {
                failed(ProcessStatus::Error, code, message, req.page)
            }
        })
    }
}

/// 데이터 없음·오류 결과. 행 수는 모두 0이다.
fn failed(status: ProcessStatus, code: String, message: String, page: i32) -> ProcessPageResponse {
    ProcessPageResponse {
        status: status.into(),
        error_code: code,
        error_message: format!("page {page}: {message}"),
        ..Default::default()
    }
}

/// 요청 값을 확인하고 받은 시각을 돌려준다(b-proto/README.md "gRPC 오류"의 INVALID_ARGUMENT).
fn validate(req: &ProcessPageRequest) -> Result<DateTime<Utc>, Status> {
    let invalid = |message: &str| Err(Status::invalid_argument(message));
    if req.run_id <= 0 {
        return invalid("run_id must be positive");
    }
    if req.indicator_id.is_empty() {
        return invalid("indicator_id is empty");
    }
    if req.page < 1 {
        return invalid("page must be 1 or more");
    }
    if !(100..=599).contains(&req.http_status) {
        return invalid("http_status must be between 100 and 599");
    }
    // 키를 가리지 않은 주소는 받지 않는다. DB도 같은 검사를 한다(raw_response.request_masked).
    if !req.request_masked.contains("***") {
        return invalid("request_masked has no ***");
    }
    let Some(ts) = &req.received_at else {
        return invalid("received_at is missing");
    };
    match u32::try_from(ts.nanos)
        .ok()
        .and_then(|nanos| DateTime::from_timestamp(ts.seconds, nanos))
    {
        Some(t) => Ok(t),
        None => invalid("received_at is out of range"),
    }
}

/// gRPC 상태 이름(UNAVAILABLE 등)이다.
fn code_name(status: &Status) -> String {
    format!("{:?}", status.code())
}

/// DB 오류를 gRPC 상태로 바꾼다(b-proto/README.md "gRPC 오류").
fn to_status(e: DbError) -> Status {
    match e {
        DbError::Unavailable(m) => Status::unavailable(m),
        DbError::NotFound(m) => Status::not_found(m),
        DbError::Internal(m) => Status::internal(m),
    }
}
