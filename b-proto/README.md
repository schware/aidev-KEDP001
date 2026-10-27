# b-proto

Go(수집)와 Rust(처리 서버) 사이의 gRPC 약속서다. Go가 부르고 Rust가 받는다(KEDP001-D002).

- 변경은 Agent가 변경안만 내고 소유자가 승인한 뒤 한다(KEDP001-D007). 이 첫 버전은 Issue #30의 PR이 변경안이고, 소유자의 merge가 승인이다.
- DB 테이블과 칼럼은 `b-sql-db/migrations/`, 저장 규칙은 `specification/30.database/README.md`(KEDP001-D022)를 따른다.

```text
b-proto/
└── kedp/v1/processor.proto    package kedp.v1, service ProcessorService (Check, ProcessPage)
```

## 흐름

```text
Go                                         Rust                                DB
──                                         ────                                ──
1. Check ────────────────────────────────▶ DB 접속 확인
   실패하면 멈춘다
2. 수집 기록을 running으로 넣는다 ──────────────────────────────────────────▶ collection_run
3. 제공처 응답을 받을 때마다(페이지마다)
   ProcessPage ──────────────────────────▶ 응답 읽기, 기간 변환,               raw_response
                                           빈 값 건너뛰기                      observation (observation_history는 트리거)
               ◀────────────────────────── 결과, 행 수, 오류
4. 결과로 수집 기록을 고친다 ───────────────────────────────────────────────▶ collection_run
```

- collection_run은 Go가 쓴다(KEDP001-D022 7). Rust가 쓰는 `raw_response.run_id`, `observation.run_id`가 `collection_run.id`를 참조하므로, Go는 Rust를 부르기 전에 행을 만들고(2) 결과를 받은 뒤 고친다(4).
- 응답이 여러 페이지면 페이지마다 부르고, 같은 수집은 같은 `run_id`를 쓴다(KEDP001-D022 21).

## Check

상태 확인. Go는 시작할 때 부르고, 실패하면 멈춘다(KEDP001-D002 영향, KEDP001-D020 결론 1).

| 메시지 | 필드 | 뜻 |
| --- | --- | --- |
| `CheckRequest` | 없음 | |
| `CheckResponse` | 없음 | gRPC 상태가 OK이면 정상이다 |

- Rust가 DB에 접속할 수 있을 때만 OK다. 접속할 수 없으면 `UNAVAILABLE`을 돌려준다. Rust에 연결하지 못한 경우도 실패다.
- DB에 아무것도 쓰지 않는다. 상태 확인 기록은 저장하지 않는다(설계서 2.2, KEDP001-D020).

## ProcessPage

제공처 응답 하나(한 페이지)를 처리한다. Rust가 할 일: 응답 읽기, 기간 변환(설계서 5.1), 빈 값 건너뛰기(KEDP001-D022 15), `observation`·`raw_response`에 쓰기(`observation_history`는 `observation`의 트리거가 쓴다).

### 요청 `ProcessPageRequest`

키 값과 키가 들어간 주소는 넣지 않는다(`.claude/api.md`).

| 번호 | 필드 | 타입 | 뜻 | DB | 근거 |
| --- | --- | --- | --- | --- | --- |
| 1 | `run_id` | int64 | 수집 기록 id. 같은 수집의 페이지는 같은 값이다 | `raw_response.run_id`, `observation.run_id`, `observation_history.run_id` | D022 7 |
| 2 | `indicator_id` | string | 지표 id. Rust는 이 id로 `indicator`에서 제공처, 코드, 주기, `value_field`를 읽는다 | `observation.indicator_id` | D022 1, 7 |
| 3 | `page` | int32 | 페이지 번호. 1부터 센다. 오류 메시지에 쓴다 | 저장하지 않는다(요청 주소에 들어 있다) | D022 21 |
| 4 | `received_at` | google.protobuf.Timestamp | 응답을 받은 시각 | `raw_response.received_at`, `observation.collected_at` | D022 5 |
| 5 | `http_status` | int32 | HTTP 상태 코드 | `raw_response.http_status` | D022 5, 13 |
| 6 | `request_masked` | string | 키 자리를 `***`로 가린 요청 주소 | `raw_response.request_masked` | D022 5 |
| 7 | `body` | bytes | 응답 원문. 받은 바이트 그대로다 | JSON이면 `raw_response.body`(jsonb), 아니면 `raw_response.body_text` | D022 5, 13 |

- Go `fetch.Exchange`의 `URL`, `Status`, `ReceivedAt`, `Body`가 각각 `request_masked`, `http_status`, `received_at`, `body`다.
- 오류·데이터 없음 응답도 넘긴다. 원본 응답을 남겨야 한다(D022 13).
- 응답을 받지 못한 요청(시간 초과, 연결 실패)은 넘길 원문이 없으므로 부르지 않는다.
- `body`를 `string`이 아닌 `bytes`로 둔 이유: proto의 `string`은 UTF-8이 아니면 보낼 수 없다. Rust가 받아서 JSON인지 보고 `body`와 `body_text` 중 하나에 넣는다.
- `body`에도 키가 있으면 안 된다. Go는 넘기기 전에 `fetch.Client.Save`와 같은 검사를 한다.

### 응답 `ProcessPageResponse`

