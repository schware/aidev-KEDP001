//! 제공처 응답 읽기. b-proto/README.md의 "결과 ProcessStatus" 표를 따른다.
//!
//! 1. 본문이 JSON이면 제공처마다 읽는다(ecos, kosis, data_go_kr 모듈).
//! 2. 제공처 오류 코드가 있으면 HTTP 상태와 상관없이 그 코드로 NO_DATA 또는 ERROR다.
//! 3. 그 밖에 HTTP 상태가 200이 아니면 ERROR(`HTTP_<상태 코드>`)다.
//! 4. 행마다 기간을 바꾸고 값을 확인한다. 한 행이라도 읽을 수 없으면 ERROR(`INVALID_RESPONSE`)다.
//!
//! 이 모듈은 DB를 쓰지 않는다. 읽은 결과(Reading)를 db 모듈이 저장한다.

mod data_go_kr;
mod ecos;
mod kosis;

use std::collections::HashSet;

use chrono::NaiveDate;
use serde_json::Value;

use crate::period::{self, Cycle, Source};

/// 응답을 읽을 수 없을 때의 Rust 오류 코드(b-proto/README.md)
pub const INVALID_RESPONSE: &str = "INVALID_RESPONSE";

/// 응답을 읽는 데 필요한 지표 정의(indicator 테이블의 한 행)
#[derive(Clone, Debug)]
pub struct Indicator {
    pub id: String,
    pub source: Source,
    /// KOSIS 기관 코드(orgId)
    pub org_id: Option<String>,
    /// ECOS 통계 코드, KOSIS tblId, 공공데이터포털 종목 코드
    pub stat_code: String,
    /// ECOS 항목 코드, KOSIS itmId. 공공데이터포털은 없다
    pub item_code: Option<String>,
    /// 분류 코드. ECOS ITEM_CODE2…, KOSIS C1…
    pub classes: Vec<String>,
    /// 공공데이터포털에서 지표 값으로 쓰는 칸(clpr)
    pub value_field: Option<String>,
    pub cycle: Cycle,
}

/// 응답 하나를 읽은 결과
#[derive(Debug, PartialEq)]
pub enum Reading {
    /// 정상 응답. 관측값으로 쓸 행과, 값이 비어 건너뛴 행 수
    Success {
        rows: Vec<Observation>,
        skipped_empty: i32,
    },
    /// 제공처가 데이터 없음으로 답했다
    NoData { code: String, message: String },
    /// 제공처 오류이거나 응답을 읽을 수 없다
    Error { code: String, message: String },
}

/// observation 테이블에 쓸 한 행
#[derive(Debug, PartialEq)]
pub struct Observation {
    /// 기간 시작일(설계서 5.1)
    pub period_start: NaiveDate,
    /// 응답의 기간 원문(예: 202401)
    pub period_raw: String,
    /// 숫자로 확인한 값(앞뒤 공백을 뺀 응답 문자열). DB에서 numeric으로 바꾼다
    pub value: String,
    /// 응답의 값 원문(KEDP001-D022 10)
    pub value_raw: String,
    /// KOSIS LST_CHN_DE. 다른 제공처는 None
    pub source_changed_at: Option<NaiveDate>,
}

/// 제공처 모듈이 응답을 읽은 결과. Rows의 행은 아직 기간과 값을 바꾸지 않은 원문이다.
enum Answer {
    Rows(Vec<Row>),
    NoData { code: String, message: String },
    Error { code: String, message: String },
}

/// 응답의 한 행. 제공처마다 다른 칸 이름(TIME, PRD_DE, basDt …)을 여기서 맞춘다.
struct Row {
    period: String,
    /// None이면 응답의 값이 null이다
    value: Option<String>,
    source_changed_at: Option<String>,
}

/// 응답 하나를 읽는다.
pub fn read(ind: &Indicator, http_status: i32, body: &[u8]) -> Reading {
    // 1. JSON이면 제공처마다 읽는다. Err는 응답 모양이 예상과 다르다는 뜻이다.
    let answer = match serde_json::from_slice::<Value>(body) {
        Ok(json) => match ind.source {
            Source::Ecos => ecos::read(ind, &json),
            Source::Kosis => kosis::read(ind, &json),
            Source::DataGoKr => data_go_kr::read(ind, &json),
        },
        Err(_) => Err("body is not JSON".to_string()),
    };

    let rows = match answer {
        // 2. 제공처 오류 코드가 있으면 그 코드를 쓴다.
        Ok(Answer::NoData { code, message }) => return Reading::NoData { code, message },
        Ok(Answer::Error { code, message }) => return Reading::Error { code, message },
        // 3. 제공처 오류 코드가 없고 HTTP 상태가 200이 아니다(예: 게이트웨이 오류 문서).
        _ if http_status != 200 => {
            return Reading::Error {
                code: format!("HTTP_{http_status}"),
                message: format!("HTTP status {http_status}"),
            };
        }
        Ok(Answer::Rows(rows)) => rows,
        Err(message) => return invalid(message),
    };

    // 4. 행마다 기간과 값을 바꾼다.
    convert(ind, rows).unwrap_or_else(invalid)
}

