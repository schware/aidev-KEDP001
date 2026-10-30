# 사용자 조회 API

- 상태: draft
- 근거: KEDP001-D016(net/http), KEDP001-D017(MVP 범위), KEDP001-D022 9·11(단위, 주기), `specification/50.web/README.md` 2·3장, Issue #36
- 구현: `b-go-end/internal/api`, `b-go-end/internal/store/read.go`
- 관리자 API는 다음 Issue에서 이 문서에 더한다.

화면(TypeScript)은 이 문서만 보고 API를 쓴다. ECOS·KOSIS·DB를 직접 부르지 않는다.

---

# 1. 공통

| 항목 | 내용 |
| --- | --- |
| 서버 | Go `cmd/server`. 수집과 같은 프로그램에서 함께 돈다 |
| 주소 | `.env`의 `KEDP_API_ADDR`. 비우면 `127.0.0.1:8080` |
| 경로 | 모두 `/api/`로 시작한다 |
| method | `GET`만 받는다(`HEAD`도 된다) |
| 응답 | JSON, `Content-Type: application/json; charset=utf-8` |
| CORS | 두지 않는다. 화면 개발은 Vite proxy로 `/api`를 Go로 넘긴다 |
| 인증 | 없다(사용자 화면) |
| DB | `kedp_go` 계정으로 읽기만 한다 |

Vite proxy 예(`vite.config.ts`):

```ts
server: { proxy: { '/api': 'http://127.0.0.1:8080' } }
```

## 1.1 값의 형식

| 값 | 형식 | 예 |
| --- | --- | --- |
| 기간 `period` | 기간 시작일 `YYYY-MM-DD`. 주기에 맞게 화면이 바꿔 보여준다 | M `2026-09-01` → `2026-09`, Q `2025-04-01` → `2025 Q2`, A `2025-01-01` → `2025` |
| 값 `value` | JSON 숫자 | `2.5`, `51684564` |
| 주기 `cycle` | `D`, `M`, `Q`, `A`(연) | `M` |
| 시각 | Asia/Seoul, RFC 3339 | `2026-10-25T06:00:03+09:00` |
| 없는 값 | `null`. 목록은 비어 있으면 `[]`(`null`이 아니다) | |

- 기간 시작일 규칙은 `specification/30.database/README.md` 5.1을 따른다.
- 값의 소수 자릿수·단위 표시 형식(`indicator_display.number_format`, `date_format`)은 이 API에 아직 없다. 화면 작업에서 형식을 정한 뒤 더한다.

## 1.2 단위 `unit`

관리자 단위(`indicator.unit`)가 있으면 그것, 없으면 응답 단위(`indicator.source_unit`), 둘 다 없으면 `null`이다(KEDP001-D022 9). 빈 글자는 없는 것으로 본다.

## 1.3 관측값의 주기

관측값은 지표의 지금 주기(`indicator.cycle`) 것만 돌려준다. 관리자가 주기를 바꾸기 전의 값은 DB에 남아 있지만 보여주지 않는다(KEDP001-D022 11).

## 1.4 오류

| 상태 | 언제 | 응답 |
| --- | --- | --- |
| 404 | 없는 지표 id | `{"error":"indicator not found"}` |
| 404 | 없는 주소 | `{"error":"not found"}` |
| 405 | `GET`, `HEAD`가 아닌 method. `Allow: GET, HEAD` 헤더가 붙는다 | `{"error":"method not allowed"}` |
| 500 | DB 오류 등 서버 문제. 자세한 내용은 서버 기록에만 남는다 | `{"error":"internal error"}` |

---

# 2. 메인 화면 카드 목록

```text
GET /api/indicators
```

- 노출(`indicator_display.is_visible`)된 지표만 돌려준다. 화면 설정이 없는 지표도 나오지 않는다.
- 순서는 `sort_order`, 같으면 `id` 순서다. 화면은 받은 순서대로 카드를 놓는다.
- `latest`는 마지막 관측값이다. 아직 수집한 값이 없으면 `null`이다.

응답 200:

```json
{
  "indicators": [
    {
      "id": "ecos-base-rate",
      "name": "한국은행 기준금리",
      "cycle": "M",
      "unit": "연%",
      "latest": { "period": "2026-09-01", "value": 2.5 }
    },
    {
      "id": "data-go-kr-kodex-200",
      "name": "KODEX 200",
      "cycle": "D",
      "unit": null,
      "latest": null
    }
  ]
}
```

> 값은 예시다. 실제 수집값이 아니다.

| 칸 | 형식 | 뜻 |
| --- | --- | --- |
| `id` | 문자열 | 지표 id. 상세 화면 주소 `/indicator/{id}`에 쓴다 |
| `name` | 문자열 | 지표 이름 |
| `cycle` | 문자열 | 주기 |
| `unit` | 문자열 또는 `null` | 단위(1.2) |
| `latest` | 객체 또는 `null` | 최신값 `{period, value}` |

---

# 3. 지표 상세

```text
GET /api/indicators/{id}
```

- 노출하지 않는 지표도 id로 읽을 수 있다(관련 지표에서 들어올 수 있다).
- 없는 id는 404다.

응답 200:

```json
{
  "id": "ecos-base-rate",
  "name": "한국은행 기준금리",
  "cycle": "M",
  "unit": "연%",
  "notice": null,
  "source": { "code": "ecos", "name": "ECOS" },
  "stat_code": "722Y001",
  "last_collected_at": "2026-10-25T06:00:03+09:00",
  "latest": { "period": "2026-09-01", "value": 2.5 },
  "chart_period": "10 years",
  "chart": [
    { "period": "2016-11-01", "value": 1.25 },
    { "period": "2026-09-01", "value": 2.5 }
  ],
  "recent": [
    { "period": "2026-09-01", "value": 2.5 },
    { "period": "2026-08-01", "value": 2.5 }
  ],
  "related": [
    { "id": "kosis-employment-rate", "name": "고용률" }
  ]
}
```

> 값은 예시다. 실제 수집값이 아니다. `chart`, `recent`는 줄였다.

| 칸 | 형식 | 뜻 |
| --- | --- | --- |
| `id`, `name`, `cycle`, `unit` | | 2장과 같다 |
| `notice` | 문자열 또는 `null` | 상세 화면 맨 위 안내 문구(`indicator_display.notice`). 있으면 눈에 띄게 보여준다(예: KODEX 레버리지·인버스) |
| `source` | 객체 | 제공처 `code`(`ecos`, `kosis`, `data-go-kr`)와 화면 이름 `name`(`ECOS`, `KOSIS`, `공공데이터포털`) |
| `stat_code` | 문자열 | 통계 코드(ECOS 통계 코드, KOSIS 표 id, 공공데이터포털 종목 코드) |
| `last_collected_at` | 시각 또는 `null` | 마지막 수집 일시. 이 지표의 수집 기록(`collection_run`) 중 `success`로 끝난 마지막 시각이다. 데이터 없음·오류는 넣지 않는다. 없으면 `null` |
| `latest` | 객체 또는 `null` | 현재값(최신 관측값) |
| `chart_period` | 문자열 또는 `null` | 차트 기간(PostgreSQL interval 글, 예: `10 years`, `1 year`). 화면 설정이 없으면 `null` |
| `chart` | 배열 | 차트 데이터. 기간 시작일이 (오늘 − 차트 기간) 이후인 관측값, 오래된 것부터. 오늘은 Asia/Seoul 기준이다. 화면 설정이 없으면 `[]` |
| `recent` | 배열 | 최근 데이터 표. 최근 관측값 최대 10개, 최근 것부터 |
| `related` | 배열 | 관련 지표(`indicator_relation`) `{id, name}`. 관리자가 정한 순서다. 관련 지표가 노출하지 않는 지표여도 나온다 |

---

# 4. 확인 명령

서버를 띄운 뒤(`b-go-end/README.md`):

```sh
curl -s http://127.0.0.1:8080/api/indicators
curl -s http://127.0.0.1:8080/api/indicators/ecos-base-rate
curl -s -i http://127.0.0.1:8080/api/indicators/none     # 404
```
