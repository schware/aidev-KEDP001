//! KOSIS statisticsParameterData 응답 (samples/kosis/)
//!
//! - 정상: 최상위가 행 배열 `[{"ORG_ID","TBL_ID","ITM_ID","C1","PRD_SE","PRD_DE","DT","LST_CHN_DE"…}]`
//! - 오류: `{"err":"11","errMsg":"…"}` (HTTP 200). `30`은 데이터 없음이다.

use serde_json::Value;

use super::{Answer, Indicator, Row, expect, read_rows, required, text};

const NO_DATA_CODE: &str = "30";

pub(super) fn read(ind: &Indicator, json: &Value) -> Result<Answer, String> {
    if json.is_object() {
        let code = required(json, "err").map_err(|_| "response is neither rows nor err")?;
        let message = text(json, "errMsg")?.unwrap_or_default();
        return Ok(if code == NO_DATA_CODE {
            Answer::NoData { code, message }
        } else {
            Answer::Error { code, message }
        });
    }
    let rows = json.as_array().ok_or("response is neither rows nor err")?;
    read_rows(rows, |row| read_row(ind, row))
}

fn read_row(ind: &Indicator, row: &Value) -> Result<Row, String> {
    // 지표 정의와 같은 계열인지 확인한다. 분류 C1, C2…는 요청의 objL1, objL2…와 같은 순서다.
    expect(row, "ORG_ID", ind.org_id.as_deref().unwrap_or_default())?;
    expect(row, "TBL_ID", &ind.stat_code)?;
    expect(row, "ITM_ID", ind.item_code.as_deref().unwrap_or_default())?;
    for (i, class) in ind.classes.iter().enumerate() {
        expect(row, &format!("C{}", i + 1), class)?;
    }
    // 월과 분기는 기간 원문이 둘 다 6자리라 주기를 함께 본다(설계서 5.1).
    // 요청의 Y는 응답에서 A로 오며 indicator.cycle도 A다(KEDP001-D022 3).
    expect(row, "PRD_SE", ind.cycle.code())?;
    Ok(Row {
        period: required(row, "PRD_DE")?,
        value: text(row, "DT")?,
        // 변경 일자는 없어도 된다.
        source_changed_at: row
            .get("LST_CHN_DE")
            .and_then(Value::as_str)
            .map(str::to_string),
    })
}
