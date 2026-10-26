# DB 설계서

- 상태: decided (KEDP001-D022, 2026-10-26)
- 근거 결정: KEDP001-D002(기술 구성), KEDP001-D004(API 조사 뒤 스키마 결정), KEDP001-D017(MVP 범위), KEDP001-D018(관리자 화면), KEDP001-D019(수집 주기), KEDP001-D020(운영 최소 기준), KEDP001-D021(Go 수집 모듈), KEDP001-D022(DB 저장 방식과 데이터 저장 규칙)
- 근거 자료: `specification/10.indicators/README.md`, `specification/50.web/README.md`, `b-go-end/config/indicators.yaml`, `samples/`, `samples/repository/16-period/`
- 저장 방식과 규칙은 KEDP001-D022로 확정했다(6장). 테이블 정의는 `b-sql-db/migrations/`를 따른다(4장).
- 아직 할 일은 7장에 둔다.

---

# 1. 세 제공처 응답 비교

## 1.1 근거 샘플

| 제공처 | 샘플 파일 | 지표 | 주기 | 행 수 |
| --- | --- | --- | --- | --- |
| ECOS | `samples/ecos/722Y001-0101000-M-202401-202609.json` | 기준금리 | M | 33 |
| KOSIS | `samples/kosis/DT_1DA7001S-T90-0-M-202401-202609.json` | 고용률 | M | 33 |
| KOSIS | `samples/kosis/INH_1B8000F_01-T1-11-Y-2015-2025.json` | 출생아 수 | Y(응답 A) | 11 |
| KOSIS | `samples/kosis/DT_1IN1502-T100-00-Y-2015-2025.json` | 총인구 | Y(응답 A) | 11 |
| 공공데이터포털 | `samples/data-go-kr/069500-D-20260901-20261022.json` | KODEX 200 | D | 34 |
| 공공데이터포털 | `samples/data-go-kr/122630-D-20260901-20261022.json` | KODEX 레버리지 | D | 34 |
| 공공데이터포털 | `samples/data-go-kr/114800-D-20260901-20261022.json` | KODEX 인버스 | D | 34 |
| ECOS | `samples/repository/16-period/ecos/722Y001-0101000-A-2023-2025.json` | 기준금리 | A | 3 |
| ECOS | `samples/repository/16-period/ecos/722Y001-0101000-Q-2025Q1-2025Q3.json` | 기준금리 | Q | 3 |
| ECOS | `samples/repository/16-period/ecos/722Y001-0101000-D-20261001-20261023.json` | 기준금리 | D | 23 |
| KOSIS | `samples/repository/16-period/kosis/DT_1DA7001S-T90-0-Q-202401-202402.json` | 고용률 | Q | 2 |

오류·데이터 없음 샘플: `samples/ecos/error-INFO-100.json`, `error-INFO-200.json`, `samples/kosis/error-11.json`, `error-30.json`, `samples/data-go-kr/error-30.json`, `nodata-totalCount-0.json`

- `samples/repository/16-period/`는 기간 형식을 확인하려고 2026-10-26에 받은 조사 샘플이다(`b-go-end/cmd/sample`). 파일마다 요청 정보(`*.request.json`)가 옆에 있다. 조사 샘플은 git에 올리지 않는다(`.claude/environment.md`).
- CPI, GDP, 외환보유액(MVP 처음 화면 지표)은 아직 샘플이 없다(7장 남은 일 20).

## 1.2 비교표