| 번호 | 필드 | 타입 | 뜻 | 근거 |
| --- | --- | --- | --- | --- |
| 1 | `status` | ProcessStatus | 처리 결과(아래 표) | D022 7, 13 |
| 2 | `inserted_count` | int32 | `observation`에 새로 넣은 행 수 | D022 7 |
| 3 | `updated_count` | int32 | 값이 달라져 덮어쓴 행 수. `observation_history`에 남은 행 수와 같다 | D022 4 |
| 4 | `unchanged_count` | int32 | 이미 같은 값이 있던 행 수. 값은 숫자로 비교한다(설계서 5.2) | D022 4 |
| 5 | `skipped_empty_count` | int32 | 값이 비어 건너뛴 행 수. 원본 응답에는 남는다 | D022 15 |
| 6 | `error_code` | string | 오류 코드(아래 표). `SUCCESS`이면 비어 있다 | D022 13 |
| 7 | `error_message` | string | 오류 메시지. `SUCCESS`이면 비어 있다 | D022 13 |

- 행 수는 이 페이지 하나의 수다. 여러 페이지면 Go가 더한다.
- `NO_DATA`, `ERROR`이면 행 수는 모두 0이다.

### 결과 `ProcessStatus`

`collection_run.status`의 `success`, `no_data`, `error`에 대응한다. `PROCESS_STATUS_UNSPECIFIED`(0)는 쓰지 않으며, 받으면 오류로 다룬다.

| 결과 | 언제 | `error_code` | Rust가 DB에 쓰는 것 |
| --- | --- | --- | --- |
| `SUCCESS` | HTTP 200, 제공처 정상 응답, 모든 행을 읽었다 | 비어 있다 | `raw_response` 1행, `observation` |
| `NO_DATA` | 제공처가 데이터 없음으로 답했다: ECOS `INFO-200`, KOSIS `err` `30`, 공공데이터포털 `resultCode` `00` + `totalCount` 0 | 제공처 코드 (`INFO-200`, `30`, `00`) | `raw_response` 1행 |
| `ERROR` | 제공처 오류: ECOS `RESULT.CODE`(`INFO-100` 등), KOSIS `err`(`11` 등), 공공데이터포털 `returnReasonCode`(`30` 등) | 제공처 코드 그대로 | `raw_response` 1행 |
| `ERROR` | HTTP 200이 아니고 본문에 제공처 오류 코드가 없다 | `HTTP_<상태 코드>` (예: `HTTP_502`) | `raw_response` 1행 |
| `ERROR` | 응답을 읽을 수 없다: 예상한 모양이 아님, 기간이나 값을 바꿀 수 없음, 응답 주기가 지표 주기와 다름 | `INVALID_RESPONSE` | `raw_response` 1행 |

- 제공처 코드는 `samples/`의 오류 샘플(`ecos/error-INFO-100.json`, `error-INFO-200.json`, `kosis/error-11.json`, `error-30.json`, `data-go-kr/error-30.json`, `nodata-totalCount-0.json`)과 설계서 1.2에서 확인했다. Go `source.APIError`의 `Code`, `NoData`와 같다.
- `HTTP_<상태 코드>`, `INVALID_RESPONSE`는 이 약속서에서 정한 Rust 코드다.
- 관측값은 페이지 단위로 모두 쓰거나 하나도 쓰지 않는다. 한 행이라도 읽을 수 없으면 `ERROR`이고 `observation`에는 쓰지 않는다.

### gRPC 오류

Rust가 처리를 끝내지 못하면 `ProcessPageResponse` 대신 gRPC 오류를 돌려준다. 이때 DB에는 아무것도 쓰지 않는다(`raw_response` 포함).

| gRPC 상태 | 언제 |
| --- | --- |
| `INVALID_ARGUMENT` | 요청 값이 잘못됐다: `run_id` 0 이하, `indicator_id` 비어 있음, `page` 1 미만, `received_at` 없음, `http_status`가 100~599 밖, `request_masked`에 `***` 없음 |
| `NOT_FOUND` | `indicator_id`가 `indicator`에 없거나, `run_id`가 `collection_run`에 없다 |
| `UNAVAILABLE` | DB에 접속할 수 없다 |
| `INTERNAL` | 그 밖의 이유로 DB에 쓰지 못했다 |

- Go는 gRPC 오류와 Rust에 연결하지 못한 경우를 수집 실패로 다룬다(KEDP001-D020).

## 확인과 코드 생성

문법 확인(저장소 최상위 폴더에서):

```sh
protoc -I b-proto --descriptor_set_out=/dev/null b-proto/kedp/v1/processor.proto
```

- Rust는 tonic(KEDP001-D016)의 코드 생성을 쓴다.
- Go는 `protoc-gen-go`, `protoc-gen-go-grpc`를 쓴다. 생성 코드를 둘 Go 패키지는 proto에 쓰지 않는다(`go_package` 없음). Go 클라이언트 작업에서 정해 `--go_opt=M`, `--go-grpc_opt=M`으로 준다.

## 변경 규칙

- 필드 번호는 바꾸거나 다시 쓰지 않는다. 지운 필드의 번호와 이름은 `reserved`로 막는다.
- Go나 Rust가 받지 못하게 되는 변경은 package `kedp.v2`로 새로 만든다.
