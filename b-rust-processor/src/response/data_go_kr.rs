//! 공공데이터포털 getETFPriceInfo_V2 응답 (samples/data-go-kr/)
//!
//! - 정상: `{"response":{"header":{"resultCode":"00"},"body":{"totalCount":34,"items":{"item":[{"basDt","srtnCd","clpr"…}]}}}}`
//! - 데이터 없음: 정상 코드 `00`과 `totalCount` 0
//! - 게이트웨이 오류: `{"OpenAPI_ServiceResponse":{"cmmMsgHeader":{"returnReasonCode":"30","errMsg","returnAuthMsg"}}}` (HTTP 403 등)

use serde_json::Value;

use super::{Answer, Indicator, Row, expect, read_rows, required, text};

const OK_CODE: &str = "00";

pub(super) fn read(ind: &Indicator, json: &Value) -> Result<Answer, String> {
    if let Some(service_error) = json.get("OpenAPI_ServiceResponse") {
        let header = service_error
            .get("cmmMsgHeader")
            .ok_or("OpenAPI_ServiceResponse has no cmmMsgHeader")?;
        let code = required(header, "returnReasonCode")?;
        // b-go-end data_go_kr.Parse와 같게 errMsg와 returnAuthMsg를 잇는다.
        let message = format!(
            "{} {}",
            text(header, "errMsg")?.unwrap_or_default(),
            text(header, "returnAuthMsg")?.unwrap_or_default()
        );
        return Ok(Answer::Error {
            code,
            message: message.trim().to_string(),
        });
    }

    let response = json
        .get("response")
        .ok_or("response has neither response nor OpenAPI_ServiceResponse")?;
    let header = response.get("header").ok_or("response has no header")?;
    let code = required(header, "resultCode")?;
    if code != OK_CODE {
        let message = text(header, "resultMsg")?.unwrap_or_default();
        return Ok(Answer::Error { code, message });
    }
    let body = response.get("body").ok_or("response has no body")?;
    let total = body
        .get("totalCount")
        .and_then(Value::as_u64)
        .ok_or("totalCount is not a number")?;
    // 데이터가 없으면 이 API는 정상 코드(00)와 totalCount 0을 준다.
    if total == 0 {
        return Ok(Answer::NoData {
            code,
            message: "totalCount 0".to_string(),
        });
    }

    let rows = body
        .get("items")
        .and_then(|items| items.get("item"))
        .and_then(Value::as_array)
        .ok_or("items.item is not an array")?;
    // 한 행에 값이 여러 개다. 지표 값은 indicator.value_field 칸(clpr)만 쓴다(KEDP001-D022 8).
    let value_field = ind
        .value_field
        .as_deref()
        .ok_or("indicator has no value_field")?;
    read_rows(rows, |row| {
        expect(row, "srtnCd", &ind.stat_code)?;
        Ok(Row {
            period: required(row, "basDt")?,
            value: text(row, value_field)?,
            source_changed_at: None,
        })
    })
}
