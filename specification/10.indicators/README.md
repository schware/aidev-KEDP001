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

## 추가 후보

### ECOS

- 국고채 3년 금리
- 국고채 10년 금리
- 회사채 3년 AA- 금리
- CD 91일 금리
- 예금은행 수신금리
- 예금은행 대출금리
- 광의통화 M2
- 원·달러 환율
- 경상수지
- 생산자물가지수
- 실질 GDP 성장률

### ECOS 채권 관련 (pending, 추가 조사 중)

- 채권종류별 거래 (`901Y015`), 주기 A, M
- 채권시장별 거래 (`901Y120`), 주기 A, M
- 주요 국공채 발행액/잔액 (`191Y001`), 주기 A, M, Q

### KOSIS

- 생산가능인구
- 65세 이상 인구 비율
- 취업자 수
- 실업률
- 청년실업률
- 경제활동참가율

## 샘플 관리

- 지표별 실제 정상 응답 하나를 우선 확보한다.
- 오류 응답은 API별로 실제 확인됐을 때 추가한다.
- 결측·수정·단위 변경은 실제 발견 시 기록한다.
- 합성 샘플은 `synthetic-` 접두어와 생성 이유를 표시한다.
