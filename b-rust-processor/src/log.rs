//! 로그. KEDP 로그 형식(KEDP001-D026)으로 stderr에 한 줄씩 남긴다. 새 의존성 없이 쓴다.
//!
//! ```text
//! time=2026-11-24T06:00:03.456+09:00 level=INFO comp=rust event=process.page run=812 indicator=cpi page=1 status=SUCCESS inserted=0 updated=1
//! ```
//!
//! - 모든 줄에 time(Asia/Seoul, 밀리초), level(DEBUG, INFO, WARN, ERROR), comp=rust, event가 있다.
//! - 값이 비었거나 공백, `"`, `=`, 제어 문자가 있으면 큰따옴표로 감싼다(b-go-end의 log/slog TextHandler와 같다).
//! - KEDP_LOG_LEVEL 이상만 남긴다(기본 INFO).
//!
//! 응답 본문, 요청 주소, DB 비밀번호는 넘기지 않는다(D026 7장).
//!
//! ```ignore
//! log_info!("process.page", "run" => req.run_id, "indicator" => &req.indicator_id);
//! ```

use std::fmt::Write as _;
use std::io::Write as _;
use std::sync::atomic::{AtomicU8, Ordering};

use chrono::{DateTime, FixedOffset, Utc};

/// 이 서버의 comp 값이다.
pub const COMP: &str = "rust";

/// 로그 level을 정하는 변수다(b-go-end와 같다).
pub const LEVEL_VAR: &str = "KEDP_LOG_LEVEL";

/// Asia/Seoul. 일광 절약 시간이 없으므로 +09:00 고정이다.
const SEOUL_OFFSET_SECS: i32 = 9 * 3600;

#[derive(Clone, Copy, Debug, PartialEq, Eq, PartialOrd, Ord)]
#[repr(u8)]
pub enum Level {
    /// 개발 중 흐름을 따라가기 위한 줄. 운영 기본값(INFO)에서는 남지 않는다.
    Debug = 0,
    /// 정상 동작.
    Info = 1,
    /// 하나가 실패했거나 이상하지만 서버는 계속 정상이다.
    Warn = 2,
    /// 서버 쪽 문제로 기능이 실패했다. 같은 실패는 결과를 정하는 곳에서 한 번만 남긴다.
    Error = 3,
}

impl Level {
    pub fn as_str(self) -> &'static str {
        match self {
            Level::Debug => "DEBUG",
            Level::Info => "INFO",
            Level::Warn => "WARN",
            Level::Error => "ERROR",
        }
    }

    fn from_u8(v: u8) -> Level {
        match v {
            0 => Level::Debug,
            1 => Level::Info,
            2 => Level::Warn,
            _ => Level::Error,
        }
    }
}

static LEVEL: AtomicU8 = AtomicU8::new(Level::Info as u8);

/// KEDP_LOG_LEVEL 값을 읽는다. 비어 있으면 INFO다. DEBUG, INFO, WARN, ERROR가 아니면 None이다(D026 3장).
pub fn parse_level(s: &str) -> Option<Level> {
    match s.trim().to_ascii_uppercase().as_str() {
        "" | "INFO" => Some(Level::Info),
        "DEBUG" => Some(Level::Debug),
        "WARN" => Some(Level::Warn),
        "ERROR" => Some(Level::Error),
        _ => None,
    }
}

/// lv 이상만 남기게 한다. 서버를 시작할 때 한 번 부른다.
pub fn set_level(lv: Level) {
    LEVEL.store(lv as u8, Ordering::Relaxed);
}

/// 지금 남기는 가장 낮은 level이다.
pub fn level() -> Level {
    Level::from_u8(LEVEL.load(Ordering::Relaxed))
}

/// lv를 남기는지 알려 준다.
pub fn enabled(lv: Level) -> bool {
    lv >= level()
}

/// 한 줄을 만든다. fields는 (key, value) 쌍이다. 줄 끝에 줄바꿈은 없다.
pub fn format_line(
    time: DateTime<Utc>,
    lv: Level,
    event: &str,
    fields: &[(&str, String)],
) -> String {
    let offset = FixedOffset::east_opt(SEOUL_OFFSET_SECS).expect("valid offset");
    let mut line = format!(
        "time={} level={} comp={COMP} event={}",
        time.with_timezone(&offset)
            .format("%Y-%m-%dT%H:%M:%S%.3f%:z"),
        lv.as_str(),
        quote(event)
    );
    for (k, v) in fields {
        let _ = write!(line, " {k}={}", quote(v));
    }
    line
}