| 구분 | ECOS | KOSIS | 공공데이터포털 |
| --- | --- | --- | --- |
| 응답 모양 | `StatisticSearch.row` 배열, `list_total_count` | 최상위가 행 배열 | `response.body.items.item` 배열, `totalCount` |
| 값 key | `DATA_VALUE` | `DT` | 한 행에 값이 여러 개(`clpr`, `nav`, `mkp`, `hipr`, `lopr`, `vs`, `fltRt`, `trqu` 등). 지표 값은 `clpr`만 쓴다(결정 8) |
| 값 형식 | 문자열. `"3.5"`, `"3"` (소수 자릿수가 행마다 다름) | 문자열. `"61"`, `"61.6"`, `"51069375"` | 문자열. `"86525"`, `"86756.18"`, `fltRt`는 `".25"`, `"-.39"`(앞 0 없음) |
| 빈 값·결측 | 샘플에 없음. 저장 규칙은 5.5 | 샘플에 없음. 저장 규칙은 5.5 | 샘플에 없음. 저장 규칙은 5.5 |
| 단위 | `UNIT_NAME` (`연%`) | `UNIT_NM` (`%`, `명`). 출생아 수 표는 `UNIT_NM` 없음, `C1_NM`(`출생아수(명)`)에 들어 있음 | 단위 항목 없음. 관리자가 단위를 넣는다(결정 9) |
| 기간 key | `TIME` | `PRD_DE` | `basDt` |
| 기간 형식 | D: `20261001`, M: `202401`, Q: `2025Q1`, A: `2023` | M: `202401`, Q: `202401`(끝 두 자리가 분기), A: `2015` | D: `20261022` |
| 주기 표시 | 응답에 없음(요청 경로에만 있음) | `PRD_SE` (`M`, `Q`, `A`). 요청 `prdSe=Y`가 응답에서 `A` | 응답에 없음. 일만 있음 |
| 행 순서 | 오래된 것부터 | 오래된 것부터 | 최근 것부터 |
| 분류 | `ITEM_CODE1`~`4`(샘플은 1만 값, 2~4는 null) | `C1`(objL1), 이름 `C1_NM`, 분류 이름 `C1_OBJ_NM`. C2 이상은 샘플 없음(7장 남은 일 19) | 없음. 종목 코드 `srtnCd`가 곧 계열 |
| 이름 | `STAT_NAME`, `ITEM_NAME1` | `TBL_NM`, `ITM_NM`, `C1_NM` (+ `_ENG`) | `itmsNm`, `bssIdxIdxNm` |
| 변경 일자 | 없음 | `LST_CHN_DE` (예: `2024-02-01`, 행마다 다름) | 없음 |
| 논리 key 후보 | `STAT_CODE` + `ITEM_CODE1`(~4) + 주기 + `TIME` | `ORG_ID` + `TBL_ID` + `ITM_ID` + `C1` + `PRD_SE` + `PRD_DE` | `srtnCd` + `basDt` (+ 값 항목 이름) |
| 데이터 없음 | `RESULT.CODE` `INFO-200` | `err` `30` | `resultCode` `00` + `totalCount` 0 |
| 오류 | `RESULT.CODE` `INFO-100` 등 | `{"err","errMsg"}` (HTTP 200) | `OpenAPI_ServiceResponse.cmmMsgHeader.returnReasonCode` (HTTP 403 등) |

## 1.3 비교에서 나온 점

- 세 곳 모두 **한 행 = 한 기간의 값**이다. 기간, 값, 단위, 계열 식별자로 정리하면 같은 모양이 된다.
- 계열 식별자 구성은 제공처마다 다르다(ECOS 2~5개, KOSIS 4개 이상, 공공데이터포털 1개). 이 차이는 "지표 정의" 쪽에 두고, 관측값 쪽은 공통으로 둘 수 있다.
- 공공데이터포털은 한 행에 값이 여러 개다. 지표 값은 `clpr`만 쓰고 나머지는 원본 응답에 남긴다(결정 8).
- 단위는 제공처마다 표기가 다르고(`연%`, `%`), 없는 경우도 있다. 응답 단위만으로는 화면 단위를 정할 수 없으므로 관리자 단위를 둔다(결정 9).
- 값 변경을 알 수 있는 단서는 KOSIS `LST_CHN_DE`뿐이다. 이것이 "처음 공표일"인지 "마지막 수정일"인지는 샘플만으로 구분할 수 없다. 이력(5.2)은 이 값과 상관없이 값을 비교해 남기므로 따로 확인하지 않는다(KEDP001-D022 5절 17).

