# 경제지표 명세 및 API 조사

- 상태: draft

## 상태

- `pending`: 조사 전 또는 조사 중
- `verified`: 실제 API 응답과 주요 key 확인 완료
- `excluded`: 현재 범위에서 제외

통계표 코드와 주기는 2026-10-23 ECOS 웹에서 확인한 값이다. 항목 코드와 응답 key는 API 조사로 확인한다.

## 우선 조사 대상

### ECOS

#### 한국은행 기준금리

- 상태: verified
- API: StatisticSearch
- 통계표: 한국은행 기준금리 및 여수신금리 (`722Y001`)
- 항목 코드: `0101000` (한국은행 기준금리, 2026-10-25 StatisticItemList로 확인)
- 주기: A, D, M, Q
- 기간 key: `TIME` (월 주기 예: `202401`)
- 값 key: `DATA_VALUE` (문자열, 예: `"3.5"`)
- 단위 key: `UNIT_NAME` (`연%`)
- 논리 key 후보: `STAT_CODE` + `ITEM_CODE1` + 주기 + `TIME`
- 샘플: `samples/ecos/722Y001-0101000-M-202401-202609.json` (월 주기, 33행)
- 미확정 사항: 수집 주기(현재 샘플과 `indicators.yaml`은 M), 논리 key

#### 소비자물가지수

- 상태: pending
- API: StatisticSearch
- 통계표: 소비자물가지수 (`901Y009`)
- 항목 코드:
- 주기: A, M, Q
- 기간 key:
- 값 key:
- 단위 key:
- 논리 key 후보:
- 샘플:
- 미확정 사항: 항목 코드

#### 실질 국내총생산

- 상태: pending
- API: StatisticSearch
- 통계표 후보:
  - 국민계정 주요지표(연간지표) (`200Y101`), 주기 A
  - 국민계정 주요지표 (`200Y102`), 주기 Q
- 항목 코드:
- 기간 key:
- 값 key:
- 단위 key:
- 논리 key 후보:
- 샘플:
- 미확정 사항: 항목 코드, `200Y102`의 정확한 통계표 이름(월간/분기)

#### 외환보유액

- 상태: pending
- API: StatisticSearch
- 통계표: 외환보유액 (`732Y001`)
- 항목 코드:
- 주기: A, M, Q
- 기간 key:
- 값 key:
- 단위 key:
- 논리 key 후보:
- 샘플:
- 미확정 사항: 항목 코드

#### KOSPI

- 상태: pending
- API: StatisticSearch
- 통계표 후보:
  - 주식시장(일) (`802Y001`), 주기 D
  - 주식시장(월,년) (`901Y014`), 주기 A, M
- 항목 코드:
- 기간 key:
- 값 key:
- 단위 key:
- 논리 key 후보:
- 샘플:
- 미확정 사항: 항목 코드

#### KOSPI200

- 상태: pending
- API: StatisticSearch
- 통계표 후보: KOSPI와 같은 통계표 (`802Y001`, `901Y014`)
- 항목 코드:
- 기간 key:
- 값 key:
- 단위 key:
- 논리 key 후보:
- 샘플:
- 미확정 사항: 위 통계표에 KOSPI200 항목이 있는지

### KOSIS

같은 지표가 여러 통계표에 있어 웹 검색으로 찾기 어렵다. KOSIS 검색·목록 API로 후보 통계표를 모은 뒤 소유자가 고른다.

#### 고용률

- 상태: verified
- API: statisticsParameterData (`method=getList`, `format=json&jsonVD=Y`)
- 기관 코드: `101`
- 통계표 ID: `DT_1DA7001S` (성별 경제활동인구 총괄)
- 항목 코드: `T90` (고용률, 2026-10-25 getMeta ITM으로 확인)
- 분류 코드: `objL1=0` (성별: 계)
- 주기: M, Q, Y (월 1999.06~, 분기 1999 3/4~, 연 2000~)
- 기간 key: `PRD_DE` (월 주기 예: `202401`)
- 값 key: `DT` (문자열, 예: `"61"`)
- 단위 key: `UNIT_NM` (`%`)
- 논리 key 후보: `ORG_ID` + `TBL_ID` + `ITM_ID` + `C1` + `PRD_SE` + `PRD_DE`
- 샘플: `samples/kosis/DT_1DA7001S-T90-0-M-202401-202609.json` (월 주기, 33행)
- 미확정 사항: 수집 주기(현재 샘플과 `indicators.yaml`은 M), 논리 key
- 참고: 이 통계표의 고용률은 15세 이상 기준이다(같은 표의 `T10`이 15세이상인구). 15~64세 고용률은 이 표에 없다.
- 분류 후보(`objL1`, 성별): `0` 계, `2` 남자, `3` 여자, `4` 농가, `5` 농가남자, `6` 농가여자, `7` 비농가, `8` 비농가남자, `9` 비농가여자

#### 출생아 수

- 상태: verified
- API: statisticsParameterData (`method=getList`, `format=json&jsonVD=Y`)
- 기관 코드: `101`
- 통계표 ID: `INH_1B8000F_01` (출생아수, 합계출산율, 자연증가 등)
- 항목 코드: `T1` (인구동태건수 및 동태율 추이, 이 표의 유일한 항목)
- 분류 코드: `objL1=11` (기본항목별: 출생아수(명))
- 주기: Y (1970~2025). 이 표에는 월 주기가 없다.
- 기간 key: `PRD_DE` (예: `2015`)
- 값 key: `DT` (문자열, 예: `"438420"`)
- 단위 key: 없음. 단위는 `C1_NM`(`출생아수(명)`)에 들어 있다.
- 논리 key 후보: `ORG_ID` + `TBL_ID` + `ITM_ID` + `C1` + `PRD_SE` + `PRD_DE`
- 샘플: `samples/kosis/INH_1B8000F_01-T1-11-Y-2015-2025.json` (연 주기, 11행)
- 미확정 사항: 월 주기가 필요하면 다른 통계표를 찾아야 한다, 단위를 어디서 가져올지, 논리 key
- 참고: 요청 `prdSe=Y`가 응답 `PRD_SE`에서는 `A`로 온다.
- 분류 후보(`objL1`, 기본항목별): `11` 출생아수(명), `14` 자연증가건수(명), `21` 조출생률(천명당), `24` 자연증가율(천명당), `30` 합계출산율(명), `31` 출생성비(명)

