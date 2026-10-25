# 공공데이터포털 샘플 (금융위원회_증권상품시세정보)

요청 주소의 키 자리는 `***`로 가렸다. 받은 날: 2026-10-25

모든 요청의 앞부분은 같다: `GetSecuritiesProductInfoService_V2/getETFPriceInfo_V2?serviceKey=***&resultType=json&numOfRows=1000&pageNo=1`

| 파일 | 내용 | 요청 (앞부분 뒤) |
| --- | --- | --- |
| `069500-D-20260901-20261022.json` | KODEX 200, 일, 정상 응답 34행 | `likeSrtnCd=069500&beginBasDt=20260901&endBasDt=20261023` |
| `122630-D-20260901-20261022.json` | KODEX 레버리지, 일, 정상 응답 34행 | `likeSrtnCd=122630&beginBasDt=20260901&endBasDt=20261023` |
| `114800-D-20260901-20261022.json` | KODEX 인버스, 일, 정상 응답 34행 | `likeSrtnCd=114800&beginBasDt=20260901&endBasDt=20261023` |
| `error-30.json` | 등록되지 않은 서비스키 (HTTP 403) | 틀린 키로 요청 |
| `nodata-totalCount-0.json` | 데이터 없음: 정상 코드 `00`, `totalCount` 0, `item` 빈 배열 (HTTP 200) | `numOfRows=5&likeSrtnCd=069500&beginBasDt=19000101&endBasDt=19000105` |

- 정상 응답은 `response.header`(`resultCode` `00`)와 `response.body.items.item` 배열이다. 행은 `basDt` 내림차순(최근 날짜가 먼저)으로 온다.
- 행이 하나여도 `item`은 배열로 온다(`numOfRows=1`로 확인).
- 데이터가 없어도 오류 코드가 아니라 `00`과 `totalCount` 0이 온다. 수집 모듈은 다른 제공처와 맞추려고 이것을 "데이터 없음" 오류로 다룬다.
- 게이트웨이 오류는 `OpenAPI_ServiceResponse.cmmMsgHeader`(`returnReasonCode`, `errMsg`, `returnAuthMsg`)로 온다. 인증키 오류는 HTTP 403, 서비스 이름이 틀리면 HTTP 400(`12` NO_OPENAPI_SERVICE_ERROR), `numOfRows=abc`는 HTTP 200(`04` HTTP_ERROR)이었다.
- `endBasDt`는 그 날을 포함하지 않는다(`beginBasDt=20261020&endBasDt=20261022`는 20일·21일 2행). 수집 모듈은 요청의 끝 날짜에 하루를 더해 보낸다.
- `likeSrtnCd`는 부분 일치 검색이지만 세 종목 모두 그 종목만 왔다.
- 날짜 형식이 틀리면(`beginBasDt=2026-09-01`) 오류 없이 조건이 무시된다.
- 2026-10-25(일) 16시에 받았을 때 마지막 `basDt`는 20261022(목)였다. 20261023(금)은 아직 없었다.

정상 응답은 `b-go-end`에서 받았다. `.env`의 `DATA_GO_KR_API_KEY`는 Decoding 키와 Encoding 키 모두 쓸 수 있다.

```sh
cd b-go-end
go run ./cmd/server -indicator data-go-kr-kodex-200 -start 20260901 -end 20261022
go run ./cmd/server -indicator data-go-kr-kodex-leverage -start 20260901 -end 20261022
go run ./cmd/server -indicator data-go-kr-kodex-inverse -start 20260901 -end 20261022
```