/// event 한 줄을 stderr에 남긴다. 보통은 log_debug!, log_info!, log_warn!, log_error!를 쓴다.
pub fn emit(lv: Level, event: &str, fields: &[(&str, String)]) {
    if !enabled(lv) {
        return;
    }
    let line = format_line(Utc::now(), lv, event, fields);
    // 한 번에 써서 여러 task의 줄이 섞이지 않게 한다. 쓰지 못해도(stderr가 닫힘) 처리는 계속한다.
    let _ = writeln!(std::io::stderr().lock(), "{line}");
}

/// 값에 따옴표가 필요하면 감싼다. Rust Debug 꼴로 감싸므로 줄바꿈·따옴표는 \n, \"로 바뀐다.
fn quote(v: &str) -> String {
    let needs = v.is_empty()
        || v.chars()
            .any(|c| c.is_whitespace() || c == '"' || c == '=' || c.is_control());
    if needs {
        format!("{v:?}")
    } else {
        v.to_string()
    }
}

/// `log_at!(level, "event", "key" => value, ...)`. value는 Display로 바꾼다.
/// 그 level을 남기지 않으면 값을 만들지 않는다.
#[macro_export]
macro_rules! log_at {
    ($lv:expr, $event:expr $(, $k:literal => $v:expr)* $(,)?) => {
        if $crate::log::enabled($lv) {
            $crate::log::emit($lv, $event, &[$(($k, ($v).to_string())),*]);
        }
    };
}

#[macro_export]
macro_rules! log_debug {
    ($($t:tt)*) => { $crate::log_at!($crate::log::Level::Debug, $($t)*) };
}

#[macro_export]
macro_rules! log_info {
    ($($t:tt)*) => { $crate::log_at!($crate::log::Level::Info, $($t)*) };
}

#[macro_export]
macro_rules! log_warn {
    ($($t:tt)*) => { $crate::log_at!($crate::log::Level::Warn, $($t)*) };
}

#[macro_export]
macro_rules! log_error {
    ($($t:tt)*) => { $crate::log_at!($crate::log::Level::Error, $($t)*) };
}

#[cfg(test)]
mod tests {
    use super::*;

    fn at(s: &str) -> DateTime<Utc> {
        DateTime::parse_from_rfc3339(s).unwrap().with_timezone(&Utc)
    }

    #[test]
    fn line_format() {
        let line = format_line(
            at("2026-11-23T21:00:03.456789Z"),
            Level::Info,
            "process.page",
            &[
                ("run", 812.to_string()),
                ("indicator", "cpi".into()),
                ("status", "SUCCESS".into()),
            ],
        );
        assert_eq!(
            line,
            "time=2026-11-24T06:00:03.456+09:00 level=INFO comp=rust event=process.page run=812 indicator=cpi status=SUCCESS"
        );
    }

    #[test]
    fn quotes_when_needed() {
        let line = format_line(
            at("2026-11-24T00:00:00Z"),
            Level::Warn,
            "source_unit.changed",
            &[
                ("old", String::new()),
                ("new", "십억원 (2020=100)".into()),
                ("err", "a\"b\nlevel=ERROR".into()),
                ("plain", "한글값".into()),
            ],
        );
        assert!(
            line.ends_with(
                r#"old="" new="십억원 (2020=100)" err="a\"b\nlevel=ERROR" plain=한글값"#
            ),
            "{line}"
        );
        // 값에 줄바꿈이 있어도 한 줄이다.
        assert!(!line.contains('\n'), "{line}");
    }

    #[test]
    fn parse_levels() {
        for (s, want) in [
            ("", Level::Info),
            ("DEBUG", Level::Debug),
            ("INFO", Level::Info),
            ("WARN", Level::Warn),
            ("ERROR", Level::Error),
            (" debug ", Level::Debug),
            ("warn", Level::Warn),
        ] {
            assert_eq!(parse_level(s), Some(want), "{s:?}");
        }
        for s in ["WARNING", "TRACE", "FATAL", "1"] {
            assert_eq!(parse_level(s), None, "{s:?}");
        }
    }

    #[test]
    fn level_filter() {
        // LEVEL은 process 전체에서 하나이므로 이 시험만 바꾸고 되돌린다.
        let before = level();
        set_level(Level::Warn);
        assert!(!enabled(Level::Debug));
        assert!(!enabled(Level::Info));
        assert!(enabled(Level::Warn));
        assert!(enabled(Level::Error));
        set_level(Level::Debug);
        assert!(enabled(Level::Debug));
        set_level(before);
    }
}
