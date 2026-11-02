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

## KEDP001-D024 추가 지표 (Issue #55)

받은 날: 2026-11-02. 지표마다 정상 응답 하나다. 요청 주소의 키 자리는 `***`로 가렸다.
같은 통계 코드의 샘플이 여러 개다. `cmd/fakesource`는 통계 코드로 고른 파일 중 이름순 첫 파일을 돌려주므로, 이 표들은 fakesource 시험에서 다른 항목의 응답이 나갈 수 있다.

| 파일 | 지표 | 행 | 요청 |
| --- | --- | --- | --- |
| `DT_1JH20202-T1-1-M-202401-202609.json` | `kosis-industrial-production-sa` 전산업생산지수(계절조정) | 33 | `apiKey=***&endPrdDe=202609&format=json&itmId=T1&jsonVD=Y&method=getList&objL1=1&orgId=101&prdSe=M&startPrdDe=202401&tblId=DT_1JH20202` |
| `DT_1F02001-T20-00-C-M-202401-202609.json` | `kosis-manufacturing-production-sa` 제조업생산지수(계절조정) | 33 | `apiKey=***&endPrdDe=202609&format=json&itmId=T20&jsonVD=Y&method=getList&objL1=00&objL2=C&orgId=101&prdSe=M&startPrdDe=202401&tblId=DT_1F02001` |
| `DT_1KC2020-T3-T-M-202401-202609.json` | `kosis-service-production-sa` 서비스업생산지수(계절조정) | 33 | `apiKey=***&endPrdDe=202609&format=json&itmId=T3&jsonVD=Y&method=getList&objL1=T&orgId=101&prdSe=M&startPrdDe=202401&tblId=DT_1KC2020` |
| `DT_1K41012-T3-G0-M-202401-202609.json` | `kosis-retail-sales-sa` 소매판매액지수(계절조정) | 33 | `apiKey=***&endPrdDe=202609&format=json&itmId=T3&jsonVD=Y&method=getList&objL1=G0&orgId=101&prdSe=M&startPrdDe=202401&tblId=DT_1K41012` |
| `DT_1F70011-T5-ALL-C-M-202401-202609.json` | `kosis-facility-investment-sa` 설비투자지수(계절조정) (pending) | 33 | `apiKey=***&endPrdDe=202609&format=json&itmId=T5&jsonVD=Y&method=getList&objL1=ALL&objL2=C&orgId=101&prdSe=M&startPrdDe=202401&tblId=DT_1F70011` |
| `DT_1J22042-T03-0-M-202401-202609.json` | `kosis-cpi-yoy` 소비자물가지수 전년 동월 대비 | 33 | `apiKey=***&endPrdDe=202609&format=json&itmId=T03&jsonVD=Y&method=getList&objL1=0&orgId=101&prdSe=M&startPrdDe=202401&tblId=DT_1J22042` |
| `DT_1J22042-T03-1-M-202401-202609.json` | `kosis-living-cpi-yoy` 생활물가지수 전년 동월 대비 | 33 | `apiKey=***&endPrdDe=202609&format=json&itmId=T03&jsonVD=Y&method=getList&objL1=1&orgId=101&prdSe=M&startPrdDe=202401&tblId=DT_1J22042` |
| `DT_1J22042-T03-2-M-202401-202609.json` | `kosis-fresh-food-cpi-yoy` 신선식품지수 전년 동월 대비 | 33 | `apiKey=***&endPrdDe=202609&format=json&itmId=T03&jsonVD=Y&method=getList&objL1=2&orgId=101&prdSe=M&startPrdDe=202401&tblId=DT_1J22042` |
| `DT_1J22042-T03-4-M-202401-202609.json` | `kosis-core-cpi-yoy` 근원물가(식료품·에너지 제외) 전년 동월 대비 | 33 | `apiKey=***&endPrdDe=202609&format=json&itmId=T03&jsonVD=Y&method=getList&objL1=4&orgId=101&prdSe=M&startPrdDe=202401&tblId=DT_1J22042` |
| `DT_1DA7001S-T30-0-M-202401-202609.json` | `kosis-employed` 취업자 수 | 33 | `apiKey=***&endPrdDe=202609&format=json&itmId=T30&jsonVD=Y&method=getList&objL1=0&orgId=101&prdSe=M&startPrdDe=202401&tblId=DT_1DA7001S` |
| `DT_1DA7001S-T80-0-M-202401-202609.json` | `kosis-unemployment-rate` 실업률 | 33 | `apiKey=***&endPrdDe=202609&format=json&itmId=T80&jsonVD=Y&method=getList&objL1=0&orgId=101&prdSe=M&startPrdDe=202401&tblId=DT_1DA7001S` |
| `DT_1DA7001S-T60-0-M-202401-202609.json` | `kosis-participation-rate` 경제활동참가율 | 33 | `apiKey=***&endPrdDe=202609&format=json&itmId=T60&jsonVD=Y&method=getList&objL1=0&orgId=101&prdSe=M&startPrdDe=202401&tblId=DT_1DA7001S` |
| `DT_1DA7002S-T90-75-M-202401-202609.json` | `kosis-youth-employment-rate` 청년 고용률(15~29세) | 33 | `apiKey=***&endPrdDe=202609&format=json&itmId=T90&jsonVD=Y&method=getList&objL1=75&orgId=101&prdSe=M&startPrdDe=202401&tblId=DT_1DA7002S` |
| `DT_1DA7002S-T10-63-M-202401-202609.json` | `kosis-working-age-population` 생산가능인구(15~64세) | 33 | `apiKey=***&endPrdDe=202609&format=json&itmId=T10&jsonVD=Y&method=getList&objL1=63&orgId=101&prdSe=M&startPrdDe=202401&tblId=DT_1DA7002S` |
| `DT_1BPA002-T10-1-33-Y-2015-2025.json` | `kosis-elderly-ratio` 65세 이상 인구 비율(장래인구추계) | 11 | `apiKey=***&endPrdDe=2025&format=json&itmId=T10&jsonVD=Y&method=getList&objL1=1&objL2=33&orgId=101&prdSe=Y&startPrdDe=2015&tblId=DT_1BPA002` |

- `DT_1F70011`(설비투자지수)은 분류 B가 둘째 자리라 `objL1=ALL&objL2=C`로 불러야 한다. 응답 행에 `C1`이 없어 지금 수집기(Rust)가 읽지 못한다.
- `DT_1J22042`(소비자물가 등락률)와 `DT_1BPA002`(주요 인구지표)는 `UNIT_NM`이 없다.

받은 명령(조사 샘플은 `samples/repository/d024/`):

```sh
cd b-go-end
go run ./cmd/sample -out ../samples/repository/d024 -list ../samples/repository/d024/round2.yaml   # round3.yaml, round4.yaml도 같은 방법
```
