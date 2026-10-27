//! DB 읽기와 쓰기. kedp_rust 계정으로 접속한다(KEDP001-D022 14, 설계서 4.1).
//!
//! - 읽는 테이블: indicator
//! - 쓰는 테이블: raw_response, observation
//!   (observation_history는 observation의 트리거 observation_keep_history가 쓴다)

use std::time::Duration;

use chrono::{DateTime, Utc};
use deadpool_postgres::{
    Client, Config as PgConfig, ManagerConfig, Pool, PoolConfig, PoolError, RecyclingMethod,
    Runtime, Timeouts,
};
use tokio_postgres::NoTls;
use tokio_postgres::error::SqlState;

use crate::config::{DB_USER, DbConfig};
use crate::period::{Cycle, Source};
use crate::response::{Indicator, Observation};

/// DB 오류. service가 gRPC 상태로 바꾼다(b-proto/README.md "gRPC 오류").
#[derive(Debug)]
pub enum DbError {
    /// DB에 접속할 수 없다 → UNAVAILABLE
    Unavailable(String),
    /// 참조하는 행이 없다(run_id가 collection_run에 없다) → NOT_FOUND
    NotFound(String),
    /// 그 밖의 이유로 읽거나 쓰지 못했다 → INTERNAL
    Internal(String),
}

/// raw_response에 넣을 원본 응답. Go가 넘긴 값 그대로다.
pub struct RawResponse<'a> {
    pub run_id: i64,
    pub received_at: DateTime<Utc>,
    pub request_masked: &'a str,
    pub http_status: i32,
    pub body: &'a [u8],
}

/// observation에 쓴 행 수
#[derive(Debug, Default, PartialEq)]
pub struct Counts {
    /// 새로 넣은 행
    pub inserted: i32,
    /// 값이 달라져 덮어쓴 행(트리거가 이력을 남긴 행)
    pub updated: i32,
    /// 값이 같았던 행
    pub unchanged: i32,
}

/// 연결 pool을 만든다. 실제 접속은 처음 쓸 때 한다.
pub fn connect(cfg: &DbConfig) -> Result<Pool, String> {
    let mut pg = PgConfig::new();
    pg.host = Some(cfg.host.clone());
    pg.port = Some(cfg.port);
    pg.dbname = Some(cfg.name.clone());
    pg.user = Some(DB_USER.to_string());
    pg.password = Some(cfg.password.clone());
    pg.connect_timeout = Some(Duration::from_secs(5));
    // pool에서 꺼낼 때 끊긴 연결이면 버리고 새로 접속한다(DB를 다시 켰을 때).
    pg.manager = Some(ManagerConfig {
        recycling_method: RecyclingMethod::Fast,
    });
    let wait = Some(Duration::from_secs(10));
    pg.pool = Some(PoolConfig {
        max_size: 4,
        timeouts: Timeouts {
            wait,
            create: wait,
            recycle: wait,
        },
        ..PoolConfig::default()
    });
    pg.create_pool(Some(Runtime::Tokio1), NoTls)
        .map_err(|e| format!("create DB pool: {e}"))
}

/// pool에서 연결 하나를 꺼낸다.
pub async fn client(pool: &Pool) -> Result<Client, DbError> {
    pool.get().await.map_err(|e| {
        // 접속할 때 DB가 거절했으면(예: DB가 없음, 비밀번호가 틀림) DB가 준 메시지를 쓴다.
        // 이 메시지에는 비밀번호가 들어 있지 않다.
        let message = match &e {
            PoolError::Backend(pg) => match pg.as_db_error() {
                Some(db) => db.message().to_string(),
                None => describe(pg),
            },
            other => other.to_string(),
        };
        DbError::Unavailable(format!("connect DB: {message}"))
    })
}

/// 상태 확인. 접속해서 indicator를 읽을 수 있는지 본다. 아무것도 쓰지 않는다.
pub async fn check(pool: &Pool) -> Result<(), DbError> {
    let client = client(pool).await?;
    client
        .query_opt("SELECT 1 FROM indicator LIMIT 1", &[])
        .await
        .map_err(db_error)?;
    Ok(())
}

/// indicator에서 지표 정의를 읽는다. 없으면 None이다.
pub async fn load_indicator(client: &Client, id: &str) -> Result<Option<Indicator>, DbError> {
    let row = client
        .query_opt(
            "SELECT source_code, org_id, stat_code, item_code, classes, value_field, cycle
             FROM indicator WHERE id = $1",
            &[&id],
        )
        .await
        .map_err(db_error)?;
    let Some(row) = row else {
        return Ok(None);
    };
    // source_code와 cycle은 DB가 값을 제한한다(source 테이블, CHECK). 모르는 값이면 스키마가 바뀐 것이다.
    let source_code: String = row.get("source_code");
    let cycle: String = row.get("cycle");
    let source = Source::from_code(&source_code).ok_or_else(|| {
        DbError::Internal(format!(
            "indicator {id}: unknown source_code {source_code:?}"
        ))
    })?;
    let cycle = Cycle::from_code(&cycle)
        .ok_or_else(|| DbError::Internal(format!("indicator {id}: unknown cycle {cycle:?}")))?;
    Ok(Some(Indicator {
        id: id.to_string(),
        source,
        org_id: row.get("org_id"),
        stat_code: row.get("stat_code"),
        item_code: row.get("item_code"),
        classes: row.get("classes"),
        value_field: row.get("value_field"),
        cycle,
    }))
}

