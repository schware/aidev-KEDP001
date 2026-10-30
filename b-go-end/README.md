# b-go-end

Go 수집 모듈. 자세한 구조와 결정은 `docs/dev-complete/b-go-end.md`를 본다.

| 명령 | 하는 일 |
| --- | --- |
| `cmd/server` | DB `indicator`의 지표를 수집 시각마다 수집하고, 응답을 Rust 처리 서버로 넘긴다. 사용자 조회 API와 관리자 API도 연다 |
| `cmd/adminpass` | 관리자 계정 파일(`KEDP_ADMIN_FILE`)에 넣을 한 줄을 만든다 |
| `cmd/seed` | `config/indicators.yaml`의 지표 정의와 화면 처음 값을 DB에 넣는다 |
| `cmd/sample` | 요청 값을 바로 넣어 부르고, 받은 응답을 그대로 저장한다(조사용 샘플) |
| `cmd/fakesource` | 시험용 제공처 서버. `../samples/`의 응답을 돌려준다 |

명령은 모두 `b-go-end` 폴더에서 실행한다. 설정은 `../.env`에서 읽는다(`-env`로 바꿀 수 있다). 같은 이름의 환경 변수가 있으면 환경 변수를 쓴다.

| 변수 | 뜻 | 쓰는 명령 |
| --- | --- | --- |
| `ECOS_API_KEY`, `KOSIS_API_KEY`, `DATA_GO_KR_API_KEY` | 제공처 키 | `server`, `sample` |
| `KEDP_DB_HOST`, `KEDP_DB_PORT`, `KEDP_DB_NAME` | DB 접속 정보(`b-sql-db/README.md`) | `server`, `seed` |
| `kedp_go` | `kedp_go` 계정 비밀번호(KEDP001-D022 14) | `server`, `seed` |
| `KEDP_PROCESSOR_ADDR` | Rust 처리 서버 주소(IP:포트). 비우면 `127.0.0.1:50051` | `server` |
| `KEDP_API_ADDR` | 조회 API 주소(IP:포트). 비우면 `127.0.0.1:8080`. 관리자 API도 같은 주소다 | `server` |
| `KEDP_ADMIN_ALLOW_IPS` | 관리자 API를 쓸 수 있는 IP 또는 CIDR, 쉼표로 나눈다. 비우면 관리자 API는 모두 403 | `server` |
| `KEDP_TRUSTED_PROXIES` | `X-Forwarded-For`를 믿을 proxy의 IP 또는 CIDR, 쉼표로 나눈다. 비우면 `127.0.0.1`(개발용 Vite proxy) | `server` |
| `KEDP_ADMIN_FILE` | 관리자 계정 파일 위치(절대 경로). 비우면 로그인할 수 없다 | `server` |

- 오류와 기록에는 변수 이름만 나오고 값(키, 접속 정보, 비밀번호, Rust 주소, API 주소, 허용 IP, 믿을 proxy, 계정 파일 위치)은 나오지 않는다.

## 지표 넣기 (`cmd/seed`)

```sh
go run ./cmd/seed
```

- `config/indicators.yaml`의 지표를 `indicator`에, `display`를 `indicator_display`에 넣는다. 한 transaction이라 하나라도 실패하면 아무것도 넣지 않는다.
- 이미 있는 지표(같은 `id`)는 고치지 않는다. 다시 실행해도 결과가 같고, 관리자 화면에서 바꾼 값을 덮어쓰지 않는다.
- 결과: `indicator: added 7, kept 0` (다시 실행하면 `added 0, kept 7`)
- 수집은 DB `indicator`만 읽는다. DB에 넣은 뒤에는 `indicators.yaml`을 고쳐도 수집에 반영되지 않는다.

## 수집 서버 (`cmd/server`)

```sh
go run ./cmd/server          # 조회 API를 열고, 수집 시각마다 수집한다(collection_run.trigger = schedule)
go run ./cmd/server -once    # 지금 한 번 수집하고 끝낸다(trigger = manual). 조회 API는 열지 않는다
```

- 시작할 때 Rust `Check`를 부르고, 실패하면 멈춘다(KEDP001-D020). Rust 서버를 먼저 띄운다(`b-rust-processor/README.md`).
- 수집 시각은 `app_setting`의 `collect_time`(HH:MM, Asia/Seoul)이다. 30초마다 다시 읽으므로 바뀌면 다음 수집부터 따른다(KEDP001-D019). 서버를 켰을 때 오늘 수집 시각이 이미 지났으면 다음 날부터 수집한다.
- `Ctrl+C` 또는 SIGTERM을 받으면 멈춘다.
- 실행 결과는 `collection_run`에 지표마다 한 행으로 남는다.

지표 하나의 수집 순서(`internal/collect`, `b-proto/README.md` 흐름):

