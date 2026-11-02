# 조회 API와 관리자 API

- 상태: draft
- 근거: KEDP001-D016(net/http), KEDP001-D017(MVP 범위), KEDP001-D022 9·11(단위, 주기), `specification/50.web/README.md` 2·3장, Issue #36
- 관리자 API(5장) 근거: KEDP001-D012, D018, D019, D022, `specification/50.web/README.md` 1장 관리자, 4~6장, Issue #40
- 방문자 수 API(6장) 근거: Issue #48
- 지표 구분(`category`, 2·3장, 5.4~5.6, 5.11) 근거: KEDP001-D024, Issue #52
- 구현: `b-go-end/internal/api`, `b-go-end/internal/store/read.go`, 관리자 API는 `b-go-end/internal/admin`, `b-go-end/internal/store/admin.go`

화면(TypeScript)은 이 문서만 보고 API를 쓴다. ECOS·KOSIS·DB를 직접 부르지 않는다.

---

# 1. 공통

| 항목 | 내용 |
| --- | --- |
| 서버 | Go `cmd/server`. 수집과 같은 프로그램에서 함께 돈다 |
| 주소 | `.env`의 `KEDP_API_ADDR`. 비우면 `127.0.0.1:8080` |
| 경로 | 모두 `/api/`로 시작한다 |
| method | 사용자 조회 API는 `GET`만 받는다(`HEAD`도 된다). 방문 세기(`POST /api/visits`)는 6장, 관리자 API는 5장 |
| 응답 | JSON, `Content-Type: application/json; charset=utf-8` |
| CORS | 두지 않는다. 화면 개발은 Vite proxy로 `/api`를 Go로 넘긴다 |
| 인증 | 사용자 조회 API와 방문자 수 공개 API(6.2, 6.3)는 없다. 관리자 API는 허용 IP + 로그인(5장) |
| DB | `kedp_go` 계정. 사용자 조회 API는 읽기만 한다. 방문 세기는 날짜별 숫자만 쓴다(6장) |

Vite proxy 예(`vite.config.ts`). `xfwd: true`는 관리자 API가 허용 IP를 보는 데 쓴다(5.1). 배포에서는 nginx가 같은 일을 한다(`deploy/README.md`):