fn invalid(message: String) -> Reading {
    Reading::Error {
        code: INVALID_RESPONSE.to_string(),
        message,
    }
}

/// 응답 행을 observation 행으로 바꾼다. 한 행이라도 바꿀 수 없으면 Err다.
fn convert(ind: &Indicator, rows: Vec<Row>) -> Result<Reading, String> {
    let mut out = Vec::with_capacity(rows.len());
    let mut skipped_empty = 0;
    let mut seen = HashSet::new();
    for (i, row) in rows.into_iter().enumerate() {
        let at = |message: String| format!("row {}: {message}", i + 1);

        let period_start = period::period_start(ind.source, ind.cycle, &row.period).map_err(at)?;
        // 한 응답에 같은 기간이 두 번 오면 어느 값을 쓸지 알 수 없다.
        if !seen.insert(period_start) {
            return Err(at(format!("period {:?} appears twice", row.period)));
        }

        // 빈 값(결측)은 관측값으로 쓰지 않는다(KEDP001-D022 15).
        // 결측 표기는 샘플로 확인하지 못했다(설계서 5.5). null과 빈 문자열만 빈 값으로 보고,
        // 그 밖의 표기(예: "-")는 숫자가 아니므로 INVALID_RESPONSE가 된다.
        let Some(value_raw) = row.value.filter(|v| !v.trim().is_empty()) else {
            skipped_empty += 1;
            continue;
        };
        let value = numeric(&value_raw)
            .ok_or_else(|| at(format!("value {value_raw:?} is not a number")))?
            .to_string();

        let source_changed_at = match row.source_changed_at.filter(|s| !s.is_empty()) {
            Some(s) => Some(
                NaiveDate::parse_from_str(&s, "%Y-%m-%d")
                    .map_err(|_| at(format!("LST_CHN_DE {s:?} is not YYYY-MM-DD")))?,
            ),
            None => None,
        };

        out.push(Observation {
            period_start,
            period_raw: row.period,
            value,
            value_raw,
            source_changed_at,
        });
    }
    Ok(Reading::Success {
        rows: out,
        skipped_empty,
    })
}

/// 값이 10진수 숫자이면 앞뒤 공백을 뺀 문자열을 돌려준다.
/// 응답에서 본 형식: "3.5", "61", "86525", ".25", "-.39"(설계서 1.2).
/// 지수(1e5), 쉼표(1,000), NaN처럼 본 적 없는 형식은 받지 않는다.
fn numeric(raw: &str) -> Option<&str> {
    let s = raw.trim();
    let unsigned = s.strip_prefix(['+', '-']).unwrap_or(s);
    let (int, frac) = unsigned.split_once('.').unwrap_or((unsigned, ""));
    let all_digits = |part: &str| part.bytes().all(|b| b.is_ascii_digit());
    let ok = all_digits(int) && all_digits(frac) && !(int.is_empty() && frac.is_empty());
    ok.then_some(s)
}

// 아래는 제공처 모듈이 함께 쓰는 칸 읽기 함수다.

/// 문자열 칸을 읽는다. null이면 None이다. 칸이 없거나 문자열이 아니면 Err다.
fn text(obj: &Value, key: &str) -> Result<Option<String>, String> {
    match obj.get(key) {
        Some(Value::String(s)) => Ok(Some(s.clone())),
        Some(Value::Null) => Ok(None),
        Some(_) => Err(format!("{key} is not a string")),
        None => Err(format!("{key} is missing")),
    }
}

/// 꼭 값이 있어야 하는 문자열 칸을 읽는다.
fn required(obj: &Value, key: &str) -> Result<String, String> {
    text(obj, key)?.ok_or_else(|| format!("{key} is null"))
}

/// 칸 값이 지표 정의와 같은지 확인한다. 다른 지표의 응답이 잘못 넘어온 것을 막는다.
fn expect(obj: &Value, key: &str, want: &str) -> Result<(), String> {
    let got = required(obj, key)?;
    if got == want {
        Ok(())
    } else {
        Err(format!("{key} is {got:?}, indicator has {want:?}"))
    }
}

