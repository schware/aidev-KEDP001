# KOSIS 샘플

요청 주소의 키 자리는 `***`로 가렸다. 받은 날: 2026-10-25

모든 요청의 앞부분은 같다: `statisticsParameterData.do?method=getList&apiKey=***&format=json&jsonVD=Y&orgId=101`

| 파일 | 내용 | 요청 (앞부분 뒤) |
| --- | --- | --- |
| `DT_1DA7001S-T90-0-M-202401-202609.json` | 고용률(15세 이상, 성별 계), 월, 정상 응답 33행 | `tblId=DT_1DA7001S&itmId=T90&objL1=0&prdSe=M&startPrdDe=202401&endPrdDe=202609` |
| `INH_1B8000F_01-T1-11-Y-2015-2025.json` | 출생아 수, 연, 정상 응답 11행 | `tblId=INH_1B8000F_01&itmId=T1&objL1=11&prdSe=Y&startPrdDe=2015&endPrdDe=2025` |
| `DT_1IN1502-T100-00-Y-2015-2025.json` | 총인구(전국), 연, 정상 응답 11행 | `tblId=DT_1IN1502&itmId=T100&objL1=00&prdSe=Y&startPrdDe=2015&endPrdDe=2025` |
| `error-11.json` | 유효하지 않은 인증KEY | 틀린 키로 요청 |
| `error-30.json` | 데이터가 존재하지 않습니다 | `tblId=DT_1IN1502&itmId=T100&objL1=00&prdSe=Y&startPrdDe=1900&endPrdDe=1901` |

- 정상 응답은 행 배열, 오류 응답은 `{"err","errMsg"}` 객체다. 둘 다 HTTP 200으로 온다.
- 요청의 `prdSe=Y`는 응답 `PRD_SE`에서 `A`로 온다.
- 출생아 수 통계표는 `UNIT_NM`이 없다. 단위는 분류 이름(`C1_NM`: `출생아수(명)`)에 들어 있다.

정상 응답은 `b-go-end`에서 받았다.

```sh
cd b-go-end
go run ./cmd/server -indicator kosis-employment-rate -start 202401 -end 202609
go run ./cmd/server -indicator kosis-births -start 2015 -end 2025
go run ./cmd/server -indicator kosis-total-population -start 2015 -end 2025
```