```ts
server: { proxy: { '/api': { target: 'http://127.0.0.1:8080', xfwd: true } } }
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
- 소수 자릿수 `decimals`: 관리자가 정한 소수 자릿수(0~4, `indicator_display.number_format`의 `{"decimals": n}`). 없으면 `null`이고, 받은 값 그대로 보여준다.
- 날짜 형식(`date_format`)은 MVP에서 쓰지 않는다. 기간은 위 표처럼 주기에 맞춰 보여준다.

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
- 순서는 구분 순서(`category.sort_order`)가 먼저이고, 같은 구분 안에서는 `sort_order`, 같으면 `id` 순서다. 화면은 받은 순서대로 구분을 묶고 카드를 놓는다.
- `latest`는 마지막 관측값이다. 아직 수집한 값이 없으면 `null`이다.

응답 200:

```json
{
  "indicators": [
    {
      "id": "ecos-base-rate",
      "name": "한국은행 기준금리",
      "category": { "code": "money", "name": "금리·통화" },
      "cycle": "M",
      "unit": "연%",
      "decimals": 2,
      "latest": { "period": "2026-09-01", "value": 2.5 }
    },
    {
      "id": "data-go-kr-kodex-200",
      "name": "KODEX 200",
      "category": { "code": "market", "name": "금융시장" },
      "cycle": "D",
      "unit": null,
      "decimals": null,
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
| `category` | 객체 | 지표 구분. `code`(예: `money`)와 화면 이름 `name`(예: `금리·통화`). 목록은 5.11 |
| `cycle` | 문자열 | 주기 |
| `unit` | 문자열 또는 `null` | 단위(1.2) |
| `decimals` | 숫자 또는 `null` | 소수 자릿수(1.1) |
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
  "category": { "code": "money", "name": "금리·통화" },
  "cycle": "M",
  "unit": "연%",
  "decimals": 2,
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
| `id`, `name`, `category`, `cycle`, `unit`, `decimals` | | 2장과 같다 |
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


---

# 5. 관리자 API

관리자 화면(`/admin/login`, `/admin/dashboard`, `/admin/api`, `specification/50.web/README.md` 4~6장)이 쓴다.

## 5.1 접근과 인증 (KEDP001-D018)

| 항목 | 내용 |
| --- | --- |
| 경로 | 모두 `/api/admin/`으로 시작한다 |
| 허용 IP | `.env`의 `KEDP_ADMIN_ALLOW_IPS`(쉼표로 나눈 IP 또는 CIDR). 비어 있으면 관리자 API는 모두 403이다 |
| 요청 주소 | `RemoteAddr`로 본다. `RemoteAddr`가 믿을 proxy(`.env`의 `KEDP_TRUSTED_PROXIES`, 쉼표로 나눈 IP 또는 CIDR) 안에 있을 때만 `X-Forwarded-For`의 마지막 주소를 쓴다. `KEDP_TRUSTED_PROXIES`가 비어 있으면 `127.0.0.1`(같은 서버의 proxy)이다. 개발용 Vite proxy는 `xfwd: true`로 둔다. 배포의 nginx는 `X-Forwarded-For`를 실제 접속 주소로 덮어써서 넘기고, 믿을 proxy에는 Docker 내부 network를 넣는다(`deploy/README.md`) |
| 계정 | `.env`의 `KEDP_ADMIN_FILE`이 가리키는 파일 하나. 한 줄에 `계정:bcrypt hash`. 비밀번호 원문은 어디에도 저장하지 않는다. 한 줄은 `cmd/adminpass`로 만든다(`b-go-end/README.md`). 서버를 켤 때 읽는다 |
| session | 로그인하면 서버 메모리에 session을 두고 cookie로 준다. 서버를 다시 켜면 다시 로그인한다 |
| cookie | 이름 `kedp_admin_session`, 무작위 32바이트, `HttpOnly`, `SameSite=Strict`, `Path=/api/admin`, 8시간(로그인한 때부터). `Secure`는 배포 Issue에서 HTTPS와 함께 켠다 |
| 확인 순서 | 허용 IP가 아니면 403 → 로그인(`POST /api/admin/login`)이 아닌 요청은 로그인하지 않았으면 401 |
| 바꾸는 요청 | `POST`, `PUT`은 `Content-Type: application/json`만 받는다(아니면 415). 본문은 64 KiB까지다 |
| 응답 | JSON, `Cache-Control: no-store`. `POST /api/admin/logout`만 본문이 없다(204) |
| 기록 | 관리자가 바꾼 내용(계정, 무엇을)은 서버 기록에 `admin audit:`로 남는다. 비밀번호와 cookie 값은 남기지 않는다 |

## 5.2 오류

사용자 조회 API의 오류(1.4)와 같은 형식이다. 입력 오류(400)는 `fields`에 칸 이름별 이유가 붙는다.

| 상태 | 언제 | 응답 |
| --- | --- | --- |
| 400 | 본문이 JSON 하나가 아니다 | `{"error":"invalid JSON"}` |
| 400 | 칸이 빠졌거나 틀렸다. 모르는 칸도 오류다 | `{"error":"invalid input","fields":{"cycle":"must be one of D, M, Q, A"}}` |
| 401 | 로그인하지 않았다(cookie가 없거나, 모르거나, 8시간이 지났다) | `{"error":"login required"}` |
| 401 | 로그인 실패. 계정이 틀렸든 비밀번호가 틀렸든 같다 | `{"error":"login failed"}` |
| 403 | 허용 IP가 아니다 | `{"error":"forbidden"}` |
| 404 | 없는 지표 / 없는 주소 | `{"error":"indicator not found"}` / `{"error":"not found"}` |
| 405 | 받지 않는 method. `Allow` 헤더가 붙는다 | `{"error":"method not allowed"}` |
| 409 | 지표 추가: 같은 id가 있다 | `{"error":"indicator already exists"}` |
| 415 | `POST`, `PUT`의 `Content-Type`이 `application/json`이 아니다 | `{"error":"content type must be application/json"}` |
| 500 | 서버 문제. 자세한 내용은 서버 기록에만 남는다 | `{"error":"internal error"}` |

## 5.3 로그인, 로그아웃, 로그인 상태

```text
POST /api/admin/login       {"account": "…", "password": "…"}
POST /api/admin/logout
GET  /api/admin/session
```

- 로그인 성공 200: cookie를 주고 로그인 상태를 돌려준다. 이미 로그인한 cookie가 있으면 그 session은 지운다.
- 로그인 실패 401 `{"error":"login failed"}`. `account`, `password`가 빠지면 400이다.
- 로그아웃 204: session을 지우고 cookie를 지운다(`Max-Age=0`).
- 로그인 상태 200: 로그인했으면 로그인 상태, 아니면 401 `{"error":"login required"}`. 화면은 처음 열 때 이것으로 로그인 화면을 보일지 정한다.

로그인 상태(로그인, 로그인 상태 응답):

```json
{ "account": "admin", "expires_at": "2026-10-30T18:00:00+09:00" }
```

| 칸 | 뜻 |
| --- | --- |
| `account` | 계정 이름 |
| `expires_at` | session이 끝나는 시각(Asia/Seoul). 지나면 다시 로그인한다 |

## 5.4 지표 목록 (Dashboard 관리)

```text
GET /api/admin/indicators
```

숨긴 지표, 화면 설정이 없는 지표까지 모두 준다. 순서는 화면 순서(`order`), 그다음 `id`이고, 화면 설정이 없는 지표는 맨 뒤다.

응답 200:

```json
{
  "indicators": [
    {
      "id": "ecos-base-rate",
      "name": "한국은행 기준금리",
      "category": "money",
      "source": "ecos",
      "stat_code": "722Y001",
      "item_code": "0101000",
      "classes": [],
      "org_id": null,
      "value_field": null,
      "cycle": "M",
      "unit": null,
      "source_unit": "연%",
      "visible": true,
      "order": 1,
      "chart_period": "10 years",
      "decimals": 2,
      "notice": null,
      "related": ["kosis-employment-rate"]
    }
  ]
}
```

> 값은 예시다.

| 칸 | 형식 | 뜻 |
| --- | --- | --- |
| `id` | 문자열 | 지표 id. 화면 주소 `/indicator/{id}`에도 쓴다 |
| `name` | 문자열 | 지표 이름 |
| `category` | 문자열 | 구분 code(5.11). 이름은 5.11 목록에서 찾는다 |
| `source` | 문자열 | 제공처 `ecos`, `kosis`, `data-go-kr` |
| `stat_code` | 문자열 | ECOS 통계 코드, KOSIS 표 id(`tblId`), 공공데이터포털 종목 코드 |
| `item_code` | 문자열 또는 `null` | ECOS 항목 코드, KOSIS `itmId`. 공공데이터포털은 `null` |
| `classes` | 문자열 배열 | 분류 코드. KOSIS `objL1`, `objL2`… 순서 |
| `org_id` | 문자열 또는 `null` | KOSIS 기관 코드(`orgId`) |
| `value_field` | 문자열 또는 `null` | 공공데이터포털에서 값으로 쓰는 항목(`clpr`) |
| `cycle` | 문자열 | 주기 `D`, `M`, `Q`, `A` |
| `unit` | 문자열 또는 `null` | 관리자가 넣은 단위. 있으면 응답 단위보다 먼저 쓴다(1.2) |
| `source_unit` | 문자열 또는 `null` | 응답에 있는 단위(읽기만) |
| `visible` | 참/거짓 | 메인 화면 노출 |
| `order` | 숫자 또는 `null` | 메인 화면 순서. 작은 것이 앞이다. 화면 설정이 없으면 `null` |
| `chart_period` | 문자열 또는 `null` | 차트 기간 `N years`, `N months`, `N days`(예: `1 years`도 이 형식이다). 화면 설정이 없으면 `null` |
| `decimals` | 숫자 또는 `null` | 소수 자릿수 0~4. 없으면 받은 값 그대로 보인다 |
| `notice` | 문자열 또는 `null` | 상세 화면 맨 위 안내 문구 |
| `related` | 문자열 배열 | 관련 지표 id. 상세 화면에 이 순서로 보인다 |

- 화면 설정이 없는 지표(`order`가 `null`)는 지표 수정(5.6)으로 차트 기간을 넣으면 화면 설정이 생긴다(노출 안 함, 맨 뒤).
- 관련 지표 이름은 같은 목록에서 `id`로 찾는다.

## 5.5 지표 추가

```text
POST /api/admin/indicators
```

요청:

```json
{
  "id": "kosis-kospi",
  "name": "KOSPI",
  "category": "market",
  "source": "kosis",
  "stat_code": "DT_XXXX",
  "item_code": "T1",
  "classes": ["00"],
  "org_id": "101",
  "value_field": null,
  "cycle": "M",
  "unit": "pt",
  "visible": false,
  "chart_period": "10 years",
  "decimals": 2,
  "notice": null,
  "related": ["ecos-base-rate"]
}
```

> 코드는 예시다. 확인한 지표 코드가 아니다.

| 칸 | 꼭 | 규칙 |
| --- | --- | --- |
| `id` | 예 | 영문 소문자와 숫자를 `-`로 이은 글(예: `kosis-kospi`). 같은 id가 있으면 409 |
| `name` | 예 | 빈 글자가 아니다. 앞뒤 공백은 뺀다 |
| `category` | 예 | 구분 code(5.11). 앞뒤 공백은 뺀다. 빠지거나 빈 글자면 `required`, 없는 code면 400 `fields.category` = `category "…" does not exist` |
| `source` | 예 | `ecos`, `kosis`, `data-go-kr` |
| `stat_code` | 예 | 빈 글자가 아니다 |
| `item_code` | 제공처마다 | `ecos`, `kosis`는 꼭 넣는다. `data-go-kr`은 비운다 |
| `org_id` | 제공처마다 | `kosis`는 꼭 넣는다 |
| `value_field` | 제공처마다 | `data-go-kr`은 꼭 넣는다(`clpr`) |
| `classes` | 아니요 | 문자열 배열. 빈 코드는 안 된다. 빠지면 `[]` |
| `cycle` | 예 | `D`, `M`, `Q`, `A` |
| `unit` | 아니요 | 빈 글자나 `null`이면 단위 없음 |
| `visible` | 아니요 | 빠지면 `false` |
| `chart_period` | 예 | `N years`, `N months`, `N days`만 받는다(N은 1~9999). 처음 수집할 과거 범위도 이 기간이다(KEDP001-D022 12) |
| `decimals` | 아니요 | 0~4 또는 `null` |
| `notice` | 아니요 | 빈 글자나 `null`이면 없음 |
| `related` | 아니요 | 있는 지표의 id 배열. 자기 자신, 같은 id 두 번은 안 된다. 빠지면 `[]` |

- 규칙은 DB 제약(`b-sql-db/migrations/0001_schema.up.sql`, `0004_category.up.sql`)과 같다. DB에 넣기 전에 모두 검사하고, 틀린 칸을 모두 `fields`에 담아 400으로 돌려준다. 구분 code가 목록에 있는지와 관련 지표가 있는지만 DB에 넣을 때 확인한다(그 칸 하나만 `fields`에 담긴다).
- 제공처에 그 코드가 있는지는 확인하지 않는다. 다음 수집 결과가 API 관리(5.9)에 보인다.
- 순서(`order`)는 맨 뒤다. 지표, 화면 설정, 관련 지표는 한 transaction으로 넣는다.
- 성공 201: 추가한 지표(5.4의 지표 하나).
- 지표 삭제는 없다. 안 쓰는 지표는 노출을 끈다.

## 5.6 지표 수정

```text
PUT /api/admin/indicators/{id}
```

요청(바꿀 수 있는 칸만, 모두 보낸다):

```json
{
  "name": "한국은행 기준금리",
  "category": "money",
  "cycle": "M",
  "unit": null,
  "chart_period": "10 years",
  "decimals": 2,
  "notice": null,
  "related": ["kosis-employment-rate"]
}
```

| 칸 | 꼭 | 규칙 |
| --- | --- | --- |
| `name`, `category`, `cycle`, `chart_period` | 예 | 5.5와 같다 |
| `unit`, `decimals`, `notice` | 아니요 | 5.5와 같다. 빠지거나 `null`이면 지운다 |
| `related` | 예 | 5.5와 같다. 관련 지표를 모두 지우려면 `[]`. 보낸 목록으로 바꾼다 |

- `id`, 제공처, 코드(`stat_code`, `item_code`, `classes`, `org_id`, `value_field`)는 바꾸지 않는다. 보내면 400 `unknown field`다. 바꾸려면 새 지표를 추가한다.
- 노출과 순서는 5.7로 바꾼다.
- 주기를 바꾸면 이전 주기의 값은 DB에 남고 화면에는 새 주기 값만 보인다(1.3, KEDP001-D022 11). 새 주기 값이 없으므로 다음 수집은 차트 기간만큼 처음부터 받는다.
- 화면 설정이 없던 지표는 화면 설정이 생긴다(노출 안 함, 맨 뒤).
- 성공 200: 고친 지표(5.4의 지표 하나). 없는 id는 404.

## 5.7 노출과 순서 저장

```text
PUT /api/admin/display-order
```

Dashboard 관리 화면의 [저장]이 부른다. 화면 설정이 있는 지표(`order`가 `null`이 아닌 지표) 전체를 보낸다.

```json
{
  "indicators": [
    { "id": "ecos-base-rate", "visible": true, "order": 1 },
    { "id": "kosis-employment-rate", "visible": false, "order": 2 }
  ]
}
```

- 칸 `id`, `visible`, `order`(정수)는 모두 꼭 넣는다.
- 한 transaction으로 저장한다. 빠진 지표, 없는 지표, 같은 지표 두 번이 있으면 400이고 아무것도 바꾸지 않는다.
- 성공 200: 저장한 뒤의 지표 목록(5.4 응답과 같다).

## 5.8 수집 시각 (API 관리)

```text
GET /api/admin/collect-time
PUT /api/admin/collect-time     {"collect_time": "07:30"}
```

응답 200: `{"collect_time": "06:00"}`

- 하루 1회 수집 시각(Asia/Seoul, KEDP001-D019). `HH:MM`(00:00~23:59)만 받는다(`6:00`은 안 된다).
- 수집 서버가 30초마다 다시 읽으므로 서버를 다시 켤 필요가 없다. 바꾼 뒤 처음 오는 그 시각부터 따른다.

## 5.9 제공처 상태 (API 관리)

```text
GET /api/admin/sources
```

응답 200:

```json
{
  "sources": [
    { "code": "data-go-kr", "name": "공공데이터포털", "key_set": true, "last_collected_at": null, "status": "unchecked" },
    { "code": "ecos", "name": "ECOS", "key_set": true, "last_collected_at": "2026-10-25T06:00:03+09:00", "status": "connected" },
    { "code": "kosis", "name": "KOSIS", "key_set": false, "last_collected_at": "2026-10-25T06:00:05+09:00", "status": "no_key" }
  ]
}
```

> 값은 예시다.

| 칸 | 뜻 |
| --- | --- |
| `code`, `name` | 제공처 코드와 화면 이름. 순서는 `code` 순서다 |
| `key_set` | `.env`에 키가 있는지(`true`/`false`). 키 값은 일부도 주지 않는다(KEDP001-D012) |
| `last_collected_at` | 그 제공처의 마지막으로 끝난 수집 기록(`collection_run`)의 끝난 시각. 결과(성공, 데이터 없음, 오류)와 관계없다. 없으면 `null` |
| `status` | 아래 표 |

| `status` | 화면 | 언제 |
| --- | --- | --- |
| `connected` | 연결 | 마지막으로 끝난 수집이 `success` 또는 `no_data` |
| `error` | 오류 | 마지막으로 끝난 수집이 `error` |
| `unchecked` | 확인 전 | 끝난 수집 기록이 없다 |
| `no_key` | 연결 안 됨 | `.env`에 키가 없다(수집 기록보다 먼저 본다) |

- 실제 연결 시험은 하지 않는다(`specification/50.web/README.md` 6장 향후 확장).
- 키 설정 여부는 서버를 켤 때 읽은 값이다. `.env`의 키를 바꾸면 서버를 다시 켠다.

## 5.10 확인 명령

수집 서버를 띄운 뒤(`b-go-end/README.md`), 허용 IP에 든 곳에서 실행한다. 같은 서버에서 실행하면 요청 주소는 `127.0.0.1`이다. cookie는 파일(`$J`)에 두고, 확인이 끝나면 지운다.

```sh
A=http://127.0.0.1:8080/api/admin
J=$(mktemp)                                   # cookie 파일
H='Content-Type: application/json'

# 로그인하지 않으면 401, 허용 IP가 아니면 403
curl -s -i $A/session

# 로그인: 비밀번호는 화면과 명령 기록에 남지 않게 읽는다
read -r -p 'account: ' ADMIN_ACCOUNT; read -r -s -p 'password: ' ADMIN_PASSWORD; echo
printf '{"account":"%s","password":"%s"}' "$ADMIN_ACCOUNT" "$ADMIN_PASSWORD" |
  curl -s -c $J -H "$H" --data-binary @- $A/login
unset ADMIN_PASSWORD

curl -s -b $J $A/session
curl -s -b $J $A/indicators
curl -s -b $J $A/categories
curl -s -b $J $A/collect-time
curl -s -b $J $A/sources

# 입력 오류(400)
curl -s -b $J -X PUT -H "$H" -d '{"collect_time":"6:00"}' $A/collect-time

curl -s -b $J -X POST $A/logout -H "$H" -i
curl -s -b $J -i $A/session                    # 401
rm -f $J
```

- 비밀번호에 `"`나 `\`가 있으면 위 `printf`로는 JSON이 깨진다. 그때는 `jq -n --arg a "$ADMIN_ACCOUNT" --arg p "$ADMIN_PASSWORD" '{account:$a,password:$p}'`로 만든다.

## 5.11 지표 구분 (KEDP001-D024, Issue #52)

```text
GET /api/admin/categories
```

지표 추가·수정 화면에서 구분을 고를 때 쓴다. 순서는 `order`(작은 것이 앞)다.

응답 200:

```json
{
  "categories": [
    { "code": "growth", "name": "성장·경기", "order": 1 },
    { "code": "price", "name": "물가", "order": 2 }
  ]
}
```

> 줄였다. 전체 목록은 아래 표다.

| `code` | `name` | `order` |
| --- | --- | --- |
| `growth` | 성장·경기 | 1 |
| `price` | 물가 | 2 |
| `labor` | 고용·소득 | 3 |
| `money` | 금리·통화 | 4 |
| `credit` | 부채·금융안정 | 5 |
| `external` | 대외경제 | 6 |
| `housing` | 주택 | 7 |
| `population` | 인구 | 8 |
| `market` | 금융시장 | 9 |

- 구분 목록은 DB(`category`)에 있고 migration으로만 바꾼다. 관리자 API로 추가·수정·삭제하지 않는다.
- 지표마다 구분이 꼭 하나 있다(`indicator.category_code`). 지표의 구분은 5.5, 5.6으로 정한다.
- 노출과 순서 저장(5.7)은 바뀌지 않는다. 순서(`order`)는 구분과 상관없는 전체 순서이고, 공개 화면(2장)은 구분 순서로 먼저 묶은 뒤 이 순서를 쓴다.

---

# 6. 방문자 수 API (Issue #48)

공개 화면 아래쪽에 오늘 방문자 수와 전체 방문자 수를 보여주고, 관리자 화면에서 숫자를 관리한다. 구현은 `b-go-end/internal/api/visits.go`, `b-go-end/internal/admin/visits.go`, `b-go-end/internal/store/visit.go`, DB는 `specification/30.database/README.md` 4.2다.

## 6.1 세는 기준

| 항목 | 내용 |
| --- | --- |
| 방문자 | 한 사람(브라우저)이 하루에 여러 번 와도 1명 |
| 하루 | Asia/Seoul 00:00~24:00 |
| 화면이 할 일 | 처음 열릴 때 `POST /api/visits`를 한 번 부른다. 그날 이미 보냈으면 다시 보내지 않는다(브라우저에 날짜를 둔다) |
| 같은 사람 | Go가 접속 주소 + User-Agent + 그날의 salt로 hash(HMAC-SHA256)를 만들어, 그날 본 hash면 세지 않는다 |
| 접속 주소 | 관리자 허용 IP와 같은 방법이다(5.1 "요청 주소"): 믿을 proxy(`KEDP_TRUSTED_PROXIES`)에서 온 요청만 `X-Forwarded-For`의 마지막 주소를 쓴다. port는 보지 않는다 |
| salt | 날마다 새로 만드는 무작위 32바이트. Go 메모리에만 두고, 날이 바뀌면 그날 hash 목록과 함께 버린다 |
| 남기지 않는 것 | 접속 주소, User-Agent, hash는 DB, 파일, 서버 기록 어디에도 남기지 않는다. DB에는 날짜별 숫자만 있다 |
| 받아들인 한계 | Go를 다시 켜면 그날 hash 목록이 사라져 같은 사람을 한 번 더 셀 수 있다. 같은 주소(같은 회사, 같은 VPN)에 User-Agent까지 같으면 1명으로 센다 |
| 하루 상한 | 하루에 기억하는 사람은 200,000명까지다(메모리 보호). 넘으면 그날은 더 세지 않고, 서버 기록에 한 번 남긴다(주소 등은 넣지 않는다) |

## 6.2 방문 하나 세기

```text
POST /api/visits
```

- 인증 없음. 본문은 없다(보내도 읽지 않는다). `Content-Type`은 보지 않는다.
- 응답 204(본문 없음), `Cache-Control: no-store`. 그날 이미 센 사람이어도, 접속 주소를 알 수 없어도(믿을 proxy가 읽을 수 없는 주소를 넘김), 하루 상한을 넘어도 세지 않고 똑같이 204다.

## 6.3 오늘과 전체

```text
GET /api/visits/summary
```

- 인증 없음. 공개 화면 아래쪽 표시에 쓴다. 표시를 켜고 끄는 설정은 없다(항상 표시).
- 응답 200, `Cache-Control: no-store`:

```json
{ "today": 12, "total": 3456 }
```

| 칸 | 뜻 |
| --- | --- |
| `today` | 오늘(Asia/Seoul) 방문자 수. 기록이 없으면 0 |
| `total` | 전체 방문자 수 = 시작값(6.4 `base`) + 일별 방문자 수 합(오늘 포함) |

## 6.4 관리자: 방문자 수 보기

```text
GET /api/admin/visits
```

허용 IP + 로그인(5.1). 응답 200:

```json
{
  "today": 12,
  "total": 3456,
  "base": 3000,
  "days": [
    { "day": "2026-10-03", "count": 0 },
    { "day": "2026-10-04", "count": 15 },
    { "day": "2026-11-01", "count": 12 }
  ]
}
```

> 값은 예시다. `days`는 실제로 30개다.

| 칸 | 뜻 |
| --- | --- |
| `today`, `total` | 6.3과 같다 |
| `base` | 전체 방문자 수의 시작값 |
| `days` | 오늘을 포함한 최근 30일의 일별 방문자 수. 오래된 날부터, 기록이 없는 날은 `count` 0으로 채워 늘 30개다. `day`는 `YYYY-MM-DD`(Asia/Seoul) |

## 6.5 관리자: 시작값 바꾸기

```text
PUT /api/admin/visits/base     {"base": 3000}
```

- 예전 숫자를 이어 쓸 때 쓴다. `base`는 0 이상 9007199254740991 이하 정수(화면이 정확히 읽을 수 있는 가장 큰 정수)다. 빠졌거나, `null`, 음수, 소수, 문자열, 범위 밖이면 400 `{"error":"invalid input","fields":{"base":"must be an integer from 0 to 9007199254740991"}}`이고 바꾸지 않는다.
- 성공 200: 바꾼 뒤의 방문자 수(6.4 응답과 같다).
- 서버 기록: `admin audit: account "…": visit_total_base <이전> -> <새 값>`

## 6.6 관리자: 초기화

```text
POST /api/admin/visits/reset
```

- 시험 기간 숫자를 지울 때 쓴다. 일별 기록을 모두 지우고 시작값을 0으로 한다(한 transaction). 되돌릴 수 없다.
- `Content-Type: application/json`이어야 한다(5.1). 본문은 읽지 않는다(`{}`를 보낸다).
- 성공 200: 초기화한 뒤의 방문자 수(6.4 응답과 같다. 모두 0).
- Go 메모리의 그날 hash 목록은 그대로다. 오늘 이미 센 사람은 초기화한 뒤에도 오늘은 다시 세지 않는다.
- 서버 기록: `admin audit: account "…": reset visits: deleted <날 수> days (<방문자 수 합> visits), visit_total_base <이전> -> 0`

## 6.7 오류

| 상태 | 언제 | 응답 |
| --- | --- | --- |
| 400 | 관리자 시작값 입력 오류(6.5), 본문이 JSON이 아님, 모르는 칸 | 5.2와 같다 |
| 401, 403, 415 | 관리자 API(6.4~6.6). 5.2와 같다 | 5.2와 같다 |
| 404 | 없는 주소(예: `/api/visits/none`) | `{"error":"not found"}` |
| 405 | `/api/visits`에 `POST`가 아닌 method(`Allow: POST`), `/api/visits/summary`에 `GET`, `HEAD`가 아닌 method(`Allow: GET, HEAD`), 관리자 API는 5.2와 같다 | `{"error":"method not allowed"}` |
| 500 | DB 오류. 자세한 내용은 서버 기록에만 남는다(주소, User-Agent는 넣지 않는다). `POST /api/visits`가 500이면 그 사람은 세지 않은 것으로 보고, 다음 요청에서 다시 센다 | `{"error":"internal error"}` |

## 6.8 확인 명령

서버를 띄운 뒤(`b-go-end/README.md`):

```sh
V=http://127.0.0.1:8080/api/visits
curl -s -i -X POST $V                           # 204
curl -s -i -X POST $V                           # 204(같은 사람이라 세지 않는다)
curl -s -X POST -A 'other-browser' $V           # 다른 User-Agent는 다른 사람이다
curl -s $V/summary                              # {"today":2,"total":…}
curl -s -i $V                                   # 405
```

관리자(5.10처럼 로그인한 뒤, cookie 파일 `$J`):

```sh
A=http://127.0.0.1:8080/api/admin
H='Content-Type: application/json'
curl -s -b $J $A/visits
curl -s -b $J -X PUT -H "$H" -d '{"base":3000}' $A/visits/base
curl -s -b $J -X PUT -H "$H" -d '{"base":-1}' $A/visits/base     # 400
curl -s -b $J -X POST -H "$H" -d '{}' $A/visits/reset             # 숫자가 모두 지워진다. 시험할 때만
```