/// 행 배열을 하나씩 읽는다. 오류 메시지에 몇 번째 행인지 붙인다.
fn read_rows(
    rows: &[Value],
    read_row: impl Fn(&Value) -> Result<Row, String>,
) -> Result<Answer, String> {
    let mut out = Vec::with_capacity(rows.len());
    for (i, row) in rows.iter().enumerate() {
        out.push(read_row(row).map_err(|m| format!("row {}: {m}", i + 1))?);
    }
    Ok(Answer::Rows(out))
}

#[cfg(test)]
mod tests {
    use super::*;

    /// 저장소 samples/ 아래 파일을 읽는다.
    fn sample(path: &str) -> Vec<u8> {
        let full = format!("{}/../samples/{path}", env!("CARGO_MANIFEST_DIR"));
        std::fs::read(&full).unwrap_or_else(|e| panic!("{full}: {e}"))
    }

    fn ymd(y: i32, m: u32, d: u32) -> NaiveDate {
        NaiveDate::from_ymd_opt(y, m, d).unwrap()
    }

    fn indicator(
        source: Source,
        stat_code: &str,
        item_code: Option<&str>,
        classes: &[&str],
        cycle: Cycle,
    ) -> Indicator {
        Indicator {
            id: "test".into(),
            source,
            org_id: (source == Source::Kosis).then(|| "101".into()),
            stat_code: stat_code.into(),
            item_code: item_code.map(Into::into),
            classes: classes.iter().map(|c| c.to_string()).collect(),
            value_field: (source == Source::DataGoKr).then(|| "clpr".into()),
            cycle,
        }
    }

    fn ecos_base_rate(cycle: Cycle) -> Indicator {
        indicator(Source::Ecos, "722Y001", Some("0101000"), &[], cycle)
    }

    fn kosis_employment(cycle: Cycle) -> Indicator {
        indicator(Source::Kosis, "DT_1DA7001S", Some("T90"), &["0"], cycle)
    }

    fn data_go_kr(code: &str) -> Indicator {
        indicator(Source::DataGoKr, code, None, &[], Cycle::D)
    }

    /// 정상 응답이면 (행, 빈 값 수), 아니면 실패
    fn success(reading: Reading) -> (Vec<Observation>, i32) {
        match reading {
            Reading::Success {
                rows,
                skipped_empty,
            } => (rows, skipped_empty),
            other => panic!("not success: {other:?}"),
        }
    }

    /// 행 수와 첫 행·마지막 행의 (기간 시작일, 기간 원문, 값)을 확인한다.
    fn check_rows(
        rows: &[Observation],
        len: usize,
        first: (NaiveDate, &str, &str),
        last: (NaiveDate, &str, &str),
    ) {
        assert_eq!(rows.len(), len);
        for (row, (start, raw, value)) in [(&rows[0], first), (&rows[len - 1], last)] {
            assert_eq!(
                (
                    row.period_start,
                    row.period_raw.as_str(),
                    row.value.as_str()
                ),
                (start, raw, value)
            );
        }
    }

    #[test]
    fn ecos_month() {
        let (rows, skipped) = success(read(
            &ecos_base_rate(Cycle::M),
            200,
            &sample("ecos/722Y001-0101000-M-202401-202609.json"),
        ));
        check_rows(
            &rows,
            33,
            (ymd(2024, 1, 1), "202401", "3.5"),
            (ymd(2026, 9, 1), "202609", "3"),
        );
        assert_eq!(skipped, 0);
        assert_eq!(rows[0].source_changed_at, None);
    }

    #[test]
    fn kosis_month() {
        let (rows, _) = success(read(
            &kosis_employment(Cycle::M),
            200,
            &sample("kosis/DT_1DA7001S-T90-0-M-202401-202609.json"),
        ));
        check_rows(
            &rows,
            33,
            (ymd(2024, 1, 1), "202401", "61"),
            (ymd(2026, 9, 1), "202609", "63.5"),
        );
        assert_eq!(rows[0].source_changed_at, Some(ymd(2024, 2, 1)));
    }

