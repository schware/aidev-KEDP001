//! 시험 DB(kedp_rust_check) 테스트. kedp_rust 계정으로 접속해 실제로 쓴다.
//! scripts/db-test.sh가 시험 DB를 만들고(migration + tests/fixtures.sql) 이 테스트를 돌린 뒤 지운다.
//! 기본 `cargo test`에서는 돌지 않는다(#[ignore]).

use std::path::Path;

use deadpool_postgres::Pool;
use kedp_processor::config::Config;
use kedp_processor::db;
use kedp_processor::pb::processor_service_server::ProcessorService;
use kedp_processor::pb::{CheckRequest, ProcessPageRequest, ProcessPageResponse, ProcessStatus};
use kedp_processor::service::Processor;
use tonic::{Code, Request};

const TEST_DB: &str = "kedp_rust_check";

async fn setup() -> (Processor, Pool) {
    let env = Path::new(env!("CARGO_MANIFEST_DIR")).join("../.env");
    let cfg = Config::load(&env).expect("config");
    // kedp DB에 쓰지 않도록 시험 DB에서만 돌린다.
    assert_eq!(
        cfg.db.name, TEST_DB,
        "DB 테스트는 시험 DB에서만 돌린다(scripts/db-test.sh)"
    );
    let pool = db::connect(&cfg.db).expect("pool");
    (Processor::new(pool.clone()), pool)
}

fn sample(path: &str) -> Vec<u8> {
    let full = format!("{}/../samples/{path}", env!("CARGO_MANIFEST_DIR"));
    std::fs::read(&full).unwrap_or_else(|e| panic!("{full}: {e}"))
}

