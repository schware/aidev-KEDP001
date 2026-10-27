//! gRPC 서비스. b-proto/kedp/v1/processor.proto의 ProcessorService를 구현한다.
//! 흐름과 결과는 b-proto/README.md를 따른다.

use chrono::{DateTime, Utc};
use deadpool_postgres::Pool;
use tonic::{Request, Response, Status};

use crate::db::{self, DbError, RawResponse};
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
        db::check(&self.pool).await.map_err(|e| {
            let status = to_status(e);
            eprintln!("Check: {}", status.message());
            status
        })?;
        Ok(Response::new(CheckResponse {}))
    }

    /// 제공처 응답 하나(한 페이지)를 처리한다. 결과를 한 줄로 기록한다(본문과 주소는 기록하지 않는다).
    async fn process_page(
        &self,
        request: Request<ProcessPageRequest>,
    ) -> Result<Response<ProcessPageResponse>, Status> {
        let req = request.into_inner();
        let head = format!(
            "ProcessPage run_id={} indicator_id={:?} page={}",
            req.run_id, req.indicator_id, req.page
        );
        match self.process(&req).await {
            Ok(reply) => {
                eprintln!(
                    "{head} status={} inserted={} updated={} unchanged={} skipped_empty={} error_code={:?}",
                    reply.status().as_str_name(),
                    reply.inserted_count,
                    reply.updated_count,
                    reply.unchanged_count,
                    reply.skipped_empty_count,
                    reply.error_code
                );
                Ok(Response::new(reply))
            }
            Err(status) => {
                eprintln!("{head} grpc_error={:?} {}", status.code(), status.message());
                Err(status)
            }
        }
    }
}

impl Processor {
    async fn process(&self, req: &ProcessPageRequest) -> Result<ProcessPageResponse, Status> {
        // 1. 요청 값을 확인한다. 잘못됐으면 DB에 아무것도 쓰지 않고 INVALID_ARGUMENT다.
        let received_at = validate(req)?;

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

        // 3. 응답을 읽는다: 제공처 오류 판정, 기간 변환, 빈 값 건너뛰기. 여기서는 DB를 쓰지 않는다.
        let reading = response::read(&indicator, req.http_status, &req.body);

        // 4. 원본 응답은 언제나, 관측값은 정상 응답일 때만 쓴다(한 transaction).
        //    여기서 실패하면 gRPC 오류이고 DB에는 아무것도 남지 않는다.
        let raw = RawResponse {
            run_id: req.run_id,
            received_at,
            request_masked: &req.request_masked,
            http_status: req.http_status,
            body: &req.body,
        };
        let rows = match &reading {
            Reading::Success { rows, .. } => rows.as_slice(),
            _ => &[],
        };
        let counts = db::save(&mut client, &indicator, &raw, rows)
            .await
            .map_err(to_status)?;

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

/// DB 오류를 gRPC 상태로 바꾼다(b-proto/README.md "gRPC 오류").
fn to_status(e: DbError) -> Status {
    match e {
        DbError::Unavailable(m) => Status::unavailable(m),
        DbError::NotFound(m) => Status::not_found(m),
        DbError::Internal(m) => Status::internal(m),
    }
}