    #[test]
    fn kosis_year() {
        // 요청은 prdSe=Y, 응답 PRD_SE와 indicator.cycle은 A다(KEDP001-D022 3).
        let births = indicator(
            Source::Kosis,
            "INH_1B8000F_01",
            Some("T1"),
            &["11"],
            Cycle::A,
        );
        let (rows, _) = success(read(
            &births,
            200,
            &sample("kosis/INH_1B8000F_01-T1-11-Y-2015-2025.json"),
        ));
        check_rows(
            &rows,
            11,
            (ymd(2015, 1, 1), "2015", "438420"),
            (ymd(2025, 1, 1), "2025", "254341"),
        );

        let population = indicator(Source::Kosis, "DT_1IN1502", Some("T100"), &["00"], Cycle::A);
        let (rows, _) = success(read(
            &population,
            200,
            &sample("kosis/DT_1IN1502-T100-00-Y-2015-2025.json"),
        ));
        check_rows(
            &rows,
            11,
            (ymd(2015, 1, 1), "2015", "51069375"),
            (ymd(2025, 1, 1), "2025", "51817499"),
        );
        assert_eq!(rows[10].source_changed_at, Some(ymd(2026, 7, 24)));
    }

    #[test]
    fn data_go_kr_day() {
        // 최근 것부터 온다(설계서 1.2). 지표 값은 clpr다(KEDP001-D022 8).
        let cases = [
            ("069500", "86525", "107885"),
            ("122630", "66770", "107725"),
            ("114800", "1243", "1021"),
        ];
        for (code, first, last) in cases {
            let body = sample(&format!("data-go-kr/{code}-D-20260901-20261022.json"));
            let (rows, _) = success(read(&data_go_kr(code), 200, &body));
            check_rows(
                &rows,
                34,
                (ymd(2026, 10, 22), "20261022", first),
                (ymd(2026, 9, 1), "20260901", last),
            );
        }
    }

    #[test]
    fn provider_errors_and_no_data() {
        let error = |code: &str| Reading::Error {
            code: code.into(),
            message: String::new(),
        };
        let no_data = |code: &str| Reading::NoData {
            code: code.into(),
            message: String::new(),
        };
        let cases = [
            (
                ecos_base_rate(Cycle::M),
                200,
                "ecos/error-INFO-100.json",
                error("INFO-100"),
            ),
            (
                ecos_base_rate(Cycle::M),
                200,
                "ecos/error-INFO-200.json",
                no_data("INFO-200"),
            ),
            (
                kosis_employment(Cycle::M),
                200,
                "kosis/error-11.json",
                error("11"),
            ),
            (
                kosis_employment(Cycle::M),
                200,
                "kosis/error-30.json",
                no_data("30"),
            ),
            // 인증키 오류는 HTTP 403이지만 본문의 제공처 코드를 쓴다.
            (
                data_go_kr("069500"),
                403,
                "data-go-kr/error-30.json",
                error("30"),
            ),
            (
                data_go_kr("069500"),
                200,
                "data-go-kr/nodata-totalCount-0.json",
                no_data("00"),
            ),
        ];
        for (ind, status, file, want) in cases {
            // 메시지는 제공처 문구라 코드와 결과만 비교한다.
            let got = match read(&ind, status, &sample(file)) {
                Reading::Error { code, .. } => error(&code),
                Reading::NoData { code, .. } => no_data(&code),
                other => panic!("{file}: {other:?}"),
            };
            assert_eq!(got, want, "{file}");
        }
    }

    fn error_code(reading: Reading) -> String {
        match reading {
            Reading::Error { code, .. } => code,
            other => panic!("not error: {other:?}"),
        }
    }

    #[test]
    fn http_error_without_provider_code() {
        let html = b"<html><body>502 Bad Gateway</body></html>";
        assert_eq!(
            error_code(read(&ecos_base_rate(Cycle::M), 502, html)),
            "HTTP_502"
        );
        // 정상 모양의 응답이어도 HTTP 상태가 200이 아니면 오류다.
        let body = sample("ecos/722Y001-0101000-M-202401-202609.json");
        assert_eq!(
            error_code(read(&ecos_base_rate(Cycle::M), 500, &body)),
            "HTTP_500"
        );
        // HTTP 200인데 JSON이 아니면 읽을 수 없는 응답이다.
        assert_eq!(
            error_code(read(&ecos_base_rate(Cycle::M), 200, html)),
            INVALID_RESPONSE
        );
    }

    #[test]
    fn rejects_response_of_other_indicator() {
        let body = sample("ecos/722Y001-0101000-M-202401-202609.json");
        let other_item = indicator(Source::Ecos, "722Y001", Some("0102000"), &[], Cycle::M);
        assert_eq!(error_code(read(&other_item, 200, &body)), INVALID_RESPONSE);
        // KOSIS 응답 주기(PRD_SE M)가 지표 주기(Q)와 다르다.
        let body = sample("kosis/DT_1DA7001S-T90-0-M-202401-202609.json");
        assert_eq!(
            error_code(read(&kosis_employment(Cycle::Q), 200, &body)),
            INVALID_RESPONSE
        );
        // 종목 코드가 다르다.
        let body = sample("data-go-kr/069500-D-20260901-20261022.json");
        assert_eq!(
            error_code(read(&data_go_kr("122630"), 200, &body)),
            INVALID_RESPONSE
        );
    }