#### 총인구

- 상태: verified
- API: statisticsParameterData (`method=getList`, `format=json&jsonVD=Y`)
- 기관 코드: `101`
- 통계표 ID: `DT_1IN1502` (인구, 가구 및 주택 – 읍면동(연도 끝자리 0, 5), 시군구(그 외 연도))
- 항목 코드: `T100` (총인구)
- 분류 코드: `objL1=00` (행정구역별: 전국)
- 주기: Y (2015~2025)
- 기간 key: `PRD_DE` (예: `2015`)
- 값 key: `DT` (문자열, 예: `"51069375"`)
- 단위 key: `UNIT_NM` (`명`)
- 논리 key 후보: `ORG_ID` + `TBL_ID` + `ITM_ID` + `C1` + `PRD_SE` + `PRD_DE`
- 샘플: `samples/kosis/DT_1IN1502-T100-00-Y-2015-2025.json` (연 주기, 11행)
- 미확정 사항: 통계표 선택(이 표는 인구총조사 계열이다. 주민등록인구·장래인구추계와 비교하지 않았다), 논리 key
- 참고: 요청 `prdSe=Y`가 응답 `PRD_SE`에서는 `A`로 온다.
- 항목 후보: `T100` 총인구, `T110` 남자, `T120` 여자, `T130` 내국인_계, `T140` 외국인_계 등 20개 (인구·가구·주택)
- 분류 후보(`objL1`, 행정구역별): `00` 전국, `03` 동부, `04` 읍부, `05` 면부, 시도·시군구·읍면동 등 4158개

#### 전산업 사업체 수

- 상태: pending
- API:
- 기관 코드:
- 통계표 ID:
- 항목 코드:
- 분류 코드:
- 주기:
- 기간 key:
- 값 key:
- 단위 key:
- 논리 key 후보:
- 샘플:
- 미확정 사항:

### 공공데이터포털 (금융위원회_증권상품시세정보)

- API 기본 주소: `https://apis.data.go.kr/1160100/GetSecuritiesProductInfoService_V2`
- 오퍼레이션: `getETFPriceInfo_V2` (2026-10-25 실제 요청으로 확인. `getETFPriceInfo`는 `12` NO_OPENAPI_SERVICE_ERROR)
- 요청: `serviceKey`, `resultType=json`, `numOfRows`, `pageNo`, `likeSrtnCd`(종목 코드), `beginBasDt`, `endBasDt`(그 날은 포함하지 않는다)
- 주기: 일(거래일)만 있다.
- 응답 항목 이름(세 종목 같다): `basDt` 기준일자, `srtnCd` 종목 코드, `isinCd`, `itmsNm` 종목명, `clpr` 종가, `vs` 전일 대비, `fltRt` 등락률, `nav`, `mkp` 시가, `hipr` 고가, `lopr` 저가, `trqu` 거래량, `trPrc` 거래대금, `mrktTotAmt` 시가총액, `stLstgCnt` 상장좌수, `bssIdxIdxNm` 기초지수명, `bssIdxClpr` 기초지수 종가, `nPptTotAmt` 순자산총액. 값은 모두 문자열이다.
- 데이터 갱신 시점: 2026-10-25(일) 16시에 마지막 `basDt`가 20261022(목)였고 20261023(금)은 없었다. 적어도 하루 이상 늦게 들어온다. 정확한 반영 시각은 확인하지 않았다.
- 데이터 없음: 오류 코드가 아니라 `resultCode` `00`과 `totalCount` 0으로 온다.
- 오류: `OpenAPI_ServiceResponse.cmmMsgHeader.returnReasonCode` (예: `30` 등록되지 않은 서비스키, HTTP 403)
- 주의: 이름이 비슷한 "ESG증권상품" API와 다르다.
- 주의: 레버리지·인버스는 투자 권유처럼 보이지 않도록 화면에 안내 문구가 필요하다.

#### KODEX 200

- 상태: verified
- 종목 코드: `069500` (`isinCd` `KR7069500007`)
- 기초지수(`bssIdxIdxNm`): 코스피 200
- 기간 key: `basDt` (예: `20261022`)
- 값 key: `clpr` 종가 (문자열, 예: `"86525"`)
- 단위 key: 없음 (응답에 단위 항목이 없다)
- 논리 key 후보: `srtnCd` + `basDt`
- 샘플: `samples/data-go-kr/069500-D-20260901-20261022.json` (일, 34행)
- 미확정 사항: 어느 값(`clpr`, `nav` 등)을 지표로 쓸지, 정확한 데이터 반영 시각, 논리 key

#### KODEX 레버리지

- 상태: verified
- 종목 코드: `122630` (`isinCd` `KR7122630007`)
- 기초지수(`bssIdxIdxNm`): 코스피 200
- 기간 key: `basDt` (예: `20261022`)
- 값 key: `clpr` 종가 (문자열, 예: `"66770"`)
- 단위 key: 없음 (응답에 단위 항목이 없다)
- 논리 key 후보: `srtnCd` + `basDt`
- 샘플: `samples/data-go-kr/122630-D-20260901-20261022.json` (일, 34행)
- 미확정 사항: 어느 값(`clpr`, `nav` 등)을 지표로 쓸지, 정확한 데이터 반영 시각, 논리 key

#### KODEX 인버스

- 상태: verified
- 종목 코드: `114800` (`isinCd` `KR7114800006`)
- 기초지수(`bssIdxIdxNm`): 코스피 200 선물지수
- 기간 key: `basDt` (예: `20261022`)
- 값 key: `clpr` 종가 (문자열, 예: `"1243"`)
- 단위 key: 없음 (응답에 단위 항목이 없다)
- 논리 key 후보: `srtnCd` + `basDt`
- 샘플: `samples/data-go-kr/114800-D-20260901-20261022.json` (일, 34행)
- 미확정 사항: 어느 값(`clpr`, `nav` 등)을 지표로 쓸지, 정확한 데이터 반영 시각, 논리 key