/// ECOS 기준금리 샘플(33행)의 202401 값(3.5)을 바꾼 본문
fn ecos_with_first_value(value: &str) -> Vec<u8> {
    let body = String::from_utf8(sample("ecos/722Y001-0101000-M-202401-202609.json")).unwrap();
    let from = r#""TIME":"202401","DATA_VALUE":"3.5""#;
    assert!(body.contains(from));
    body.replacen(from, &format!(r#""TIME":"202401","DATA_VALUE":{value}"#), 1)
        .into_bytes()
}

fn request(run_id: i64, indicator_id: &str, http_status: i32, body: Vec<u8>) -> ProcessPageRequest {
    ProcessPageRequest {
        run_id,
        indicator_id: indicator_id.into(),
        page: 1,
        // 2026-10-27 03:00:00 UTC
        received_at: Some(prost_types::Timestamp {
            seconds: 1_793_070_000,
            nanos: 0,
        }),
        http_status,
        request_masked: "https://example.test/api/***/".into(),
        body,
    }
}

async fn process(p: &Processor, req: ProcessPageRequest) -> ProcessPageResponse {
    p.process_page(Request::new(req))
        .await
        .expect("process_page")
        .into_inner()
}

/// (결과, 새로 넣은 행, 덮어쓴 행, 같은 행, 빈 값)
fn counts(r: &ProcessPageResponse) -> (ProcessStatus, i32, i32, i32, i32) {
    (
        r.status(),
        r.inserted_count,
        r.updated_count,
        r.unchanged_count,
        r.skipped_empty_count,
    )
}

/// run_id로 쓴 원본 응답. JSON이면(body) true, 아니면(body_text) false
async fn raw_responses(pool: &Pool, run_id: i64) -> Vec<bool> {
    let client = pool.get().await.unwrap();
    client
        .query(
            "SELECT body IS NOT NULL FROM raw_response WHERE run_id = $1 ORDER BY id",
            &[&run_id],
        )
        .await
        .unwrap()
        .iter()
        .map(|row| row.get(0))
        .collect()
}

async fn observation_count(pool: &Pool, indicator_id: &str) -> i64 {
    let client = pool.get().await.unwrap();
    client
        .query_one(
            "SELECT count(*) FROM observation WHERE indicator_id = $1",
            &[&indicator_id],
        )
        .await
        .unwrap()
        .get(0)
}

#[tokio::test]
#[ignore = "시험 DB가 필요하다(scripts/db-test.sh)"]
async fn check_ok() {
    let (p, _) = setup().await;
    p.check(Request::new(CheckRequest {})).await.expect("check");
}

#[tokio::test]
#[ignore = "시험 DB가 필요하다(scripts/db-test.sh)"]
async fn ecos_insert_unchanged_updated() {
    let (p, pool) = setup().await;

    // run 1: 처음이라 33행을 모두 새로 넣는다.
    let r = process(
        &p,
        request(1, "ecos-base-rate", 200, ecos_with_first_value(r#""3.5""#)),
    )
    .await;
    assert_eq!(counts(&r), (ProcessStatus::Success, 33, 0, 0, 0));
    assert_eq!((r.error_code.as_str(), r.error_message.as_str()), ("", ""));

    // run 2: 202401을 3.50으로 받았다. 숫자로는 같으므로 33행 모두 같은 값이다(이력 없음).
    let r = process(
        &p,
        request(2, "ecos-base-rate", 200, ecos_with_first_value(r#""3.50""#)),
    )
    .await;
    assert_eq!(counts(&r), (ProcessStatus::Success, 0, 0, 33, 0));

    // run 3: 202401이 3.25로 바뀌었다. 1행만 덮어쓰고, 이력은 트리거가 남긴다(tests/check_history.sql에서 확인).
    let r = process(
        &p,
        request(3, "ecos-base-rate", 200, ecos_with_first_value(r#""3.25""#)),
    )
    .await;
    assert_eq!(counts(&r), (ProcessStatus::Success, 0, 1, 32, 0));

    let client = pool.get().await.unwrap();
    let row = client
        .query_one(
            "SELECT value::text, value_raw, period_raw, run_id, collected_at = to_timestamp(1793070000)
             FROM observation WHERE indicator_id = 'ecos-base-rate' AND cycle = 'M' AND period_start = '2024-01-01'",
            &[],
        )
        .await
        .unwrap();
    let got: (String, String, String, i64, bool) =
        (row.get(0), row.get(1), row.get(2), row.get(3), row.get(4));
    assert_eq!(
        got,
        ("3.25".into(), "3.25".into(), "202401".into(), 3, true)
    );
    assert_eq!(observation_count(&pool, "ecos-base-rate").await, 33);

    // 원본 응답은 실행마다 1행, JSON이라 body에 들어간다.
    for run in 1..=3 {
        assert_eq!(raw_responses(&pool, run).await, vec![true], "run {run}");
    }
}

#[tokio::test]
#[ignore = "시험 DB가 필요하다(scripts/db-test.sh)"]
async fn kosis_and_data_go_kr() {
    let (p, pool) = setup().await;
    let client = pool.get().await.unwrap();

    // KOSIS 월: LST_CHN_DE가 source_changed_at에 들어간다.
    let body = sample("kosis/DT_1DA7001S-T90-0-M-202401-202609.json");
    let r = process(&p, request(4, "kosis-employment-rate", 200, body)).await;
    assert_eq!(counts(&r), (ProcessStatus::Success, 33, 0, 0, 0));
    let changed: Option<chrono::NaiveDate> = client
        .query_one(
            "SELECT source_changed_at FROM observation
             WHERE indicator_id = 'kosis-employment-rate' AND cycle = 'M' AND period_start = '2024-01-01'",
            &[],
        )
        .await
        .unwrap()
        .get(0);
    assert_eq!(changed, chrono::NaiveDate::from_ymd_opt(2024, 2, 1));

    // KOSIS 연: 주기 A, 기간 시작일은 1월 1일
    let body = sample("kosis/INH_1B8000F_01-T1-11-Y-2015-2025.json");
    let r = process(&p, request(5, "kosis-births", 200, body)).await;
    assert_eq!(counts(&r), (ProcessStatus::Success, 11, 0, 0, 0));
    let value: String = client
        .query_one(
            "SELECT value::text FROM observation
             WHERE indicator_id = 'kosis-births' AND cycle = 'A' AND period_start = '2015-01-01'",
            &[],
        )
        .await
        .unwrap()
        .get(0);
    assert_eq!(value, "438420");

    // 공공데이터포털 일: 값은 clpr
    let body = sample("data-go-kr/069500-D-20260901-20261022.json");
    let r = process(&p, request(6, "data-go-kr-kodex-200", 200, body)).await;
    assert_eq!(counts(&r), (ProcessStatus::Success, 34, 0, 0, 0));
    let value: String = client
        .query_one(
            "SELECT value::text FROM observation
             WHERE indicator_id = 'data-go-kr-kodex-200' AND cycle = 'D' AND period_start = '2026-10-22'",
            &[],
        )
        .await
        .unwrap()
        .get(0);
    assert_eq!(value, "86525");
}

#[tokio::test]
#[ignore = "시험 DB가 필요하다(scripts/db-test.sh)"]
async fn skips_empty_values() {
    let (p, pool) = setup().await;
    // 202401은 null, 202402는 빈 문자열이다.
    let body = String::from_utf8(ecos_with_first_value("null")).unwrap();
    let body = body.replacen(
        r#""TIME":"202402","DATA_VALUE":"3.5""#,
        r#""TIME":"202402","DATA_VALUE":"""#,
        1,
    );
    let r = process(&p, request(7, "ecos-empty", 200, body.into_bytes())).await;
    assert_eq!(counts(&r), (ProcessStatus::Success, 31, 0, 0, 2));
    assert_eq!(observation_count(&pool, "ecos-empty").await, 31);
}

#[tokio::test]
#[ignore = "시험 DB가 필요하다(scripts/db-test.sh)"]
async fn invalid_value_writes_no_observation() {
    let (p, pool) = setup().await;
    let r = process(
        &p,
        request(8, "ecos-invalid", 200, ecos_with_first_value(r#""-""#)),
    )
    .await;
    assert_eq!(counts(&r), (ProcessStatus::Error, 0, 0, 0, 0));
    assert_eq!(r.error_code, "INVALID_RESPONSE");
    assert!(
        r.error_message.starts_with("page 1: row 1: "),
        "{}",
        r.error_message
    );
    // 원본 응답만 남고 관측값은 하나도 쓰지 않는다.
    assert_eq!(observation_count(&pool, "ecos-invalid").await, 0);
    assert_eq!(raw_responses(&pool, 8).await, vec![true]);
}

#[tokio::test]
#[ignore = "시험 DB가 필요하다(scripts/db-test.sh)"]
async fn no_data_and_errors_keep_raw_response() {
    let (p, pool) = setup().await;
    let html = b"<html><body>502 Bad Gateway</body></html>".to_vec();
    let cases = [
        (
            9,
            "ecos-base-rate",
            200,
            sample("ecos/error-INFO-200.json"),
            ProcessStatus::NoData,
            "INFO-200",
            true,
        ),
        (
            10,
            "kosis-employment-rate",
            200,
            sample("kosis/error-11.json"),
            ProcessStatus::Error,
            "11",
            true,
        ),
        (
            11,
            "data-go-kr-kodex-200",
            403,
            sample("data-go-kr/error-30.json"),
            ProcessStatus::Error,
            "30",
            true,
        ),
        // JSON이 아닌 응답은 body_text에 들어간다.
        (
            12,
            "ecos-base-rate",
            502,
            html,
            ProcessStatus::Error,
            "HTTP_502",
            false,
        ),
    ];
    for (run, indicator, http_status, body, status, code, json) in cases {
        let r = process(&p, request(run, indicator, http_status, body)).await;
        assert_eq!(counts(&r), (status, 0, 0, 0, 0), "run {run}");
        assert_eq!(r.error_code, code, "run {run}");
        assert_eq!(raw_responses(&pool, run).await, vec![json], "run {run}");
    }
}

#[tokio::test]
#[ignore = "시험 DB가 필요하다(scripts/db-test.sh)"]
async fn grpc_errors_write_nothing() {
    let (p, pool) = setup().await;
    let ok = || {
        request(
            13,
            "ecos-base-rate",
            200,
            sample("ecos/error-INFO-200.json"),
        )
    };
    let cases = [
        (
            ProcessPageRequest {
                request_masked: "https://example.test/api/KEY/".into(),
                ..ok()
            },
            Code::InvalidArgument,
        ),
        (
            ProcessPageRequest { page: 0, ..ok() },
            Code::InvalidArgument,
        ),
        (
            ProcessPageRequest {
                received_at: None,
                ..ok()
            },
            Code::InvalidArgument,
        ),
        (
            ProcessPageRequest {
                indicator_id: "no-such-indicator".into(),
                ..ok()
            },
            Code::NotFound,
        ),
        (
            ProcessPageRequest {
                run_id: 999,
                ..ok()
            },
            Code::NotFound,
        ),
    ];
    for (req, code) in cases {
        let status = p
            .process_page(Request::new(req))
            .await
            .expect_err("grpc error");
        assert_eq!(status.code(), code, "{}", status.message());
    }
    assert_eq!(raw_responses(&pool, 13).await, Vec::<bool>::new());
    assert_eq!(raw_responses(&pool, 999).await, Vec::<bool>::new());
}