/// 원본 응답과 관측값을 한 transaction으로 쓴다. 중간에 실패하면 아무것도 남지 않는다.
/// 데이터 없음·오류 응답은 rows가 비어 있어 원본 응답만 쓴다(KEDP001-D022 13).
pub async fn save(
    client: &mut Client,
    ind: &Indicator,
    raw: &RawResponse<'_>,
    rows: &[Observation],
) -> Result<Counts, DbError> {
    let tx = client.transaction().await.map_err(db_error)?;
    insert_raw_response(&tx, raw).await?;
    let counts = upsert_observations(&tx, ind, raw, rows).await?;
    tx.commit().await.map_err(db_error)?;
    Ok(counts)
}

async fn insert_raw_response(
    tx: &deadpool_postgres::Transaction<'_>,
    raw: &RawResponse<'_>,
) -> Result<(), DbError> {
    // JSON 응답은 body(jsonb), JSON이 아닌 응답은 body_text에 넣는다. 둘 중 하나만 쓴다(설계서 5.3).
    // jsonb에는 원문 글자를 넘겨 DB가 읽게 한다(Rust에서 숫자를 다시 쓰지 않도록).
    let (json, text) = match std::str::from_utf8(raw.body) {
        Ok(s) if serde_json::from_str::<serde_json::Value>(s).is_ok() => (Some(s), None),
        // text 칸은 UTF-8이어야 하고 NUL 글자를 넣을 수 없다. 그런 바이트는 U+FFFD로 바꾼다.
        _ => (
            None,
            Some(String::from_utf8_lossy(raw.body).replace('\0', "\u{FFFD}")),
        ),
    };
    tx.execute(
        "INSERT INTO raw_response (run_id, received_at, request_masked, http_status, body, body_text)
         VALUES ($1, $2, $3, $4, $5::text::jsonb, $6)",
        &[
            &raw.run_id,
            &raw.received_at,
            &raw.request_masked,
            &raw.http_status,
            &json,
            &text,
        ],
    )
    .await
    .map_err(db_error)?;
    Ok(())
}

/// 같은 지표·주기·기간의 행이 이미 있으면, 그 값이 새 값과 숫자로 같은지 돌려준다.
/// 3.5와 3.50은 같다(설계서 5.2). 덮어쓰기 전까지 다른 쪽이 바꾸지 못하게 잠근다(FOR UPDATE).
const SELECT_SAME: &str = "SELECT value = $4::text::numeric FROM observation
     WHERE indicator_id = $1 AND cycle = $2 AND period_start = $3
     FOR UPDATE";

/// 새 행은 넣고 있던 행은 덮어쓴다(설계서 5.2).
/// 값이 달라졌으면 트리거(observation_keep_history)가 이전 값을 observation_history에 남긴다.
const UPSERT: &str = "INSERT INTO observation
         (indicator_id, cycle, period_start, period_raw, value, value_raw, source_changed_at, collected_at, run_id)
     VALUES ($1, $2, $3, $4, $5::text::numeric, $6, $7, $8, $9)
     ON CONFLICT (indicator_id, cycle, period_start) DO UPDATE SET
         period_raw = EXCLUDED.period_raw,
         value = EXCLUDED.value,
         value_raw = EXCLUDED.value_raw,
         source_changed_at = EXCLUDED.source_changed_at,
         collected_at = EXCLUDED.collected_at,
         run_id = EXCLUDED.run_id";

async fn upsert_observations(
    tx: &deadpool_postgres::Transaction<'_>,
    ind: &Indicator,
    raw: &RawResponse<'_>,
    rows: &[Observation],
) -> Result<Counts, DbError> {
    let mut counts = Counts::default();
    if rows.is_empty() {
        return Ok(counts);
    }
    let select_same = tx.prepare(SELECT_SAME).await.map_err(db_error)?;
    let upsert = tx.prepare(UPSERT).await.map_err(db_error)?;
    // 이전 주기 값을 남기려고 관측값에 주기를 함께 쓴다(KEDP001-D022 11).
    let cycle = ind.cycle.code();
    for o in rows {
        let same: Option<bool> = tx
            .query_opt(&select_same, &[&ind.id, &cycle, &o.period_start, &o.value])
            .await
            .map_err(db_error)?
            .map(|row| row.get(0));
        tx.execute(
            &upsert,
            &[
                &ind.id,
                &cycle,
                &o.period_start,
                &o.period_raw,
                &o.value,
                &o.value_raw,
                &o.source_changed_at,
                // 수집 시각은 Go가 응답을 받은 시각이다.
                &raw.received_at,
                &raw.run_id,
            ],
        )
        .await
        .map_err(db_error)?;
        match same {
            None => counts.inserted += 1,
            Some(false) => counts.updated += 1,
            Some(true) => counts.unchanged += 1,
        }
    }
    Ok(counts)
}

/// tokio_postgres 오류를 DbError로 바꾼다.
fn db_error(e: tokio_postgres::Error) -> DbError {
    match e.as_db_error() {
        // DB가 돌려준 오류. 외래 키 위반은 run_id가 collection_run에 없는 경우다
        // (indicator_id는 먼저 읽어서 확인했다).
        Some(db) if *db.code() == SqlState::FOREIGN_KEY_VIOLATION => {
            DbError::NotFound(db.detail().unwrap_or(db.message()).to_string())
        }
        Some(db) => DbError::Internal(db.message().to_string()),
        // DB 오류가 아니면 연결 문제다(끊김, 시간 초과 등).
        None => DbError::Unavailable(describe(&e)),
    }
}

/// 연결 오류는 원인(예: Connection refused)까지 붙인다.
fn describe(e: &tokio_postgres::Error) -> String {
    match std::error::Error::source(e) {
        Some(cause) => format!("{e}: {cause}"),
        None => e.to_string(),
    }
}