---

# 2. DB에 저장할 것과 저장하지 않을 것

## 2.1 저장할 것

| 무엇 | 근거 |
| --- | --- |
| 지표 정의 (제공처, 통계 코드, 항목 코드, 분류, 주기) | D018(지표 추가를 관리자 화면에서, DB에 저장). `indicators.yaml`은 DB가 생기기 전까지 임시 |
| 메인 화면 노출 여부, 순서 | D017, D018 |
| 지표별 차트 기간, 관련 지표, 숫자·날짜 표시 형식 | D018 |
| 수집 시각 (기본 06:00, Asia/Seoul) | D019(수집 시각 설정은 DB에 저장) |
| 수집한 관측값 (지표, 기간, 값) | D017(수집 → 저장 → 조회) |
| 수집 실행 기록 (마지막 수집 일시, 성공·실패) | D018(API 관리 화면에 마지막 수집 일시), D019(상세 화면에 마지막 수집 일시) |
| 원본 응답 (오류·데이터 없음 포함, 90일 보관) | 결정 5, 6, 13 (5.3) |

## 2.2 저장하지 않을 것

| 무엇 | 근거 |
| --- | --- |
| API 키 값 | D018(키는 화면에서 다루지 않고 `.env`에서 관리), `.claude/api.md` |
| API 키 "설정됨/안 됨" 상태 | D018. `.env`에 있는지를 그때그때 보여주는 것이라 DB에 둘 필요가 없다 |
| 키가 들어간 요청 주소 | `.claude/api.md`. 기록이 필요하면 키 자리를 `***`로 가린다 |
| 관리자 계정, 비밀번호 | D018(서버의 파일 하나로 관리) |
| 허용 IP | D018(실제 값은 저장소에 쓰지 않음). 저장 위치는 인증 파일 또는 서버 설정 |
| 지표 비교, AI 분석·요약, 예측 결과 | D017(2차 이후) |
| Rust 서버 상태 확인·재시작 기록 | D020(MVP 이후) |
| 백업 기록 | D020(스크립트 + crontab). DB 밖에서 관리 |

---

# 3. 저장 방식 비교

| 방식 | 모양 | 장점 | 단점 |
| --- | --- | --- | --- |
| 제공처별 | `ecos_data`, `kosis_data`, `data_go_kr_data`처럼 제공처마다 응답 칼럼을 그대로 담는 테이블 | 응답과 1:1이라 이해하기 쉽다. 제공처 고유 정보(`LST_CHN_DE`, `nav` 등)를 잃지 않는다 | 조회 API가 제공처마다 따로 읽어야 한다. 화면은 지표 하나를 같은 모양으로 보여주므로 합치는 코드가 늘어난다. 제공처가 늘면 테이블이 늘어난다 |
| 범용 | `indicator` + `observation`(지표 id, 기간, 값) 두 테이블로 모든 제공처를 담는다 | 조회가 단순하다. 지표 추가(D018)가 행 추가로 끝난다 | 제공처 고유 정보가 빠진다. 공공데이터포털처럼 값이 여러 개인 행을 담기 어렵다. 원본과 비교·재처리가 어렵다 |
| 혼합 | 지표 정의는 공통 테이블에, 제공처별 식별자는 지표 정의의 칼럼(또는 jsonb)으로. 관측값은 공통 테이블 | 조회는 범용처럼 단순하다. 제공처 차이는 지표 정의 한 곳에 모인다 | 지표 정의 칼럼 중 일부는 제공처에 따라 비어 있다. 어떤 칼럼을 공통으로 둘지 정해야 한다 |
| Staging-Result | 받은 응답을 그대로 담는 staging, 정리·검증한 결과를 담는 result로 나눈다 | 원본이 남아 재처리·검증·감사가 쉽다. Rust의 정규화·검증 역할(D002)과 맞는다 | 테이블과 처리 단계가 늘어난다. 원본 보관 기간, 용량 관리가 필요하다. MVP에는 무겁다 |

