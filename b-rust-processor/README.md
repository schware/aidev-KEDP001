# b-rust-processor

KEDP Rust 처리 서버다. Go(수집)가 제공처 응답을 페이지마다 gRPC로 넘기면 응답을 읽고 기간을 바꾼 뒤 DB에 쓴다.

- 약속서: `b-proto/kedp/v1/processor.proto`. 흐름, 결과 표, 오류는 `b-proto/README.md`를 따른다.
- 저장 규칙: `specification/30.database/README.md` (KEDP001-D022)
- gRPC 서버는 tonic(KEDP001-D016)이다. DB에는 `kedp_rust` 계정으로 접속한다(KEDP001-D022 14).

```text
b-rust-processor/
├── build.rs              빌드할 때 b-proto의 processor.proto로 gRPC 코드를 만든다
├── src/
│   ├── main.rs           시작점: 설정 읽기 → DB pool → gRPC 서버
│   ├── config.rs         .env와 환경 변수 읽기
│   ├── service.rs        Check, ProcessPage (요청 확인 → 지표 읽기 → 응답 읽기 → DB 쓰기)
│   ├── response.rs       응답 읽기, 빈 값 건너뛰기, 값 확인 (결과: 정상 / 데이터 없음 / 오류)
│   ├── response/         제공처별 응답 모양: ecos.rs, kosis.rs, data_go_kr.rs
│   ├── period.rs         기간 변환 (설계서 5.1)
│   └── db.rs             indicator 읽기, raw_response·observation 쓰기
├── tests/
│   ├── db.rs             시험 DB 테스트
│   ├── fixtures.sql      시험 DB에 넣는 지표와 수집 기록
│   └── check_history.sql 트리거가 남긴 이력 확인
└── scripts/db-test.sh    시험 DB를 만들어 DB 테스트를 돌리고 지운다
```

## 필요한 것

- Rust(cargo), `protoc`(빌드할 때 `build.rs`가 쓴다)
- 메모리가 작으므로 빌드는 `CARGO_BUILD_JOBS=2`로 하고, 다른 무거운 작업과 동시에 돌리지 않는다(`.claude/environment.md`).

## 설정

`.env`(저장소 최상위)와 환경 변수에서만 읽는다. 같은 이름의 환경 변수가 있으면 환경 변수를 쓴다. 빈 값의 목록은 `.env.example`에 있다.

| 변수 | 뜻 | 없을 때 |
| --- | --- | --- |
| `KEDP_DB_HOST` | DB 주소 | 시작하지 않는다 |
| `KEDP_DB_PORT` | DB 포트 | 시작하지 않는다 |
| `KEDP_DB_NAME` | DB 이름 | 시작하지 않는다 |
| `kedp_rust` | `kedp_rust` 계정 비밀번호 | 시작하지 않는다 |
| `KEDP_PROCESSOR_ADDR` | 이 서버의 주소(IP:포트) | `127.0.0.1:50051` |

- DB 접속 값은 `b-sql-db/README.md`의 "개발 서버의 DB"를 본다.
- 오류 메시지와 기록에는 변수 이름만 나오고 값은 나오지 않는다.

## 빌드

```sh
cd /srv/projects/aidev-kedp001/b-rust-processor
CARGO_BUILD_JOBS=2 cargo build --release
```

## 실행

```sh
cd /srv/projects/aidev-kedp001/b-rust-processor
CARGO_BUILD_JOBS=2 cargo run --release
# 다른 .env 파일: cargo run --release -- --env /경로/.env
```

- `Ctrl+C` 또는 SIGTERM(`docker stop`)을 받으면 멈춘다.
- 호출마다 한 줄씩 기록한다(run_id, indicator_id, page, 결과, 행 수, 오류 코드). 응답 본문과 요청 주소는 기록하지 않는다.
- DB 접속은 처음 호출을 받을 때 한다. 상태 확인은 `Check`로 한다.

호출해 보기(`grpcurl`, 저장소 최상위 폴더에서):

```sh
grpcurl -plaintext -import-path b-proto -proto kedp/v1/processor.proto \
  127.0.0.1:50051 kedp.v1.ProcessorService/Check
```

`ProcessPage`는 DB에 쓴다. `KEDP_DB_NAME`이 `kedp`이면 실제 DB에 쓰게 된다.

## 테스트

```sh
cd /srv/projects/aidev-kedp001/b-rust-processor

# 응답 읽기, 기간 변환, 설정 (DB와 실제 API를 쓰지 않는다)
CARGO_BUILD_JOBS=2 cargo test

# DB 테스트: 시험 DB(kedp_rust_check)를 만들어 돌리고 끝나면 지운다
./scripts/db-test.sh
```

- 응답 읽기 테스트는 `samples/`의 제공처 폴더 샘플(정상, 오류, 데이터 없음)을 쓴다.
- `samples/repository/`(조사 샘플, git 제외)로 하는 기간 확인 테스트(ECOS D·Q·A, KOSIS Q)는 기본으로 돌지 않는다. 파일이 있으면 `CARGO_BUILD_JOBS=2 cargo test repository -- --ignored`로 돌린다.
- `db-test.sh`는 DB 컨테이너 `kedp-postgres`에서 superuser로 시험 DB를 만들고 migration과 `tests/fixtures.sql`을 넣는다. 테스트는 `.env`의 접속 값과 `kedp_rust` 비밀번호로 접속하며, DB 이름만 시험 DB로 바꾼다. 테스트는 DB 이름이 `kedp_rust_check`가 아니면 멈춘다.
- `kedp_rust`는 `observation_history`를 읽을 수 없으므로, 트리거가 남긴 이력은 테스트 뒤에 superuser로 확인한다(`tests/check_history.sql`).

## 처리 규칙

`b-proto/README.md`에 적은 것 말고 이 서버가 정한 것이다.

- 응답 행이 지표 정의와 같은 계열인지 확인한다. 다르면 `INVALID_RESPONSE`다.
  - ECOS: `STAT_CODE`, `ITEM_CODE1`, 분류(`ITEM_CODE2`…)
  - KOSIS: `ORG_ID`, `TBL_ID`, `ITM_ID`, 분류(`C1`…), 주기(`PRD_SE`)
  - 공공데이터포털: `srtnCd`
- 빈 값은 `null`과 빈 문자열(공백만 있는 것 포함)이다. 결측 표기를 샘플로 확인하지 못했으므로(설계서 5.5) `"-"` 같은 다른 표기는 숫자가 아니어서 `INVALID_RESPONSE`가 된다.
- 값은 10진수(`3.5`, `61`, `.25`, `-.39`)만 받는다. 지수, 쉼표, `NaN`은 받지 않는다. `value_raw`에는 응답 문자열을 그대로 넣는다.
- 한 응답에 같은 기간이 두 번 나오면 `INVALID_RESPONSE`다.
- `observation.collected_at`에는 Go가 응답을 받은 시각(`received_at`)을 넣는다. 값이 같은 행도 `collected_at`, `run_id`, `value_raw`를 새 값으로 덮어쓴다(이력은 남지 않는다).
- 원본 응답이 UTF-8 JSON이면 `raw_response.body`(jsonb), 아니면 `body_text`에 넣는다. `body_text`에서 UTF-8이 아닌 바이트와 NUL은 U+FFFD로 바꾼다.