1. `collection_run`에 `running`으로 넣는다.
2. 수집 범위를 정한다. 관측값이 없으면 오늘에서 차트 기간(`indicator_display.chart_period`)을 뺀 날부터(KEDP001-D022 12), 있으면 마지막 관측값의 기간부터 오늘까지다. 마지막 기간을 다시 받는 것은 값이 고쳐졌을 수 있어서다.
3. 제공처에 요청하고 받은 응답을 Rust `ProcessPage`로 넘긴다. 오류·데이터 없음 응답도 넘긴다(원본 응답은 Rust가 남긴다). 전체 행 수가 1000행을 넘으면 페이지마다 요청하고 넘긴다(ECOS, 공공데이터포털. KEDP001-D022 21).
4. Rust 결과로 `collection_run`을 고친다. `row_count`는 모든 페이지의 (새로 넣음 + 덮어씀 + 그대로) 행 수다. 빈 값으로 건너뛴 행은 세지 않는다.

- 지표 하나가 실패해도 다음 지표를 수집한다.
- Rust가 결과를 주지 못하면(연결 실패, gRPC 오류) 그 지표를 `error`로 남기고 남은 지표는 수집하지 않는다(KEDP001-D020). 예약 실행이면 다음 수집 시각을 기다린다.

`collection_run.error_code`: 제공처 코드(`INFO-100`, `11`, `30` 등)와 Rust 코드(`HTTP_<상태 코드>`, `INVALID_RESPONSE`)는 `b-proto/README.md`를 본다. Rust에 넘기지 못했을 때는 Go가 아래 코드를 쓴다.

| 코드 | 뜻 |
| --- | --- |
| `INVALID_INDICATOR` | 수집 범위를 정할 수 없다(화면 설정 없음, 제공처가 받지 않는 주기) |
| `NO_API_KEY` | `.env`에 키가 없어 요청하지 않았다 |
| `FETCH_FAILED` | 응답을 받지 못했다(시간 초과, 연결 실패) |
| `RESPONSE_HAS_KEY` | 응답 본문에 키가 들어 있어 Rust에 넘기지 않았다 |
| `PROCESSOR_FAILED` | Rust에 연결하지 못했거나 Rust가 gRPC 오류를 돌려줬다 |

## 조회 API (`internal/api`)

사용자 화면이 읽는 API다. 주소, 요청, 응답, 오류는 `specification/40.api/README.md`를 본다.

| 주소 | 하는 일 |
| --- | --- |
| `GET /api/indicators` | 메인 화면 카드 목록 |
| `GET /api/indicators/{id}` | 지표 상세 |

- `cmd/server`(`-once` 없이)가 수집과 함께 연다. API 주소를 열지 못하면(이미 쓰는 포트 등) 시작하지 않는다. API가 멈추면 수집도 멈춘다.
- DB는 `kedp_go` 계정으로 읽기만 한다(`internal/store/read.go`).
- CORS는 두지 않는다. 화면 개발은 Vite proxy로 연결한다.
- DB 오류는 서버 기록에 남기고, 응답은 `{"error":"internal error"}`(500)만 준다.

### 시험 DB로 확인

`internal/store/read_test.go`는 `KEDP_TEST_DB_NAME`이 있을 때만 돈다. 시험 DB에 migration과 `internal/store/testdata/read.sql`을 넣고 `kedp_go`로 읽는다. `kedp`에는 넣지 않는다.

```sh
cd /srv/projects/aidev-kedp001
P="docker exec -i kedp-postgres psql -X -q -v ON_ERROR_STOP=1 -U sunmoon -d kedp_api_check"
docker exec kedp-postgres createdb -U sunmoon kedp_api_check
$P < b-sql-db/migrations/0001_schema.up.sql
$P < b-sql-db/migrations/0002_roles.up.sql
$P < b-go-end/internal/store/testdata/read.sql
(cd b-go-end && KEDP_TEST_DB_NAME=kedp_api_check go test -count=1 ./internal/store)
docker exec kedp-postgres dropdb -U sunmoon kedp_api_check
```

관리자 API의 DB 일(`internal/store/admin_test.go`)은 `KEDP_ADMIN_TEST_DB_NAME`이 있을 때만 돈다. 시험이 값을 바꾸므로 돌릴 때마다 DB를 새로 만든다.

```sh
cd /srv/projects/aidev-kedp001
P="docker exec -i kedp-postgres psql -X -q -v ON_ERROR_STOP=1 -U sunmoon -d kedp_admin_check"
docker exec kedp-postgres createdb -U sunmoon kedp_admin_check
$P < b-sql-db/migrations/0001_schema.up.sql
$P < b-sql-db/migrations/0002_roles.up.sql
$P < b-go-end/internal/store/testdata/admin.sql
(cd b-go-end && KEDP_ADMIN_TEST_DB_NAME=kedp_admin_check go test -count=1 ./internal/store)
docker exec kedp-postgres dropdb -U sunmoon kedp_admin_check
```