## 결정: 혼합 (원본 응답은 따로 보관, 결정 1)

- 관측값 테이블은 범용으로 하나만 둔다. 화면(대시보드, 상세)은 지표 하나의 시계열만 읽으면 된다(D017).
- 제공처별 식별자(통계 코드, 항목 코드, 분류)는 지표 정의 테이블에 둔다. 1.3에서 본 것처럼 제공처 차이는 여기에만 있다.
- Staging-Result를 다 하지 않고, 원본 응답만 별도 테이블에 남긴다. 재처리가 필요할 때 다시 쓸 수 있고, 수집 단계를 늘리지 않는다(5.3).
- 관측값 수는 적다. 10개 지표, 하루 1회 수집(D019)이며 일 주기도 지표 하나에 연 366행 이하다(ETF는 거래일만 와서 연 250행 안팎). 테이블을 나눌 이유가 성능에서 나오지 않는다.

---

# 4. 테이블

정의는 `b-sql-db/migrations/0001_schema.up.sql`이 기준이다. 아래는 칼럼의 뜻과 규칙만 적는다.

| 테이블 | 역할 | 주요 칼럼 |
| --- | --- | --- |
| `source` | 제공처 목록 | `code`(`ecos`, `kosis`, `data-go-kr`), `name`. migration이 세 행을 넣는다 |
| `indicator` | 지표 정의. `indicators.yaml`을 대신한다 | `id`(예: `ecos-base-rate`, 화면 주소 `/indicator/{id}`에도 사용. 소문자·숫자·`-`만), `name`, `source_code`, `org_id`(KOSIS), `stat_code`, `item_code`(공공데이터포털은 비움), `classes`(text 배열, KOSIS objL1…), `value_field`(공공데이터포털 값 항목, `clpr`), `cycle`(D/M/Q/A), `unit`(관리자 단위), `source_unit`(응답 단위), `created_at`, `updated_at` |
| `indicator_display` | 화면 설정 (Dashboard 관리) | `indicator_id`, `is_visible`(기본 꺼짐), `sort_order`, `chart_period`(interval, 예: `10 years`), `number_format`(jsonb), `date_format`, `notice`(안내 문구, 없으면 비움) |
| `indicator_relation` | 관련 지표 | `indicator_id`, `related_indicator_id`, `sort_order`. 자기 자신은 넣지 못한다 |
| `app_setting` | 전체 설정 | `key`, `value`, `updated_at`. migration이 `collect_time` = `06:00`(Asia/Seoul, KEDP001-D019)을 넣는다 |
| `collection_run` | 수집 실행 기록. 마지막 수집 일시와 연결 상태의 근거 | `id`, `source_code`, `indicator_id`(제공처 전체 실행이면 비움), `started_at`, `finished_at`, `status`(`running`, `success`, `no_data`, `error`), `error_code`(예: `INFO-100`, `11`, `30`), `row_count`, `trigger`(`schedule`, `manual`) |
| `observation` | 관측값 (화면이 읽는 값) | `indicator_id`, `cycle`, `period_start`(date, 5.1), `period_raw`(응답 원문), `value`(numeric), `value_raw`(응답 문자열), `source_changed_at`(KOSIS `LST_CHN_DE`, 다른 곳은 비움), `collected_at`, `run_id`. 기본 키: `indicator_id` + `cycle` + `period_start` |
| `observation_history` | 고쳐진 값의 이전 값 (5.2) | `id`, `indicator_id`, `cycle`, `period_start`, `old_value`, `new_value`, `changed_at`, `run_id`(새 값을 가져온 수집) |
| `raw_response` | 원본 응답 (5.3) | `id`, `run_id`, `received_at`, `request_masked`(키 자리 `***`), `http_status`, `body`(jsonb), `body_text`(JSON이 아닌 응답) |

참고:

- 마지막 수집 일시는 `collection_run`에서 구한다. 따로 칼럼을 두지 않는다.
- 시각 칼럼은 timestamptz로 두고, 화면 표시는 Asia/Seoul로 한다.
- `observation`에 `cycle`을 넣은 이유: 관리자가 지표의 수집 주기를 바꿀 수 있고, 이전 주기 값은 남긴다(결정 11).
- `indicator`는 제공처마다 꼭 필요한 칸을 검사한다: ECOS는 `item_code`, KOSIS는 `org_id`와 `item_code`, 공공데이터포털은 `value_field`가 있어야 하고 `item_code`는 비어야 한다.
- `collection_run.status`의 `running`은 수집을 시작했고 아직 끝나지 않은 상태다. `running`일 때만 `finished_at`이 비어 있다.
- 지표를 지울 때: 관측값이 있으면 지울 수 없다. 화면 설정과 관련 지표는 함께 지워지고, 수집 기록의 `indicator_id`는 비워진다.
- `number_format`의 내용(자릿수, 소수점)은 화면 작업에서 정한다.

## 4.1 접속 계정과 권한 (결정 14)

정의는 `b-sql-db/migrations/0002_roles.up.sql`이 기준이다. 비밀번호는 저장소에 쓰지 않는다. migration을 적용한 뒤 소유자가 psql의 `\password`로 정한다.

| 계정 | 쓰기 | 읽기만 |
| --- | --- | --- |
| `kedp_go` | `indicator`, `indicator_display`, `indicator_relation`, `app_setting`(추가·수정·삭제), `collection_run`(추가·수정) | `source`, `observation`(조회 API) |
| `kedp_rust` | `observation`(추가·수정), `observation_history`(트리거가 추가), `raw_response`(추가) | `source`, `indicator` |

- `kedp_rust`는 `observation`, `raw_response`도 읽을 수 있다(덮어쓰기와 재처리).
- 테이블을 만들고 고치는 일(migration)은 소유자 계정으로 한다.

---

# 5. 저장 규칙

## 5.1 기간 표기 통일 (결정 2, 3)

| 주기 | ECOS | KOSIS | 공공데이터포털 | `period_start` |
| --- | --- | --- | --- | --- |
| D | `20261001` | 해당 없음 | `20261022` | 그 날 (`2026-10-01`) |
| M | `202401` | `202401` | 해당 없음 | 그 달 1일 (`2024-01-01`) |
| Q | `2025Q1`, `2025Q2`, `2025Q3` | `202401`, `202402` (끝 두 자리가 분기) | 해당 없음 | 분기 첫날 (`2025Q2` → `2025-04-01`, KOSIS `202402` → `2024-04-01`) |
| A | `2023` | `2015` | 해당 없음 | 1월 1일 (`2023-01-01`) |

근거 파일:

- ECOS D: `samples/repository/16-period/ecos/722Y001-0101000-D-20261001-20261023.json` (`TIME` `20261001`~`20261023`)
- ECOS M: `samples/ecos/722Y001-0101000-M-202401-202609.json`
- ECOS Q: `samples/repository/16-period/ecos/722Y001-0101000-Q-2025Q1-2025Q3.json` (`TIME` `2025Q1`~`2025Q3`)
- ECOS A: `samples/repository/16-period/ecos/722Y001-0101000-A-2023-2025.json` (`TIME` `2023`~`2025`)
- KOSIS M: `samples/kosis/DT_1DA7001S-T90-0-M-202401-202609.json`
- KOSIS Q: `samples/repository/16-period/kosis/DT_1DA7001S-T90-0-Q-202401-202402.json` (`PRD_SE` `Q`, `PRD_DE` `202401`, `202402`)
- KOSIS A: `samples/kosis/DT_1IN1502-T100-00-Y-2015-2025.json`
- 공공데이터포털 D: `samples/data-go-kr/069500-D-20260901-20261022.json`

규칙:

- 기간 시작일(date), 주기, 응답 원문을 함께 저장한다(결정 2).
- 연 주기 코드는 `A`다. KOSIS 요청 값 `Y`는 요청할 때만 쓴다(결정 3).
- KOSIS는 M과 Q가 둘 다 6자리다. 원문만으로는 구분할 수 없으므로 주기(응답 `PRD_SE`)를 함께 보고 바꾼다.
- KOSIS 분기 `202402`가 2분기라는 것은 값으로 확인했다. 분기 값 `63.3`은 월 샘플의 2024년 4~6월 평균(`63.0`, `63.5`, `63.5` → 63.33)과 같고, 2월 값 `61.6`과 다르다.
- 요청 값도 응답과 같은 형식이다. ECOS 분기는 `2025Q1`, KOSIS 분기는 `startPrdDe=202401`로 요청했다(샘플 옆 `*.request.json`).
- ECOS 일 자료는 주말에도 행이 온다(기준금리 2026-10-01~23 요청에 23행).
- DB는 `period_start`가 주기의 첫날인지 검사한다(`observation_period_start_check`). 일 주기는 검사하지 않는다.

## 5.2 값이 고쳐졌을 때 (결정 4)

같은 지표·주기·기간의 값이 다시 수집됐는데 값이 다를 때.

| 선택지 | 내용 | 장단점 |
| --- | --- | --- |
| A. 덮어쓰기 | 새 값으로 바꾼다 | 가장 단순. 바뀐 사실이 남지 않는다 |
| B. 덮어쓰기 + 이력 | `observation`은 새 값, 이전 값은 `observation_history`에 | 화면 조회는 단순하고, 바뀐 기록이 남는다. 테이블 하나가 늘어난다 |
| C. 모두 쌓기 | 수집마다 행을 더하고 최신 것을 읽는다 | 기록이 완전하다. 조회가 복잡하고 행이 매일 늘어난다 |

- 결정: B. 값이 실제로 달라졌을 때만 이력에 남긴다.
- 이력은 `observation`의 트리거(`observation_keep_history`)가 남긴다. Rust는 `observation`만 덮어쓴다(`INSERT … ON CONFLICT … DO UPDATE`).
- 값은 숫자로 비교한다. `3.5`를 `3.50`으로 다시 쓰면 이력이 남지 않는다.

## 5.3 원본 응답 보관 (결정 5, 6, 13)

| 선택지 | 내용 |
| --- | --- |
| A. 보관 안 함 | 정리된 값만 저장 |
| B. DB `raw_response`(jsonb)에 보관 | 재처리·확인 가능. 크기는 샘플 기준 한 번에 수 KB~13KB |
| C. 파일로 보관 | DB가 가볍다. 백업(D020)과 따로 관리해야 한다 |

- 결정: B. 90일 보관한다. 지우는 작업은 MVP 이후에 만들고, 그때까지는 쌓기만 한다(KEDP001-D022 영향). `received_at` 색인을 둔다.
- 오류·데이터 없음 응답도 수집 기록과 함께 남긴다.
- 요청 주소는 키를 가린 형태로만 저장한다(`.claude/api.md`). DB는 `request_masked`에 `***`가 있는지 검사한다.
- JSON 응답은 `body`(jsonb), JSON이 아닌 응답(예: 게이트웨이 오류 문서)은 `body_text`에 넣는다. 둘 중 하나만 쓴다.

## 5.4 DB에 쓰는 쪽 (결정 7)

| 데이터 | 쓰는 쪽 | 이유 |
| --- | --- | --- |
| 관측값, 원본 응답 | Rust | D002 구조가 `Rust Processor → PostgreSQL`이고, 정규화·검증한 쪽이 바로 쓰는 것이 단순하다 |
| 수집 실행 기록 | Go | 예약 실행(D019)과 오류 응답 판정이 Go에 있다. Rust의 저장 결과(행 수, 실패)는 gRPC 응답으로 받는다 |
| 지표 정의, 화면 설정, 수집 시각 | Go | 관리자 화면 API가 Go에 있다(D002, D018) |

