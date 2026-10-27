//! ECOS StatisticSearch 응답 (samples/ecos/)
//!
//! - 정상: `{"StatisticSearch":{"list_total_count":33,"row":[{"STAT_CODE","ITEM_CODE1"…,"TIME","DATA_VALUE"}]}}`
//! - 오류: `{"RESULT":{"CODE":"INFO-100","MESSAGE":"…"}}`. `INFO-200`은 데이터 없음이다.

use serde_json::Value;

use super::{Answer, Indicator, Row, expect, read_rows, required, text};

const NO_DATA_CODE: &str = "INFO-200";

pub(super) fn read(ind: &Indicator, json: &Value) -> Result<Answer, String> {
    if let Some(result) = json.get("RESULT") {
        let code = required(result, "CODE")?;
        let message = text(result, "MESSAGE")?.unwrap_or_default();
        return Ok(if code == NO_DATA_CODE {
            Answer::NoData { code, message }
        } else {
            Answer::Error { code, message }
        });
    }
    let rows = json
        .get("StatisticSearch")
        .and_then(|s| s.get("row"))
        .and_then(Value::as_array)
        .ok_or("response has neither StatisticSearch.row nor RESULT")?;
    read_rows(rows, |row| read_row(ind, row))
}

fn read_row(ind: &Indicator, row: &Value) -> Result<Row, String> {
    // 지표 정의와 같은 계열인지 확인한다. 분류는 ITEM_CODE2부터다(b-go-end ecos 요청 주소와 같은 순서).
    expect(row, "STAT_CODE", &ind.stat_code)?;
    expect(
        row,
        "ITEM_CODE1",
        ind.item_code.as_deref().unwrap_or_default(),
    )?;
    for (i, class) in ind.classes.iter().enumerate() {
        expect(row, &format!("ITEM_CODE{}", i + 2), class)?;
    }
    // ECOS 응답에는 주기가 없다. 기간은 지표 주기(indicator.cycle)로 바꾼다.
    Ok(Row {
        period: required(row, "TIME")?,
        value: text(row, "DATA_VALUE")?,
        source_changed_at: None,
    })
}