## 관리자 API (`internal/admin`)

관리자 화면이 쓰는 API다. 주소, 요청, 응답, 오류, 확인 명령은 `specification/40.api/README.md` 5장을 본다.

| 주소 | 하는 일 |
| --- | --- |
| `POST /api/admin/login`, `POST /api/admin/logout`, `GET /api/admin/session` | 로그인, 로그아웃, 로그인 상태 |
| `GET /api/admin/indicators`, `POST /api/admin/indicators` | 전체 지표 목록, 지표 추가 |
| `PUT /api/admin/indicators/{id}` | 지표 수정 |
| `PUT /api/admin/display-order` | 노출과 순서 저장(전체 목록, 한 transaction) |
| `GET /api/admin/collect-time`, `PUT /api/admin/collect-time` | 수집 시각 |
| `GET /api/admin/sources` | 제공처별 키 설정 여부, 마지막 수집, 상태 |

- 허용 IP(`KEDP_ADMIN_ALLOW_IPS`)가 아니면 403, 로그인하지 않았으면 401이다(KEDP001-D018).
- 요청 주소는 `RemoteAddr`이고, `RemoteAddr`가 믿을 proxy(`KEDP_TRUSTED_PROXIES`, 비우면 `127.0.0.1`) 안에 있을 때만 `X-Forwarded-For`의 마지막 주소를 쓴다. 배포(`deploy/`)에서는 nginx가 있는 Docker 내부 network를 넣는다.
- session은 서버 메모리에 있다. 서버를 다시 켜면 다시 로그인한다.
- 계정 파일은 서버를 켤 때 읽는다. 파일을 바꾸면 서버를 다시 켠다. `KEDP_ADMIN_FILE`이 있는데 파일을 읽지 못하거나 형식이 틀리면 서버가 시작하지 않는다.
- 관리자가 바꾼 내용은 서버 기록에 `admin audit: account "…": …`로 남는다. 비밀번호와 cookie 값은 남기지 않는다.
- 지표 삭제, API 키 변경, 재수집, 연결 시험 API는 없다.

### 계정 파일 만들기 (`cmd/adminpass`)

```sh
go run ./cmd/adminpass -account <계정> > <계정 파일>    # 비밀번호를 두 번 입력한다(화면에 보이지 않는다)
chmod 600 <계정 파일>
```

- 비밀번호는 12자 이상, 72바이트 이하다. bcrypt(cost 12) hash만 나온다. 비밀번호 원문은 어디에도 저장하지 않는다.
- 출력은 파일에 넣을 한 줄 `계정:bcrypt hash`뿐이다(안내 글은 화면에만 나온다). 계정이 여럿이면 줄을 더한다.
- 터미널에서만 돈다(입력을 화면에 보이지 않게 읽어야 해서).
- 계정 파일은 저장소 밖에 둔다(예: `~/.config/kedp/admin`). Git에 올리지 않는다.

## Rust gRPC 코드 (`internal/kedpv1`)

`b-proto/kedp/v1/processor.proto`로 만든 코드가 `internal/kedpv1/`(`processor.pb.go`, `processor_grpc.pb.go`)에 있다. 손으로 고치지 않는다. proto가 바뀌면 다시 만든다.

```sh
go generate ./internal/kedpv1
```

- 실제 명령은 `internal/kedpv1/generate.go`에 있다. proto에 `go_package`가 없으므로 `--go_opt=M`, `--go-grpc_opt=M`으로 Go 패키지를 준다(`b-proto/README.md`).
- `protoc`, `protoc-gen-go`, `protoc-gen-go-grpc`가 필요하다.

## 전체 흐름 확인 (`cmd/fakesource`)

실제 API와 `kedp` DB 없이 수집 → Rust → DB를 확인한다. 시험 DB를 만들고, 가짜 키와 시험용 제공처 서버를 쓴다.

```sh
go run ./cmd/fakesource                 # 127.0.0.1:18080
KEDP_DB_NAME=<시험 DB> ECOS_API_KEY=fake KOSIS_API_KEY=fake DATA_GO_KR_API_KEY=fake \
  go run ./cmd/server -once -source-host http://127.0.0.1:18080
```

- `-source-host`를 주면 요청 주소의 scheme과 host만 바꿔 보낸다. 경로와 질의는 그대로다.
- `fakesource`는 요청의 통계 코드(ECOS는 주기도)로 `../samples/<제공처>/`의 샘플을 고른다. 요청 기간은 보지 않는다. 맞는 샘플이 없으면 404를 돌려준다(Rust 결과 `HTTP_404`).
- Rust 서버도 같은 `KEDP_DB_NAME`으로 띄운다.