- 계정과 권한은 4.1과 같다.
- Rust가 원본 응답을 저장하려면 Go가 응답 원문을 gRPC로 넘겨야 한다. `b-proto/` 변경이 필요하며 소유자 승인 뒤 바꾼다(KEDP001-D007).
- Rust가 멈추면 관측값 저장도 멈춘다. MVP에서는 받아들인다(KEDP001-D020).

## 5.5 빈 값(결측) (결정 15)

- 결측은 관측값으로 저장하지 않는다. 원본 응답에는 남는다.
- `observation.value`는 numeric이고 비울 수 없다. 비었거나 숫자로 바꿀 수 없는 값은 들어갈 수 없다.
- 결측이 어떤 표기(`""`, `"-"`, null 등)로 오는지는 아직 샘플로 보지 못했다.

---

# 6. 결정 (KEDP001-D022)

| # | 정할 것 | 결정 |
| --- | --- | --- |
| 1 | 저장 방식 | 혼합. 지표 정의 테이블 하나에 제공처별 코드를 두고, 관측값 테이블 하나에 세 제공처를 담는다. 원본 응답은 따로 둔다 (3장) |
| 2 | 기간 저장 | 기간 시작일(date)과 응답 원문을 함께 저장한다 (5.1) |
| 3 | 연 주기 코드 | `A`. KOSIS 요청 값 `Y`는 요청할 때만 쓴다 (5.1) |
| 4 | 값이 고쳐졌을 때 | 덮어쓰고 이전 값은 이력에 남긴다. 값이 실제로 달라졌을 때만 남긴다 (5.2) |
| 5 | 원본 응답 | DB에 보관한다. 요청 주소는 키를 가린 형태로만 저장한다 (5.3) |
| 6 | 원본 보관 기간 | 90일 (5.3) |
| 7 | DB에 쓰는 쪽 | 관측값·원본 응답은 Rust, 수집 기록·지표 정의·화면 설정·수집 시각은 Go (5.4) |
| 8 | 공공데이터포털 값 | 종가(`clpr`)만 지표 값으로 쓴다. 나머지는 원본 응답에 남는다 |
| 9 | 화면 단위 | 응답 단위와 관리자 단위를 둘 다 저장하고 관리자 값을 우선한다 |
| 10 | 값 저장 형식 | numeric과 응답 원문 문자열을 함께 저장한다 |
| 11 | 관리자가 주기를 바꿨을 때 | 이전 주기 값은 남긴다. 관측값에 주기를 함께 저장한다 |
| 12 | 처음 수집할 과거 범위 | 지표의 차트 기간만큼 (`indicator_display.chart_period`) |
| 13 | 오류·데이터 없음 응답 | 수집 기록과 원본 응답을 모두 저장한다 (5.3) |
| 14 | DB 접속 계정 | Go와 Rust를 나눈다. 비밀번호는 저장소에 쓰지 않는다 (4.1) |
| 15 | 빈 값(결측) | 관측값으로 저장하지 않는다. 원본 응답에는 남는다 (5.5) |

"확인 못 함"으로 두었던 16~22는 KEDP001-D022 5절에서 정했다. 16(기간 형식)은 5.1에 채웠다. 17·18·21·22는 DB 설계에서 더 확인하지 않으므로 지웠다. 19·20은 7장에 둔다.

---

# 7. 남은 일

DB 설계에서는 다루지 않고 남겨 둔 것이다(KEDP001-D022 5절).

| # | 항목 | 결정 | 지금 스키마 |
| --- | --- | --- | --- |
| 19 | KOSIS 분류가 2개 이상(`C2` 이상)인 표 | 확인할 수 있는 표가 없어 다루지 않는다 | `indicator.classes`는 배열이라 분류 코드 개수와 상관없이 담는다 |
| 20 | CPI, GDP, 외환보유액 | DB 설계를 위해 샘플을 더 받지 않는다. 항목 코드는 `10.indicators`에서 pending이다 | 다른 지표와 같은 테이블에 행으로 추가한다 |
