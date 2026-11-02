# ECOS 샘플

요청 주소의 키 자리는 `***`로 가렸다. 받은 날: 2026-10-25

| 파일 | 내용 | 요청 |
| --- | --- | --- |
| `722Y001-0101000-M-202401-202609.json` | 한국은행 기준금리, 월, 정상 응답 33행 | `StatisticSearch/***/json/kr/1/1000/722Y001/M/202401/202609/0101000` |
| `error-INFO-100.json` | 인증키가 유효하지 않음 | 틀린 키로 요청 |
| `error-INFO-200.json` | 해당하는 데이터가 없음 | `StatisticSearch/***/json/kr/1/10/722Y001/M/190001/190003/0101000` |

정상 응답은 `b-go-end`에서 받았다.

```sh
cd b-go-end
go run ./cmd/server -indicator ecos-base-rate -start 202401 -end 202609
```

## KEDP001-D024 추가 지표 (Issue #55)

받은 날: 2026-11-02. 지표마다 정상 응답 하나다. 요청 주소의 키 자리는 `***`로 가렸다.
같은 통계 코드의 샘플이 여러 개다. `cmd/fakesource`는 통계 코드로 고른 파일 중 이름순 첫 파일을 돌려주므로, 이 표들은 fakesource 시험에서 다른 항목의 응답이 나갈 수 있다.

| 파일 | 지표 | 행 | 요청 |
| --- | --- | --- | --- |
| `200Y105-1400-Q-2024Q1-2026Q2.json` | `ecos-gdp-nominal` 명목 GDP(원계열) | 10 | `StatisticSearch/***/json/kr/1/1000/200Y105/Q/2024Q1/2026Q2/1400` |
| `200Y106-1400-Q-2024Q1-2026Q3.json` | `ecos-gdp-real` 실질 GDP(원계열) | 11 | `StatisticSearch/***/json/kr/1/1000/200Y106/Q/2024Q1/2026Q3/1400` |
| `200Y104-1400-Q-2024Q1-2026Q3.json` | `ecos-gdp-real-sa` 실질 GDP(계절조정) | 11 | `StatisticSearch/***/json/kr/1/1000/200Y104/Q/2024Q1/2026Q3/1400` |
| `200Y102-10111-Q-2024Q1-2026Q3.json` | `ecos-gdp-growth-qoq` 실질 GDP 성장률(계절조정, 전기 대비) | 11 | `StatisticSearch/***/json/kr/1/1000/200Y102/Q/2024Q1/2026Q3/10111` |
| `200Y102-10211-Q-2024Q1-2026Q3.json` | `ecos-gdp-growth-yoy` 실질 GDP 성장률(원계열, 전년 동기 대비) | 11 | `StatisticSearch/***/json/kr/1/1000/200Y102/Q/2024Q1/2026Q3/10211` |
| `721Y001-5020000-M-202401-202610.json` | `ecos-ktb-3y` 국고채 3년 금리 | 34 | `StatisticSearch/***/json/kr/1/1000/721Y001/M/202401/202610/5020000` |
| `721Y001-5050000-M-202401-202610.json` | `ecos-ktb-10y` 국고채 10년 금리 | 34 | `StatisticSearch/***/json/kr/1/1000/721Y001/M/202401/202610/5050000` |
| `721Y001-7020000-M-202401-202610.json` | `ecos-corp-bond-3y-aa` 회사채 3년 AA- 금리 | 34 | `StatisticSearch/***/json/kr/1/1000/721Y001/M/202401/202610/7020000` |
| `121Y006-BECBLA01-M-202401-202609.json` | `ecos-loan-rate-new` 예금은행 대출금리(신규취급액) | 33 | `StatisticSearch/***/json/kr/1/1000/121Y006/M/202401/202609/BECBLA01` |
| `121Y002-BEABAA2-M-202401-202609.json` | `ecos-deposit-rate-new` 예금은행 수신금리(신규취급액) | 33 | `StatisticSearch/***/json/kr/1/1000/121Y002/M/202401/202609/BEABAA2` |
| `161Y006-BBHA00-M-202401-202609.json` | `ecos-m2-avg` M2(평잔, 원계열) | 32 | `StatisticSearch/***/json/kr/1/1000/161Y006/M/202401/202609/BBHA00` |
| `151Y001-1000000-Q-2024Q1-2026Q2.json` | `ecos-household-credit` 가계신용(잔액) | 10 | `StatisticSearch/***/json/kr/1/1000/151Y001/Q/2024Q1/2026Q2/1000000` |
| `151Y005-11100A0-M-202401-202608.json` | `ecos-housing-loan` 주택관련대출(예금취급기관, 잔액) | 32 | `StatisticSearch/***/json/kr/1/1000/151Y005/M/202401/202608/11100A0` |
| `731Y004-0000001-0000100-M-202401-202610.json` | `ecos-usd-krw` 원·달러 환율(월평균) | 34 | `StatisticSearch/***/json/kr/1/1000/731Y004/M/202401/202610/0000001/0000100` |
| `301Y013-000000-M-202401-202608.json` | `ecos-current-account` 경상수지(원계열) | 32 | `StatisticSearch/***/json/kr/1/1000/301Y013/M/202401/202608/000000` |

받은 명령(조사 샘플은 `samples/repository/d024/`):

```sh
cd b-go-end
go run ./cmd/sample -out ../samples/repository/d024 -list ../samples/repository/d024/round2.yaml   # round3.yaml, round4.yaml도 같은 방법
```
