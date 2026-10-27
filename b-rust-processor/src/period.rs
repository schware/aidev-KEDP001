//! 기간 변환 (설계서 5.1, KEDP001-D022 2·3).
//! 응답의 기간 원문(예: 202401, 2025Q1)을 기간 시작일(date)로 바꾼다.
//!
//! | 주기 | ECOS       | KOSIS                       | 공공데이터포털 | 기간 시작일 |
//! | ---- | ---------- | --------------------------- | -------------- | ----------- |
//! | D    | `20261001` | 없음                        | `20261022`     | 그 날       |
//! | M    | `202401`   | `202401`                    | 없음           | 그 달 1일   |
//! | Q    | `2025Q1`   | `202401` (끝 두 자리가 분기) | 없음           | 분기 첫날   |
//! | A    | `2023`     | `2015`                      | 없음           | 1월 1일     |

use chrono::NaiveDate;

/// 제공처. indicator.source_code 값과 같다.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Source {
    Ecos,
    Kosis,
    DataGoKr,
}

impl Source {
    pub fn from_code(code: &str) -> Option<Source> {
        match code {
            "ecos" => Some(Source::Ecos),
            "kosis" => Some(Source::Kosis),
            "data-go-kr" => Some(Source::DataGoKr),
            _ => None,
        }
    }

    pub fn code(self) -> &'static str {
        match self {
            Source::Ecos => "ecos",
            Source::Kosis => "kosis",
            Source::DataGoKr => "data-go-kr",
        }
    }
}

/// 주기. indicator.cycle, observation.cycle 값과 같다. 연은 A다(KEDP001-D022 3).
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Cycle {
    D,
    M,
    Q,
    A,
}

impl Cycle {
    pub fn from_code(code: &str) -> Option<Cycle> {
        match code {
            "D" => Some(Cycle::D),
            "M" => Some(Cycle::M),
            "Q" => Some(Cycle::Q),
            "A" => Some(Cycle::A),
            _ => None,
        }
    }

    pub fn code(self) -> &'static str {
        match self {
            Cycle::D => "D",
            Cycle::M => "M",
            Cycle::Q => "Q",
            Cycle::A => "A",
        }
    }
}

/// 기간 원문을 기간 시작일로 바꾼다. 형식이 맞지 않으면 이유를 돌려준다.
pub fn period_start(source: Source, cycle: Cycle, raw: &str) -> Result<NaiveDate, String> {
    // 아래에서 글자 위치로 자르므로 ASCII가 아니면 먼저 거른다.
    if !raw.is_ascii() {
        return Err(format!("period {raw:?} is not ASCII"));
    }
    match (source, cycle) {
        (Source::Ecos | Source::DataGoKr, Cycle::D) => {
            let (y, m, d) = (
                digits(raw, 8, 0..4),
                digits(raw, 8, 4..6),
                digits(raw, 8, 6..8),
            );
            date(raw, y, m, d, "YYYYMMDD")
        }
        (Source::Ecos | Source::Kosis, Cycle::M) => date(
            raw,
            digits(raw, 6, 0..4),
            digits(raw, 6, 4..6),
            Some(1),
            "YYYYMM",
        ),
        // ECOS 분기: 2025Q1
        (Source::Ecos, Cycle::Q) => {
            let q = if raw.len() == 6 && &raw[4..5] == "Q" {
                raw[5..].parse::<u32>().ok().filter(|q| (1..=4).contains(q))
            } else {
                None
            };
            date(
                raw,
                digits(raw, 6, 0..4),
                q.map(first_month),
                Some(1),
                "YYYYQn",
            )
        }
        // KOSIS 분기: 202401 (끝 두 자리 01~04가 분기). 월과 모양이 같아 주기로 구분한다.
        (Source::Kosis, Cycle::Q) => {
            let q = digits(raw, 6, 4..6).filter(|q| (1..=4).contains(q));
            date(
                raw,
                digits(raw, 6, 0..4),
                q.map(first_month),
                Some(1),
                "YYYY0n",
            )
        }
        (Source::Ecos | Source::Kosis, Cycle::A) => {
            date(raw, digits(raw, 4, 0..4), Some(1), Some(1), "YYYY")
        }
        _ => Err(format!(
            "{} has no {} cycle (설계서 5.1)",
            source.code(),
            cycle.code()
        )),
    }
}