    /// ECOS 샘플의 첫 행(202401, 3.5) 값을 바꾼 본문
    fn ecos_with_first_value(value: &str) -> Vec<u8> {
        let body = String::from_utf8(sample("ecos/722Y001-0101000-M-202401-202609.json")).unwrap();
        let from = r#""TIME":"202401","DATA_VALUE":"3.5""#;
        assert!(body.contains(from));
        body.replacen(from, &format!(r#""TIME":"202401","DATA_VALUE":{value}"#), 1)
            .into_bytes()
    }

    #[test]
    fn skips_empty_values() {
        for value in ["null", r#""""#, r#"" ""#] {
            let (rows, skipped) = success(read(
                &ecos_base_rate(Cycle::M),
                200,
                &ecos_with_first_value(value),
            ));
            assert_eq!((rows.len(), skipped), (32, 1), "{value}");
            assert_eq!(rows[0].period_raw, "202402");
        }
    }

    #[test]
    fn rejects_values_that_are_not_numbers() {
        for value in [r#""-""#, r#""1,000""#, r#""1e5""#, r#""NaN""#, "3.5"] {
            let reading = read(
                &ecos_base_rate(Cycle::M),
                200,
                &ecos_with_first_value(value),
            );
            assert_eq!(error_code(reading), INVALID_RESPONSE, "{value}");
        }
    }

    #[test]
    fn keeps_value_raw() {
        let (rows, _) = success(read(
            &ecos_base_rate(Cycle::M),
            200,
            &ecos_with_first_value(r#"" 3.50""#),
        ));
        assert_eq!(
            (rows[0].value.as_str(), rows[0].value_raw.as_str()),
            ("3.50", " 3.50")
        );
    }

    #[test]
    fn rejects_duplicate_period() {
        let body = String::from_utf8(sample("ecos/722Y001-0101000-M-202401-202609.json")).unwrap();
        let body = body.replacen(r#""TIME":"202402""#, r#""TIME":"202401""#, 1);
        assert_eq!(
            error_code(read(&ecos_base_rate(Cycle::M), 200, body.as_bytes())),
            INVALID_RESPONSE
        );
    }

    #[test]
    fn numeric_forms() {
        for ok in [
            "3.5", "3", "61.6", "86525", ".25", "-.39", "+1", "3.", " 3.5 ",
        ] {
            assert!(numeric(ok).is_some(), "{ok}");
        }
        for bad in [
            "", ".", "-", "+", "1,000", "1e5", "NaN", "Infinity", "3.5.1", "--1", "１",
        ] {
            assert!(numeric(bad).is_none(), "{bad}");
        }
    }

    // 설계서 5.1의 나머지 기간 형식(ECOS D·Q·A, KOSIS Q)
    #[test]
    fn ecos_day_quarter_year() {
        let (rows, _) = success(read(
            &ecos_base_rate(Cycle::D),
            200,
            &sample("ecos/722Y001-0101000-D-20261001-20261023.json"),
        ));
        check_rows(
            &rows,
            23,
            (ymd(2026, 10, 1), "20261001", "3"),
            (ymd(2026, 10, 23), "20261023", "3"),
        );
        let (rows, _) = success(read(
            &ecos_base_rate(Cycle::Q),
            200,
            &sample("ecos/722Y001-0101000-Q-2025Q1-2025Q3.json"),
        ));
        check_rows(
            &rows,
            3,
            (ymd(2025, 1, 1), "2025Q1", "2.75"),
            (ymd(2025, 7, 1), "2025Q3", "2.5"),
        );
        let (rows, _) = success(read(
            &ecos_base_rate(Cycle::A),
            200,
            &sample("ecos/722Y001-0101000-A-2023-2025.json"),
        ));
        check_rows(
            &rows,
            3,
            (ymd(2023, 1, 1), "2023", "3.5"),
            (ymd(2025, 1, 1), "2025", "2.5"),
        );
    }

    #[test]
    fn kosis_quarter() {
        let body = sample("kosis/DT_1DA7001S-T90-0-Q-202401-202402.json");
        let (rows, _) = success(read(&kosis_employment(Cycle::Q), 200, &body));
        check_rows(
            &rows,
            2,
            (ymd(2024, 1, 1), "202401", "61.6"),
            (ymd(2024, 4, 1), "202402", "63.3"),
        );
    }
}