## D024 추가 지표 (KEDP001-D024, Issue #55)

D024 "추가 후보" 44개다(Issue #55 본문의 45개는 44개다). verified 29, excluded 4, pending 11.

- 확인 방법: 1차로 후보 통계표의 항목 목록(ECOS StatisticItemList, KOSIS getMeta)을 받아 표 이름·주기·항목 코드를 확인하고, 2~4차로 고른 항목의 데이터를 받았다(2026-11-02, `b-go-end/cmd/sample`). 조사 샘플은 `samples/repository/d024/`(git 밖), 지표별 정상 응답은 `samples/ecos/`, `samples/kosis/`에 있다(파일 목록은 각 폴더의 README).
- 같은 지표가 여러 표에 있으면 원 작성기관의 표를 골랐다(통계청 작성 지표는 KOSIS, 한국은행 작성 지표는 ECOS).
- 주기: 제공처가 D와 M을 둘 다 주면 M. 그 밖에는 가장 짧은 주기 하나(분기·연간이면 분기).
- 증감률은 제공처가 주는 공식값만 넣었다. 내부에서 계산하지 않는다.
- verified 지표는 `b-sql-db/migrations/0005_indicators_d024.up.sql`이 넣는다(노출 켬, 지금 지표 뒤, D024 순서).
- pending: 다른 기관(고용노동부, 한국부동산원, 국토교통부, 금융감독원, 관세청 등)의 KOSIS 표는 샘플 도구가 orgId 101만 불러 아직 확인하지 못했다. 설비투자지수는 응답 형식이 수집기(Rust)와 맞지 않는다.

| # | 구분 | D024 후보 | 상태 | id | 이름 | 제공처 표 / 항목 | 주기 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | 성장·경기 | 명목 GDP | verified | `ecos-gdp-nominal` | 명목 GDP(원계열) | `200Y105` / `1400` | Q |
| 2 | 성장·경기 | 실질 GDP(원계열) | verified | `ecos-gdp-real` | 실질 GDP(원계열) | `200Y106` / `1400` | Q |
| 3 | 성장·경기 | 실질 GDP(계절조정) | verified | `ecos-gdp-real-sa` | 실질 GDP(계절조정) | `200Y104` / `1400` | Q |
| 4 | 성장·경기 | 실질 GDP 전기 대비 성장률 | verified | `ecos-gdp-growth-qoq` | 실질 GDP 성장률(계절조정, 전기 대비) | `200Y102` / `10111` | Q |
| 5 | 성장·경기 | 실질 GDP 전년 동기 대비 성장률 | verified | `ecos-gdp-growth-yoy` | 실질 GDP 성장률(원계열, 전년 동기 대비) | `200Y102` / `10211` | Q |
| 6 | 성장·경기 | 전산업생산지수(계절조정) | verified | `kosis-industrial-production-sa` | 전산업생산지수(계절조정) | `DT_1JH20202` / `T1` / 분류 1 | M |
| 7 | 성장·경기 | 제조업생산지수(계절조정) | verified | `kosis-manufacturing-production-sa` | 제조업생산지수(계절조정) | `DT_1F02001` / `T20` / 분류 00,C | M |
| 8 | 성장·경기 | 서비스업생산지수(계절조정) | verified | `kosis-service-production-sa` | 서비스업생산지수(계절조정) | `DT_1KC2020` / `T3` / 분류 T | M |
| 9 | 성장·경기 | 소매판매액지수(계절조정) | verified | `kosis-retail-sales-sa` | 소매판매액지수(계절조정) | `DT_1K41012` / `T3` / 분류 G0 | M |
| 10 | 성장·경기 | 설비투자지수 | pending | `kosis-facility-investment-sa` | 설비투자지수(계절조정) | `DT_1F70011` / `T5` / 분류 ALL,C | M |
| 11 | 물가 | 소비자물가지수 전년 동월 대비 | verified | `kosis-cpi-yoy` | 소비자물가지수 전년 동월 대비 | `DT_1J22042` / `T03` / 분류 0 | M |
| 12 | 물가 | 생활물가지수 전년 동월 대비 | verified | `kosis-living-cpi-yoy` | 생활물가지수 전년 동월 대비 | `DT_1J22042` / `T03` / 분류 1 | M |
| 13 | 물가 | 신선식품지수 전년 동월 대비 | verified | `kosis-fresh-food-cpi-yoy` | 신선식품지수 전년 동월 대비 | `DT_1J22042` / `T03` / 분류 2 | M |
| 14 | 물가 | 근원물가(식료품·에너지 제외) 전년 동월 대비 | verified | `kosis-core-cpi-yoy` | 근원물가(식료품·에너지 제외) 전년 동월 대비 | `DT_1J22042` / `T03` / 분류 4 | M |
| 15 | 물가 | 생산자물가지수 전년 동월 대비 | excluded | | | | |
| 16 | 물가 | 수입물가지수(원화) 전년 동월 대비 | excluded | | | | |
| 17 | 고용·소득 | 취업자 수 | verified | `kosis-employed` | 취업자 수 | `DT_1DA7001S` / `T30` / 분류 0 | M |
| 18 | 고용·소득 | 실업률 | verified | `kosis-unemployment-rate` | 실업률 | `DT_1DA7001S` / `T80` / 분류 0 | M |
| 19 | 고용·소득 | 경제활동참가율 | verified | `kosis-participation-rate` | 경제활동참가율 | `DT_1DA7001S` / `T60` / 분류 0 | M |
| 20 | 고용·소득 | 청년 고용률 | verified | `kosis-youth-employment-rate` | 청년 고용률(15~29세) | `DT_1DA7002S` / `T90` / 분류 75 | M |
| 21 | 고용·소득 | 명목임금(월) | pending | | | | |
| 22 | 고용·소득 | 실질임금(월) | pending | | | | |
| 23 | 금리·통화 | 국고채 3년 | verified | `ecos-ktb-3y` | 국고채 3년 금리 | `721Y001` / `5020000` | M |
| 24 | 금리·통화 | 국고채 10년 | verified | `ecos-ktb-10y` | 국고채 10년 금리 | `721Y001` / `5050000` | M |
| 25 | 금리·통화 | 회사채 3년 AA- | verified | `ecos-corp-bond-3y-aa` | 회사채 3년 AA- 금리 | `721Y001` / `7020000` | M |
| 26 | 금리·통화 | 예금은행 대출금리(신규취급액) | verified | `ecos-loan-rate-new` | 예금은행 대출금리(신규취급액) | `121Y006` / `BECBLA01` | M |
| 27 | 금리·통화 | 예금은행 수신금리(신규취급액) | verified | `ecos-deposit-rate-new` | 예금은행 수신금리(신규취급액) | `121Y002` / `BEABAA2` | M |
| 28 | 금리·통화 | M2 평잔(원계열) | verified | `ecos-m2-avg` | M2(평잔, 원계열) | `161Y006` / `BBHA00` | M |
| 29 | 부채·금융안정 | 가계신용 | verified | `ecos-household-credit` | 가계신용(잔액) | `151Y001` / `1000000` | Q |
| 30 | 부채·금융안정 | 주택담보대출 | verified | `ecos-housing-loan` | 주택관련대출(예금취급기관, 잔액) | `151Y005` / `11100A0` | M |
| 31 | 부채·금융안정 | 기업신용 | excluded | | | | |
| 32 | 부채·금융안정 | 개인사업자 대출 | excluded | | | | |
| 33 | 부채·금융안정 | 은행 연체율 | pending | | | | |
| 34 | 대외경제 | 원·달러 환율 | verified | `ecos-usd-krw` | 원·달러 환율(월평균) | `731Y004` / `0000001` / 분류 0000100 | M |
| 35 | 대외경제 | 경상수지 | verified | `ecos-current-account` | 경상수지(원계열) | `301Y013` / `000000` | M |
| 36 | 대외경제 | 수출액 | pending | | | | |
| 37 | 대외경제 | 수입액 | pending | | | | |
| 38 | 대외경제 | 무역수지 | pending | | | | |
| 39 | 대외경제 | 일평균 수출액 | pending | | | | |
| 40 | 주택 | 주택매매가격지수 | pending | | | | |
| 41 | 주택 | 주택전세가격지수 | pending | | | | |
| 42 | 주택 | 미분양 주택 | pending | | | | |
| 43 | 인구 | 생산가능인구(15~64세) | verified | `kosis-working-age-population` | 생산가능인구(15~64세) | `DT_1DA7002S` / `T10` / 분류 63 | M |
| 44 | 인구 | 65세 이상 인구 비율 | verified | `kosis-elderly-ratio` | 65세 이상 인구 비율(장래인구추계) | `DT_1BPA002` / `T10` / 분류 1,33 | A |

### 성장·경기

#### 1. 명목 GDP

- 상태: verified
- id: `ecos-gdp-nominal`
- 이름: 명목 GDP(원계열)
- API: ECOS StatisticSearch
- 통계표: 경제활동별 GDP 및 GNI(원계열, 명목, 분기 및 연간) (`200Y105`)
- 항목 코드: `1400` (국내총생산(시장가격, GDP))
- 주기: Q, 차트 기간 20 years
- 기간 key: `TIME`, 값 key: `DATA_VALUE`, 단위 key: `UNIT_NAME`
- 응답 단위: 십억원
- 샘플: `samples/ecos/200Y105-1400-Q-2024Q1-2026Q2.json`
- 고른 이유: 한국은행 국민계정. 분기·연간 중 분기

#### 2. 실질 GDP(원계열)

- 상태: verified
- id: `ecos-gdp-real`
- 이름: 실질 GDP(원계열)
- API: ECOS StatisticSearch
- 통계표: 경제활동별 GDP 및 GNI(원계열, 실질, 분기 및 연간) (`200Y106`)
- 항목 코드: `1400` (국내총생산(시장가격, GDP))
- 주기: Q, 차트 기간 20 years
- 기간 key: `TIME`, 값 key: `DATA_VALUE`, 단위 key: `UNIT_NAME`
- 응답 단위: 십억원
- 샘플: `samples/ecos/200Y106-1400-Q-2024Q1-2026Q3.json`
- 고른 이유: 분기·연간 중 분기

#### 3. 실질 GDP(계절조정)

- 상태: verified
- id: `ecos-gdp-real-sa`
- 이름: 실질 GDP(계절조정)
- API: ECOS StatisticSearch
- 통계표: 경제활동별 GDP 및 GNI(계절조정, 실질, 분기) (`200Y104`)
- 항목 코드: `1400` (국내총생산(시장가격, GDP))
- 주기: Q, 차트 기간 20 years
- 기간 key: `TIME`, 값 key: `DATA_VALUE`, 단위 key: `UNIT_NAME`
- 응답 단위: 십억원
- 샘플: `samples/ecos/200Y104-1400-Q-2024Q1-2026Q3.json`
- 고른 이유: 분기만 있다

#### 4. 실질 GDP 전기 대비 성장률

- 상태: verified
- id: `ecos-gdp-growth-qoq`
- 이름: 실질 GDP 성장률(계절조정, 전기 대비)
- API: ECOS StatisticSearch
- 통계표: 주요지표(분기지표) (`200Y102`)
- 항목 코드: `10111` (국내총생산(GDP)(실질, 계절조정, 전기비))
- 주기: Q, 차트 기간 20 years
- 기간 key: `TIME`, 값 key: `DATA_VALUE`, 단위 key: `UNIT_NAME`
- 응답 단위: %
- 샘플: `samples/ecos/200Y102-10111-Q-2024Q1-2026Q3.json`
- 고른 이유: 한국은행이 주는 공식 성장률

#### 5. 실질 GDP 전년 동기 대비 성장률

- 상태: verified
- id: `ecos-gdp-growth-yoy`
- 이름: 실질 GDP 성장률(원계열, 전년 동기 대비)
- API: ECOS StatisticSearch
- 통계표: 주요지표(분기지표) (`200Y102`)
- 항목 코드: `10211` (국내총생산(GDP)(실질, 원계열, 전년동기비))
- 주기: Q, 차트 기간 20 years
- 기간 key: `TIME`, 값 key: `DATA_VALUE`, 단위 key: `UNIT_NAME`
- 응답 단위: %
- 샘플: `samples/ecos/200Y102-10211-Q-2024Q1-2026Q3.json`
- 고른 이유: 한국은행이 주는 공식 성장률

#### 6. 전산업생산지수(계절조정)

- 상태: verified
- id: `kosis-industrial-production-sa`
- 이름: 전산업생산지수(계절조정)
- API: KOSIS statisticsParameterData
- 통계표: 전산업생산지수(계절조정지수) (`DT_1JH20202`), orgId `101`
- 항목 코드: `T1` (계절조정지수)
- 분류 코드: `1` (KOSIS objL1, objL2…)
- 주기: M, 차트 기간 10 years
- 기간 key: `PRD_DE`, 값 key: `DT`, 단위 key: `UNIT_NM`
- 응답 단위: 2020=100
- 샘플: `samples/kosis/DT_1JH20202-T1-1-M-202401-202609.json`
- 고른 이유: 원 작성기관 통계청(KOSIS). ECOS 901Y033은 재게시

#### 7. 제조업생산지수(계절조정)

- 상태: verified
- id: `kosis-manufacturing-production-sa`
- 이름: 제조업생산지수(계절조정)
- API: KOSIS statisticsParameterData
- 통계표: 시도/산업별 광공업생산지수(2020=100) (`DT_1F02001`), orgId `101`
- 항목 코드: `T20` (생산지수(계절조정))
- 분류 코드: `00`, `C` (KOSIS objL1, objL2…)
- 주기: M, 차트 기간 10 years
- 기간 key: `PRD_DE`, 값 key: `DT`, 단위 key: `UNIT_NM`
- 응답 단위: 2020=100
- 샘플: `samples/kosis/DT_1F02001-T20-00-C-M-202401-202609.json`
- 고른 이유: 통계청 광업제조업동향조사. 분류: 전국(00), 제조업(C). ECOS 901Y032는 재게시

#### 8. 서비스업생산지수(계절조정)

- 상태: verified
- id: `kosis-service-production-sa`
- 이름: 서비스업생산지수(계절조정)
- API: KOSIS statisticsParameterData
- 통계표: 산업별 서비스업생산지수(2020=100.0) (`DT_1KC2020`), orgId `101`
- 항목 코드: `T3` (계절조정지수)
- 분류 코드: `T` (KOSIS objL1, objL2…)
- 주기: M, 차트 기간 10 years
- 기간 key: `PRD_DE`, 값 key: `DT`, 단위 key: `UNIT_NM`
- 응답 단위: 2020=100
- 샘플: `samples/kosis/DT_1KC2020-T3-T-M-202401-202609.json`
- 고른 이유: 통계청. ECOS 901Y038은 재게시

#### 9. 소매판매액지수(계절조정)

- 상태: verified
- id: `kosis-retail-sales-sa`
- 이름: 소매판매액지수(계절조정)
- API: KOSIS statisticsParameterData
- 통계표: 재별 및 상품군별 소매판매액지수(2020=100.0) (`DT_1K41012`), orgId `101`
- 항목 코드: `T3` (계절조정지수)
- 분류 코드: `G0` (KOSIS objL1, objL2…)
- 주기: M, 차트 기간 10 years
- 기간 key: `PRD_DE`, 값 key: `DT`, 단위 key: `UNIT_NM`
- 응답 단위: 2020=100
- 샘플: `samples/kosis/DT_1K41012-T3-G0-M-202401-202609.json`
- 고른 이유: 통계청. 분류 G0 총지수

#### 10. 설비투자지수

- 상태: pending
- 이유: KOSIS 응답이 수집기와 맞지 않아 0005에 넣지 않았다. 분류 B(부문별)가 둘째 자리(objL2)라 objL1=ALL, objL2=C로 불러야 하고, 응답 행에 C1이 없다. Rust는 분류 수만큼 C1, C2…가 있어야 한다
- 확인한 값: `DT_1F70011` 항목 `T5` 분류 `ALL`, `C` 주기 M, 샘플 `samples/kosis/DT_1F70011-T5-ALL-C-M-202401-202609.json`

### 물가

#### 11. 소비자물가지수 전년 동월 대비

- 상태: verified
- id: `kosis-cpi-yoy`
- 이름: 소비자물가지수 전년 동월 대비
- API: KOSIS statisticsParameterData
- 통계표: 월별 소비자물가 등락률 (`DT_1J22042`), orgId `101`
- 항목 코드: `T03` (전년동월비(%))
- 분류 코드: `0` (KOSIS objL1, objL2…)
- 주기: M, 차트 기간 10 years
- 기간 key: `PRD_DE`, 값 key: `DT`, 단위 key: `UNIT_NM`
- 응답 단위: (없음), 관리자 단위(unit): `%`
- 샘플: `samples/kosis/DT_1J22042-T03-0-M-202401-202609.json`
- 고른 이유: 통계청 공식 등락률. 응답에 단위가 없어 관리자 단위 %를 넣는다

#### 12. 생활물가지수 전년 동월 대비

- 상태: verified
- id: `kosis-living-cpi-yoy`
- 이름: 생활물가지수 전년 동월 대비
- API: KOSIS statisticsParameterData
- 통계표: 월별 소비자물가 등락률 (`DT_1J22042`), orgId `101`
- 항목 코드: `T03` (전년동월비(%))
- 분류 코드: `1` (KOSIS objL1, objL2…)
- 주기: M, 차트 기간 10 years
- 기간 key: `PRD_DE`, 값 key: `DT`, 단위 key: `UNIT_NM`
- 응답 단위: (없음), 관리자 단위(unit): `%`
- 샘플: `samples/kosis/DT_1J22042-T03-1-M-202401-202609.json`
- 고른 이유: 통계청 공식 등락률. 관리자 단위 %

#### 13. 신선식품지수 전년 동월 대비

- 상태: verified
- id: `kosis-fresh-food-cpi-yoy`
- 이름: 신선식품지수 전년 동월 대비
- API: KOSIS statisticsParameterData
- 통계표: 월별 소비자물가 등락률 (`DT_1J22042`), orgId `101`
- 항목 코드: `T03` (전년동월비(%))
- 분류 코드: `2` (KOSIS objL1, objL2…)
- 주기: M, 차트 기간 10 years
- 기간 key: `PRD_DE`, 값 key: `DT`, 단위 key: `UNIT_NM`
- 응답 단위: (없음), 관리자 단위(unit): `%`
- 샘플: `samples/kosis/DT_1J22042-T03-2-M-202401-202609.json`
- 고른 이유: 통계청 공식 등락률. 관리자 단위 %

#### 14. 근원물가(식료품·에너지 제외) 전년 동월 대비

- 상태: verified
- id: `kosis-core-cpi-yoy`
- 이름: 근원물가(식료품·에너지 제외) 전년 동월 대비
- API: KOSIS statisticsParameterData
- 통계표: 월별 소비자물가 등락률 (`DT_1J22042`), orgId `101`
- 항목 코드: `T03` (전년동월비(%))
- 분류 코드: `4` (KOSIS objL1, objL2…)
- 주기: M, 차트 기간 10 years
- 기간 key: `PRD_DE`, 값 key: `DT`, 단위 key: `UNIT_NM`
- 응답 단위: (없음), 관리자 단위(unit): `%`
- 샘플: `samples/kosis/DT_1J22042-T03-4-M-202401-202609.json`
- 고른 이유: 통계청 공식 등락률(식료품및에너지제외지수). 관리자 단위 %

#### 15. 생산자물가지수 전년 동월 대비

- 상태: excluded
- 이유: ECOS 404Y014는 지수만 주고 전년 동월 대비 공식값이 없다(내부 계산 금지)

#### 16. 수입물가지수(원화) 전년 동월 대비

- 상태: excluded
- 이유: ECOS 401Y015는 지수(원화·달러·계약통화 기준)만 주고 전년 동월 대비 공식값이 없다(내부 계산 금지)

### 고용·소득

#### 17. 취업자 수

- 상태: verified
- id: `kosis-employed`
- 이름: 취업자 수
- API: KOSIS statisticsParameterData
- 통계표: 성별 경제활동인구 총괄 (`DT_1DA7001S`), orgId `101`
- 항목 코드: `T30` (취업자)
- 분류 코드: `0` (KOSIS objL1, objL2…)
- 주기: M, 차트 기간 10 years
- 기간 key: `PRD_DE`, 값 key: `DT`, 단위 key: `UNIT_NM`
- 응답 단위: 천명
- 샘플: `samples/kosis/DT_1DA7001S-T30-0-M-202401-202609.json`
- 고른 이유: 통계청 경제활동인구조사. 고용률(kosis-employment-rate)과 같은 표

#### 18. 실업률

- 상태: verified
- id: `kosis-unemployment-rate`
- 이름: 실업률
- API: KOSIS statisticsParameterData
- 통계표: 성별 경제활동인구 총괄 (`DT_1DA7001S`), orgId `101`
- 항목 코드: `T80` (실업률)
- 분류 코드: `0` (KOSIS objL1, objL2…)
- 주기: M, 차트 기간 10 years
- 기간 key: `PRD_DE`, 값 key: `DT`, 단위 key: `UNIT_NM`
- 응답 단위: %
- 샘플: `samples/kosis/DT_1DA7001S-T80-0-M-202401-202609.json`
- 고른 이유: 같은 표

#### 19. 경제활동참가율

- 상태: verified
- id: `kosis-participation-rate`
- 이름: 경제활동참가율
- API: KOSIS statisticsParameterData
- 통계표: 성별 경제활동인구 총괄 (`DT_1DA7001S`), orgId `101`
- 항목 코드: `T60` (경제활동참가율)
- 분류 코드: `0` (KOSIS objL1, objL2…)
- 주기: M, 차트 기간 10 years
- 기간 key: `PRD_DE`, 값 key: `DT`, 단위 key: `UNIT_NM`
- 응답 단위: %
- 샘플: `samples/kosis/DT_1DA7001S-T60-0-M-202401-202609.json`
- 고른 이유: 같은 표

#### 20. 청년 고용률

- 상태: verified
- id: `kosis-youth-employment-rate`
- 이름: 청년 고용률(15~29세)
- API: KOSIS statisticsParameterData
- 통계표: 연령별 경제활동인구 총괄 (`DT_1DA7002S`), orgId `101`
- 항목 코드: `T90` (고용률)
- 분류 코드: `75` (KOSIS objL1, objL2…)
- 주기: M, 차트 기간 10 years
- 기간 key: `PRD_DE`, 값 key: `DT`, 단위 key: `UNIT_NM`
- 응답 단위: %
- 샘플: `samples/kosis/DT_1DA7002S-T90-75-M-202401-202609.json`
- 고른 이유: 연령 분류 75(15-29세)

#### 21. 명목임금(월)

- 상태: pending
- 이유: 고용노동부 KOSIS 표로 보인다. -org Issue(샘플 도구 KOSIS orgId) 뒤에 확인한다(Sun&Moon 결정, 2026-11-02)

#### 22. 실질임금(월)

- 상태: pending
- 이유: 고용노동부 KOSIS 표로 보인다. -org Issue(샘플 도구 KOSIS orgId) 뒤에 확인한다(Sun&Moon 결정, 2026-11-02)

### 금리·통화

#### 23. 국고채 3년

- 상태: verified
- id: `ecos-ktb-3y`
- 이름: 국고채 3년 금리
- API: ECOS StatisticSearch
- 통계표: 시장금리(월,분기,년) (`721Y001`)
- 항목 코드: `5020000` (국고채(3년))
- 주기: M, 차트 기간 10 years
- 기간 key: `TIME`, 값 key: `DATA_VALUE`, 단위 key: `UNIT_NAME`
- 응답 단위: 연%
- 샘플: `samples/ecos/721Y001-5020000-M-202401-202610.json`
- 고른 이유: 일(817Y002)과 월이 있어 월

#### 24. 국고채 10년

- 상태: verified
- id: `ecos-ktb-10y`
- 이름: 국고채 10년 금리
- API: ECOS StatisticSearch
- 통계표: 시장금리(월,분기,년) (`721Y001`)
- 항목 코드: `5050000` (국고채(10년))
- 주기: M, 차트 기간 10 years
- 기간 key: `TIME`, 값 key: `DATA_VALUE`, 단위 key: `UNIT_NAME`
- 응답 단위: 연%
- 샘플: `samples/ecos/721Y001-5050000-M-202401-202610.json`
- 고른 이유: 일과 월이 있어 월

#### 25. 회사채 3년 AA-

- 상태: verified
- id: `ecos-corp-bond-3y-aa`
- 이름: 회사채 3년 AA- 금리
- API: ECOS StatisticSearch
- 통계표: 시장금리(월,분기,년) (`721Y001`)
- 항목 코드: `7020000` (회사채(3년, AA-))
- 주기: M, 차트 기간 10 years
- 기간 key: `TIME`, 값 key: `DATA_VALUE`, 단위 key: `UNIT_NAME`
- 응답 단위: 연%
- 샘플: `samples/ecos/721Y001-7020000-M-202401-202610.json`
- 고른 이유: 일과 월이 있어 월. 민평(8010000)이 아닌 기본 항목

#### 26. 예금은행 대출금리(신규취급액)

- 상태: verified
- id: `ecos-loan-rate-new`
- 이름: 예금은행 대출금리(신규취급액)
- API: ECOS StatisticSearch
- 통계표: 예금은행 대출금리(신규취급액 기준) (`121Y006`)
- 항목 코드: `BECBLA01` (대출평균)
- 주기: M, 차트 기간 10 years
- 기간 key: `TIME`, 값 key: `DATA_VALUE`, 단위 key: `UNIT_NAME`
- 응답 단위: 연%
- 샘플: `samples/ecos/121Y006-BECBLA01-M-202401-202609.json`
- 고른 이유: 대출 전체 평균

#### 27. 예금은행 수신금리(신규취급액)

- 상태: verified
- id: `ecos-deposit-rate-new`
- 이름: 예금은행 수신금리(신규취급액)
- API: ECOS StatisticSearch
- 통계표: 예금은행 수신금리(신규취급액 기준) (`121Y002`)
- 항목 코드: `BEABAA2` (저축성수신)
- 주기: M, 차트 기간 10 years
- 기간 key: `TIME`, 값 key: `DATA_VALUE`, 단위 key: `UNIT_NAME`
- 응답 단위: 연%
- 샘플: `samples/ecos/121Y002-BEABAA2-M-202401-202609.json`
- 고른 이유: 저축성수신 평균

#### 28. M2 평잔(원계열)

- 상태: verified
- id: `ecos-m2-avg`
- 이름: M2(평잔, 원계열)
- API: ECOS StatisticSearch
- 통계표: M2 상품별 구성내역(평잔, 원계열) (`161Y006`)
- 항목 코드: `BBHA00` (M2(평잔, 원계열))
- 주기: M, 차트 기간 10 years
- 기간 key: `TIME`, 값 key: `DATA_VALUE`, 단위 key: `UNIT_NAME`
- 응답 단위: 십억원
- 샘플: `samples/ecos/161Y006-BBHA00-M-202401-202609.json`
- 고른 이유: 지금 기준 표. 101Y004는 2004-09에 끝난 옛 표

### 부채·금융안정

#### 29. 가계신용

- 상태: verified
- id: `ecos-household-credit`
- 이름: 가계신용(잔액)
- API: ECOS StatisticSearch
- 통계표: 가계신용(업권별, 분기) (`151Y001`)
- 항목 코드: `1000000` (가계신용)
- 주기: Q, 차트 기간 20 years
- 기간 key: `TIME`, 값 key: `DATA_VALUE`, 단위 key: `UNIT_NAME`
- 응답 단위: 십억원
- 샘플: `samples/ecos/151Y001-1000000-Q-2024Q1-2026Q2.json`
- 고른 이유: 연·분기 중 분기

#### 30. 주택담보대출

- 상태: verified
- id: `ecos-housing-loan`
- 이름: 주택관련대출(예금취급기관, 잔액)
- API: ECOS StatisticSearch
- 통계표: 예금취급기관 가계대출(용도별, 월) (`151Y005`)
- 항목 코드: `11100A0` (주택관련대출-예금취급기관)
- 주기: M, 차트 기간 10 years
- 기간 key: `TIME`, 값 key: `DATA_VALUE`, 단위 key: `UNIT_NAME`
- 응답 단위: 십억원
- 샘플: `samples/ecos/151Y005-11100A0-M-202401-202608.json`
- 고른 이유: ECOS 잔액에는 주택담보대출 항목이 없고 주택관련대출(주택담보대출 포함)만 있다. 이 항목으로 넣는다(Sun&Moon 결정)

#### 31. 기업신용

- 상태: excluded
- 이유: ECOS·KOSIS 후보 표에서 찾지 못했다(Sun&Moon 결정: 이번에는 뺀다)

#### 32. 개인사업자 대출

- 상태: excluded
- 이유: ECOS·KOSIS 후보 표에서 찾지 못했다(Sun&Moon 결정: 이번에는 뺀다)

#### 33. 은행 연체율

- 상태: pending
- 이유: 금융감독원 표로 보인다. -org Issue(샘플 도구 KOSIS orgId) 뒤에 확인한다(Sun&Moon 결정, 2026-11-02)

### 대외경제

#### 34. 원·달러 환율

- 상태: verified
- id: `ecos-usd-krw`
- 이름: 원·달러 환율(월평균)
- API: ECOS StatisticSearch
- 통계표: 주요국 통화의 대원화환율 (`731Y004`)
- 항목 코드: `0000001` (원/미국달러(매매기준율))
- 분류 코드: `0000100` (ECOS ITEM_CODE2…)
- 주기: M, 차트 기간 10 years
- 기간 key: `TIME`, 값 key: `DATA_VALUE`, 단위 key: `UNIT_NAME`
- 응답 단위: 원
- 샘플: `samples/ecos/731Y004-0000001-0000100-M-202401-202610.json`
- 고른 이유: 일(731Y001)과 월이 있어 월. ITEM_CODE2 0000100 평균자료

#### 35. 경상수지

- 상태: verified
- id: `ecos-current-account`
- 이름: 경상수지(원계열)
- API: ECOS StatisticSearch
- 통계표: 국제수지 (`301Y013`)
- 항목 코드: `000000` (경상수지)
- 주기: M, 차트 기간 10 years
- 기간 key: `TIME`, 값 key: `DATA_VALUE`, 단위 key: `UNIT_NAME`
- 응답 단위: 백만달러
- 샘플: `samples/ecos/301Y013-000000-M-202401-202608.json`
- 고른 이유: 원계열. 계절조정은 301Y017

#### 36. 수출액

- 상태: pending
- 이유: 관세청 KOSIS 표로 보인다. ECOS 후보(901Y011, 901Y012)는 없는 표였다. -org Issue(샘플 도구 KOSIS orgId) 뒤에 확인한다(Sun&Moon 결정, 2026-11-02)

#### 37. 수입액

- 상태: pending
- 이유: 관세청 KOSIS 표로 보인다. -org Issue(샘플 도구 KOSIS orgId) 뒤에 확인한다(Sun&Moon 결정, 2026-11-02)

#### 38. 무역수지

- 상태: pending
- 이유: 관세청 KOSIS 표로 보인다. -org Issue(샘플 도구 KOSIS orgId) 뒤에 확인한다(Sun&Moon 결정, 2026-11-02)

#### 39. 일평균 수출액

- 상태: pending
- 이유: 관세청·산업통상자원부 표로 보인다. -org Issue(샘플 도구 KOSIS orgId) 뒤에 확인한다(Sun&Moon 결정, 2026-11-02)

### 주택

#### 40. 주택매매가격지수

- 상태: pending
- 이유: 한국부동산원 KOSIS 표로 확인한다(Sun&Moon 결정). ECOS 901Y062는 KB 지수. -org Issue(샘플 도구 KOSIS orgId) 뒤에 확인한다(Sun&Moon 결정, 2026-11-02)

#### 41. 주택전세가격지수

- 상태: pending
- 이유: 한국부동산원 KOSIS 표로 확인한다(Sun&Moon 결정). ECOS 901Y063은 KB 지수. -org Issue(샘플 도구 KOSIS orgId) 뒤에 확인한다(Sun&Moon 결정, 2026-11-02)

#### 42. 미분양 주택

- 상태: pending
- 이유: 국토교통부 KOSIS 표로 보인다. ECOS 901Y074는 재게시. -org Issue(샘플 도구 KOSIS orgId) 뒤에 확인한다(Sun&Moon 결정, 2026-11-02)

### 인구

#### 43. 생산가능인구(15~64세)

- 상태: verified
- id: `kosis-working-age-population`
- 이름: 생산가능인구(15~64세)
- API: KOSIS statisticsParameterData
- 통계표: 연령별 경제활동인구 총괄 (`DT_1DA7002S`), orgId `101`
- 항목 코드: `T10` (인구)
- 분류 코드: `63` (KOSIS objL1, objL2…)
- 주기: M, 차트 기간 10 years
- 기간 key: `PRD_DE`, 값 key: `DT`, 단위 key: `UNIT_NM`
- 응답 단위: 천명
- 샘플: `samples/kosis/DT_1DA7002S-T10-63-M-202401-202609.json`
- 고른 이유: 경제활동인구조사의 15~64세 인구(Sun&Moon 결정). 장래인구추계(DT_1BPA002)는 쓰지 않는다

#### 44. 65세 이상 인구 비율

- 상태: verified
- id: `kosis-elderly-ratio`
- 이름: 65세 이상 인구 비율(장래인구추계)
- API: KOSIS statisticsParameterData
- 통계표: 주요 인구지표(성비,인구성장률,인구구조,부양비 등) / 전국 (`DT_1BPA002`), orgId `101`
- 항목 코드: `T10` (인구항목)
- 분류 코드: `1`, `33` (KOSIS objL1, objL2…)
- 주기: A, 차트 기간 30 years
- 기간 key: `PRD_DE`, 값 key: `DT`, 단위 key: `UNIT_NM`
- 응답 단위: (없음), 관리자 단위(unit): `%`
- 샘플: `samples/kosis/DT_1BPA002-T10-1-33-Y-2015-2025.json`
- 고른 이유: 중위 추계(AA=1), 구성비 65세 이상(D=33). 비율을 바로 주는 통계청 표는 이것뿐이다. 관리자 단위 %

## 그 밖의 후보 (D024 이전 목록)

D024에 들어가지 않은 이전 후보다. 이번 범위가 아니다.

- ECOS: CD 91일 금리 (`721Y001` 항목 `2010000`에 있다, 조사만 함)
- ECOS 채권 관련 (pending): 채권종류별 거래 (`901Y015`), 채권시장별 거래 (`901Y120`), 주요 국공채 발행액/잔액 (`191Y001`)
- ECOS: 생산자물가지수(지수) (`404Y014` 항목 `*AA` 총지수에 있다, 조사만 함)
- KOSIS: 청년실업률 (`DT_1DA7002S` 항목 `T80`, 연령 분류 `75` 15-29세가 있다, 데이터는 받지 않았다)

## 샘플 관리

- 지표별 실제 정상 응답 하나를 우선 확보한다.
- 오류 응답은 API별로 실제 확인됐을 때 추가한다.
- 결측·수정·단위 변경은 실제 발견 시 기록한다.
- 합성 샘플은 `synthetic-` 접두어와 생성 이유를 표시한다.