/// 분기의 첫 달. 1분기 → 1월, 2분기 → 4월 …
fn first_month(quarter: u32) -> u32 {
    (quarter - 1) * 3 + 1
}

/// raw의 길이가 len이고 range 부분이 모두 숫자이면 그 숫자
fn digits(raw: &str, len: usize, range: std::ops::Range<usize>) -> Option<u32> {
    if raw.len() != len {
        return None;
    }
    let part = raw.get(range)?;
    if part.bytes().all(|b| b.is_ascii_digit()) {
        part.parse().ok()
    } else {
        None
    }
}

fn date(
    raw: &str,
    year: Option<u32>,
    month: Option<u32>,
    day: Option<u32>,
    form: &str,
) -> Result<NaiveDate, String> {
    year.zip(month)
        .zip(day)
        .and_then(|((y, m), d)| NaiveDate::from_ymd_opt(y as i32, m, d))
        .ok_or_else(|| format!("period {raw:?} is not {form}"))
}

#[cfg(test)]
mod tests {
    use super::*;

    fn ymd(y: i32, m: u32, d: u32) -> NaiveDate {
        NaiveDate::from_ymd_opt(y, m, d).unwrap()
    }

    // 설계서 5.1 표의 예
    #[test]
    fn converts_design_examples() {
        use Cycle::*;
        use Source::*;
        let cases = [
            (Ecos, D, "20261001", ymd(2026, 10, 1)),
            (DataGoKr, D, "20261022", ymd(2026, 10, 22)),
            (Ecos, M, "202401", ymd(2024, 1, 1)),
            (Kosis, M, "202401", ymd(2024, 1, 1)),
            (Ecos, Q, "2025Q1", ymd(2025, 1, 1)),
            (Ecos, Q, "2025Q2", ymd(2025, 4, 1)),
            (Ecos, Q, "2025Q3", ymd(2025, 7, 1)),
            (Ecos, Q, "2025Q4", ymd(2025, 10, 1)),
            (Kosis, Q, "202401", ymd(2024, 1, 1)),
            (Kosis, Q, "202402", ymd(2024, 4, 1)),
            (Kosis, Q, "202404", ymd(2024, 10, 1)),
            (Ecos, A, "2023", ymd(2023, 1, 1)),
            (Kosis, A, "2015", ymd(2015, 1, 1)),
        ];
        for (source, cycle, raw, want) in cases {
            assert_eq!(
                period_start(source, cycle, raw),
                Ok(want),
                "{source:?} {cycle:?} {raw}"
            );
        }
    }

    #[test]
    fn rejects_wrong_forms() {
        use Cycle::*;
        use Source::*;
        let cases = [
            (Ecos, D, "2026100"),  // 7자리
            (Ecos, D, "20260230"), // 없는 날
            (Ecos, M, "202413"),   // 13월
            (Ecos, M, "2024-01"),
            (Ecos, Q, "2025Q5"),
            (Ecos, Q, "202501"),  // KOSIS 분기 형식
            (Kosis, Q, "2025Q1"), // ECOS 분기 형식
            (Kosis, Q, "202405"),
            (Kosis, A, "15"),
            (Ecos, A, "２０２３"),  // 전각 숫자
            (Kosis, D, "20261001"), // KOSIS 일 주기는 없다
            (DataGoKr, M, "202401"),
        ];
        for (source, cycle, raw) in cases {
            assert!(
                period_start(source, cycle, raw).is_err(),
                "{source:?} {cycle:?} {raw}"
            );
        }
    }
}