## 샘플 수집 도구 (`cmd/sample`)

- `indicators.yaml`에 없는 표·항목·분류·주기·기간도 그대로 넣어 부른다.
- 정상, 오류, 데이터 없음 응답을 모두 그대로 저장한다. 응답을 받지 못했을 때(키 없음, 시간 초과)만 실패다.
- 응답마다 `<이름>.request.json`에 요청 정보(키를 `***`로 가린 주소, HTTP 상태, 받은 시각)를 함께 저장한다.
- 저장 폴더 `-out`은 꼭 넣는다(기본값 없음). 그 아래에 제공처 폴더(`ecos/`, `kosis/`, `data-go-kr/`)를 만든다.
- 이미 있는 파일은 덮어쓰지 않는다. 같은 요청을 다시 받으려면 `-out` 폴더나 `-name`을 바꾼다.
- 조사 샘플은 `../samples/repository/` 아래에 둔다(git에 올리지 않는다).
- 목록 파일의 요청은 부르기 전에 모두 확인한다. 하나라도 틀리면 아무것도 부르지 않는다.

### 요청 종류

| 제공처 `-source` | 요청 종류 `-api` | 꼭 넣는 값 | 기본 파일 이름 |
| --- | --- | --- | --- |
| `ecos` | `search` (기본, StatisticSearch) | `-stat -item -cycle -start -end` | `통계코드-항목-분류…-주기-시작-끝.json` |
| `ecos` | `items` (StatisticItemList) | `-stat` | `통계코드-items.json` |
| `kosis` | `data` (기본, statisticsParameterData) | `-stat -item -cycle -start -end` | `tblId-itmId-objL…-주기-시작-끝.json` |
| `kosis` | `meta` (getMeta) | `-stat -meta-type` | `tblId-meta-type.json` |
| `data-go-kr` | `price` (기본, getETFPriceInfo_V2) | `-stat -start -end` (YYYYMMDD) | `종목코드-D-시작-끝.json` |

- `-stat`: ECOS 통계 코드, KOSIS `tblId`, 공공데이터포털 종목 코드
- `-item`: ECOS `ITEM_CODE1`, KOSIS `itmId`
- `-classes`: 쉼표로 나눈 분류 코드. KOSIS `objL1,objL2…`, ECOS `ITEM_CODE2,ITEM_CODE3…`
- `-cycle`: 제공처 요청 값 그대로(ECOS `A/Q/M/D`, KOSIS `Y/Q/M`). 공공데이터포털은 넣지 않는다.
- `-meta-type`: KOSIS getMeta `type` 값 그대로(예: `ITM`, `TBL`, `PRD`)
- `-name`: 저장 파일 이름(폴더 없이). `.json`이 없으면 붙인다.
- KOSIS `orgId`는 `101`(통계청)로 고정이다.

### 예시: 요청 하나

```sh
# ECOS 통계표의 항목 목록
go run ./cmd/sample -out ../samples/repository/check -source ecos -api items -stat 722Y001

# KOSIS 분류가 2개인 표, 분기
go run ./cmd/sample -out ../samples/repository/check \
  -source kosis -stat DT_XXXX -item T10 -classes 00,1 -cycle Q -start 2024 -end 2025

# KOSIS 메타(항목·분류)
go run ./cmd/sample -out ../samples/repository/check -source kosis -api meta -stat DT_1DA7001S -meta-type ITM
```

저장 결과:

```text
../samples/repository/check/ecos/722Y001-items.json           응답 본문 그대로
../samples/repository/check/ecos/722Y001-items.request.json   요청 정보
```

```json
{
  "source": "ecos",
  "api": "items",
  "url": "https://ecos.bok.or.kr/api/StatisticItemList/***/json/kr/1/1000/722Y001",
  "status": 200,
  "received_at": "2026-10-25T16:00:00+09:00"
}
```

### 예시: 목록 파일로 여러 요청

목록 파일에는 키를 적지 않는다. 모르는 칸이 있으면 멈춘다.

```yaml
# requests.yaml
requests:
  - source: ecos
    stat: 722Y001
    item: "0101000"
    cycle: M
    start: "190001"
    end: "190003"
    name: nodata-722Y001   # 데이터 없음 응답 확인
  - source: kosis
    api: meta
    stat: DT_1DA7001S
    meta_type: ITM
  - source: data-go-kr
    stat: "069500"
    start: "20261020"
    end: "20261022"
```

```sh
go run ./cmd/sample -out ../samples/repository/check -list requests.yaml
```

## 테스트

실제 API, DB, Rust 서버를 쓰지 않는다(시험 DB 시험은 변수가 있을 때만 돈다). `../samples/`의 받아 둔 응답과 시험용 HTTP 서버(httptest)만 쓴다.

```sh
go test ./...
```
