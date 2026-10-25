# b-go-end 개발 완료 문서

- 대상 폴더: `b-go-end/`
- 문서 저장 위치: `docs/dev-complete/b-go-end.md`
- 독자: 새로 온 개발자. 이 문서만 보고 실행하고 구조를 이해할 수 있어야 한다.
- 작성 규칙: 문장마다 출처(결정 ID, Issue 번호, 파일 경로)를 붙인다. 기록에 없는 것은 "확인 필요"로 적는다.
- 범위: 진행 과정과 부딪힌 문제는 Issue #3~#14와 2026-10-25 Issue #14 작업 session 기록까지만 다룬다. 구조·실행 방법은 지금 `main`의 `b-go-end/` 소스를 기준으로 쓴다.
- 출처 표기: `session c8dd3c10`은 2026-10-25 Issue #14 작업 session 기록(`~/.claude/projects/-srv-projects-aidev-kedp001/c8dd3c10-….jsonl`)이다.

## 1. 한 줄 요약

`b-go-end`는 지표 하나의 원본 응답을 제공처(ECOS, KOSIS, 공공데이터포털)에서 받아 정상 응답인지 확인하고 `samples/<제공처>/`에 저장하는 수집 프로그램이다(`b-go-end/cmd/server/main.go`, KEDP001-D021).

- 아직 DB 저장, 하루 1회 예약 수집, 조회 REST API, Rust gRPC 호출은 없다(`b-go-end/cmd/server/main.go`의 `run` 함수에 해당 코드 없음; 계획은 KEDP001-D015, KEDP001-D019).
- 지금 받은 샘플은 DB 설계의 근거로 쓴다(KEDP001-D004, KEDP001-D021 "영향").

## 2. 실행하는 방법

### 2.1 준비

| 항목 | 내용 | 출처 |
| --- | --- | --- |
| Go | `1.27.3` | `b-go-end/go.mod` |
| 외부 패키지 | `gopkg.in/yaml.v3 v3.0.1` 하나(`indicators.yaml` 읽기) | `b-go-end/go.mod`, `b-go-end/internal/config/config.go` |
| API 키 | `ECOS_API_KEY`, `KOSIS_API_KEY`, `DATA_GO_KR_API_KEY` | `b-go-end/internal/config/config.go` `LoadKeys` |
| 키 보호 장치 | hook(`git config core.hooksPath .githooks`)과 gitleaks 설치 | KEDP001-D012 "영향", `docs/key-protection.md` 2-1 |

- 키는 저장소 맨 위의 `.env`에서 읽는다. `-env` flag의 기본값이 `../.env`다(`b-go-end/cmd/server/main.go`).
- 같은 이름의 환경 변수가 있으면 환경 변수가 `.env`보다 먼저다(`b-go-end/internal/config/config.go` `LoadKeys` 주석).
- `.env` 파일이 없어도 오류가 아니다. 대신 해당 제공처를 부를 때 `<환경 변수 이름> is not set` 오류가 나고 요청을 보내지 않는다(`b-go-end/internal/config/config.go`, `b-go-end/internal/source/source.go` `RequireKey`, `Base.Get`).
- 키 값은 화면, 로그, 문서, 샘플에 쓰지 않는다(`.claude/api.md`, KEDP001-D012).

### 2.2 실행 명령

`b-go-end/` 폴더에서 실행한다. 기본 경로(`../.env`, `config/indicators.yaml`, `../samples`)가 이 폴더 기준이기 때문이다(`b-go-end/cmd/server/main.go`).

```sh
cd b-go-end
go run ./cmd/server -indicator ecos-base-rate -start 202401 -end 202609
```

위 명령은 `samples/ecos/` 샘플을 받을 때 쓴 명령이다(`samples/ecos/README.md`).

| flag | 기본값 | 뜻 | 출처 |
| --- | --- | --- | --- |
| `-indicator` | (필수) | `indicators.yaml`의 지표 `id` | `b-go-end/cmd/server/main.go` |
| `-start` | (필수) | 조회 시작 기간 | `b-go-end/cmd/server/main.go` |
| `-end` | (필수) | 조회 끝 기간 | `b-go-end/cmd/server/main.go` |
| `-env` | `../.env` | `.env` 파일 위치 | `b-go-end/cmd/server/main.go` |
| `-indicators` | `config/indicators.yaml` | 지표 정보 파일 | `b-go-end/cmd/server/main.go` |
| `-out` | `../samples` | 샘플 저장 폴더(아래에 제공처 폴더를 만든다) | `b-go-end/cmd/server/main.go` |
| `-timeout` | `20s` | 요청 시간 제한 | `b-go-end/cmd/server/main.go` |

- 필수 flag 셋 중 하나라도 빠지면 사용법을 출력하고 종료 코드 2로 끝난다(`b-go-end/cmd/server/main.go` `main`).
- 기간 형식은 지표의 주기를 따른다. 예를 들어 주기가 `M`이면 `202401`이다(`b-go-end/internal/source/source.go` `Request` 주석).
- 공공데이터포털 지표는 기간을 `YYYYMMDD`로 넣고, 시작일과 끝 날짜를 모두 포함한다(`b-go-end/internal/source/data_go_kr/data_go_kr.go` `endParam`).

### 2.3 실행 결과

- 성공하면 로그에 `GET <주소>`가 키 자리 `***`로 찍히고, 마지막에 `saved <행 수> rows to <경로>`가 찍힌다(`b-go-end/internal/fetch/fetch.go` `Get`, `b-go-end/cmd/server/main.go` `run`).
- 샘플 파일 이름은 `통계코드-항목코드-분류코드…-주기-시작-끝.json`이다. 비어 있는 칸은 이름에서 빠진다(`b-go-end/internal/source/source.go` `SampleName`).
  - 예: `samples/ecos/722Y001-0101000-M-202401-202609.json`(`samples/ecos/README.md`)
- 제공처가 오류 응답을 주면 저장하지 않고 오류로 끝난다. 정상 응답만 샘플로 저장한다(`b-go-end/cmd/server/main.go` `run`의 `Check` 호출과 주석).
- 응답 본문에 키가 어떤 형태로든 들어 있으면 저장하지 않는다(`b-go-end/internal/fetch/fetch.go` `Save`).

### 2.4 테스트

```sh
cd b-go-end
go vet ./...
go test -count=1 ./...
```

- 테스트는 `samples/`에 받아 둔 응답만 읽고 실제 API는 부르지 않는다. 그래서 키 없이도 돈다(KEDP001-D021 5절, `b-go-end/internal/source/sourcetest/sourcetest.go`).
- 샘플 파일에 키가 남아 있지 않은지도 테스트로 확인한다(`b-go-end/internal/source/sourcetest/sourcetest.go` `CheckMasked`, 각 제공처의 `TestSamplesHaveNoKey…`).
- 2026-10-25 이 문서를 쓰면서 `main`에서 `go vet ./...`과 `go test -count=1 ./...`을 돌렸고 모든 패키지가 통과했다(이 문서 작성 시 실행 결과).

### 2.5 지표 추가하는 법

지표 정보는 코드에 넣지 않고 `b-go-end/config/indicators.yaml`에 둔다. DB가 생기면 DB로 옮긴다(KEDP001-D021 2절, KEDP001-D018).

```yaml
- id: ecos-base-rate      # -indicator에 넣는 값
  name: 한국은행 기준금리
  source: ecos            # ecos | kosis | data-go-kr
  stat_code: 722Y001
  item_code: "0101000"
  cycle: M                # 제공처 요청에 넣는 주기 값 그대로
```

(`b-go-end/config/indicators.yaml`, `b-go-end/internal/config/config.go` `Indicator`)

- 항목 코드는 ECOS `StatisticItemList`나 KOSIS `getMeta(type=ITM)` 응답으로 확인한 값만 적는다(`b-go-end/config/indicators.yaml` 머리 주석, `.claude/CLAUDE.md` 1절).
- `classes`는 KOSIS의 `objL1`, `objL2`… 순서로 들어가는 분류 코드다. ECOS는 비워 둔다(`b-go-end/internal/config/config.go` `Indicator` 주석).
- 지금 등록된 지표는 7개다(`b-go-end/config/indicators.yaml`).

| id | 이름 | 제공처 | 주기 |
| --- | --- | --- | --- |
| `ecos-base-rate` | 한국은행 기준금리 | ecos | M |
| `kosis-employment-rate` | 고용률(15세 이상) | kosis | M |
| `kosis-births` | 출생아 수 | kosis | Y |
| `kosis-total-population` | 총인구 | kosis | Y |
| `data-go-kr-kodex-200` | KODEX 200 | data-go-kr | D |
| `data-go-kr-kodex-leverage` | KODEX 레버리지 | data-go-kr | D |
| `data-go-kr-kodex-inverse` | KODEX 인버스 | data-go-kr | D |

## 3. 구조 (폴더와 주요 파일)

폴더 구조는 KEDP001-D021 2절을 따른다.

```text
b-go-end/
├── go.mod, go.sum
├── cmd/server/main.go          프로그램 시작점
├── config/indicators.yaml      지표 정보 (DB가 생기기 전까지 임시)
└── internal/
    ├── config/                 공통: .env 키 읽기, indicators.yaml 읽기
    ├── fetch/                  공통: 요청, 시간 제한, 키 가리기, 응답 저장
    └── source/
        ├── source.go           공통 약속(Source interface)과 공통 도구
        ├── sourcetest/         테스트 도구(샘플 읽기, 키 가림 확인)
        ├── ecos/               1단계
        ├── kosis/              2단계
        └── data_go_kr/         3단계
```

(KEDP001-D021 2절, `b-go-end/` 파일 목록)

| 파일 | 하는 일 | 출처 |
| --- | --- | --- |
| `cmd/server/main.go` | flag를 읽고, 지표의 `source`에 맞는 제공처를 골라 받기 → 확인 → 저장을 한 번 실행한다 | `b-go-end/cmd/server/main.go` |
| `internal/config/config.go` | `Keys`(키 3개), `LoadKeys`, `Indicator`, `LoadIndicators`, `FindIndicator` | `b-go-end/internal/config/config.go` |
| `internal/fetch/fetch.go` | `Client.Get`(요청과 로그), `Mask`(키 가리기), `StatusError`(200이 아닌 응답), `Client.Save`(저장) | `b-go-end/internal/fetch/fetch.go` |
| `internal/source/source.go` | `Source` interface, `Request`, `SampleName`, `APIError`, `NoData`, `RequireKey`, `Base`, `Count`, `RequireAllRows` | `b-go-end/internal/source/source.go` |
| `internal/source/ecos/ecos.go` | ECOS `StatisticSearch` 요청 주소 만들기와 응답 읽기 | `b-go-end/internal/source/ecos/ecos.go` |
| `internal/source/kosis/kosis.go` | KOSIS `statisticsParameterData` 요청 주소 만들기와 응답 읽기 | `b-go-end/internal/source/kosis/kosis.go` |
| `internal/source/data_go_kr/data_go_kr.go` | 공공데이터포털 `getETFPriceInfo_V2` 요청 주소 만들기와 응답 읽기 | `b-go-end/internal/source/data_go_kr/data_go_kr.go` |
| `internal/source/sourcetest/sourcetest.go` | 테스트에서 `samples/<제공처>/`를 읽고, 샘플에 키가 가려졌는지 확인 | `b-go-end/internal/source/sourcetest/sourcetest.go` |

### 3.1 제공처가 지키는 공통 약속

모든 제공처는 `source.Source` interface를 따른다(`b-go-end/internal/source/source.go`).

```go
type Source interface {
    Name() string                                          // 샘플 폴더 이름: ecos, kosis, data-go-kr
    Fetch(ctx context.Context, req Request) ([]byte, error) // 원본 응답을 받는다
    Check(body []byte) (int, error)                         // 정상 응답인지 확인하고 행 수를 돌려준다
}
```

- 각 제공처는 `source.Base`를 넣어(embed) 이름, 키 확인, 요청을 함께 쓴다(`b-go-end/internal/source/source.go` `Base`, 각 제공처의 `New`).
- 제공처가 본문으로 돌려준 오류는 `*source.APIError`로 통일한다. "데이터 없음"은 `NoData`로 구분한다(`b-go-end/internal/source/source.go` `APIError`, `NoData`).
- 한 번에 받은 행 수가 전체보다 적으면 `RequireAllRows`가 오류를 낸다. 페이지 나누기가 아직 없기 때문이다(`b-go-end/internal/source/source.go` `RequireAllRows` 주석).
- 응답 값은 숫자로 바꾸지 않고 응답 그대로 문자열로 둔다(각 제공처 `Row` 주석).

### 3.2 새 제공처를 붙이는 방식

- 새 제공처를 붙이다가 앞의 것과 겹치는 부분을 찾으면, 공통으로 빼고 앞의 제공처도 그 자리에서 바로 공통을 쓰게 고친다(KEDP001-D021 4절).
- 마지막에 한꺼번에 다시 연결하는 단계는 두지 않는다(KEDP001-D021 4절).
- 단계가 끝날 때마다 앞의 제공처 테스트도 모두 통과해야 한다(KEDP001-D021 5절).

## 4. 데이터 흐름

```text
flag 읽기
  → config.LoadKeys(.env / 환경 변수)
  → config.LoadIndicators + FindIndicator(indicators.yaml)
  → fetch.New(timeout, 키 목록)        키를 아는 HTTP Client
  → indicator.source로 제공처 고르기     ecos / kosis / data-go-kr
  → src.Fetch                         요청 주소 만들기 → Client.Get(로그는 *** 가림)
  → src.Check                         정상 응답인지, 행 수
  → Client.Save                       키가 섞였으면 저장 거부
  → samples/<제공처>/<SampleName>
```

(`b-go-end/cmd/server/main.go` `run`)

### 4.1 키가 새지 않게 막는 곳

키를 주소에 붙인 요청은 `fetch` 패키지를 거쳐 보내고, 기록할 때는 `Mask`를 거친 문자열만 쓴다. 이것이 키 보호 5겹 중 2겹이다(`b-go-end/internal/fetch/fetch.go` 패키지 주석, KEDP001-D012).

| 어디 | 무엇을 막나 | 출처 |
| --- | --- | --- |
| `Client.Get`의 로그 | 요청 주소의 키를 `***`로 바꿔 찍는다 | `b-go-end/internal/fetch/fetch.go` |
| `Client.Get`의 오류 | 오류 메시지의 키를 가리고, 키가 든 원래 오류는 감싸지 않는다 | `b-go-end/internal/fetch/fetch.go` |
| `StatusError.URL` | 200이 아닌 응답의 주소도 가린 주소만 둔다 | `b-go-end/internal/fetch/fetch.go` |
| `Client.Save` | 응답 본문에 키가 들어 있으면 저장하지 않는다 | `b-go-end/internal/fetch/fetch.go` |
| `readEnvFile` | `.env` 읽기 오류 메시지에 줄 내용(키 값)을 넣지 않는다 | `b-go-end/internal/config/config.go` |

- `Mask`는 키의 원래 형태뿐 아니라 주소에 들어갈 때 인코딩된 형태도 가린다(`b-go-end/internal/fetch/fetch.go` `forms`).
- 키가 이미 인코딩된 형태(공공데이터포털 Encoding 키)이면 푼 형태도 가린다(`b-go-end/internal/fetch/fetch.go` `forms`).
- 요청 주소를 만드는 함수(`searchURL`, `dataURL`, `priceURL`)가 만든 주소는 `fetch.Client`를 거쳐서만 기록한다(각 제공처 함수 주석).

### 4.2 제공처별 차이

| | ECOS | KOSIS | 공공데이터포털 |
| --- | --- | --- | --- |
| API | `StatisticSearch` | `statisticsParameterData` | `getETFPriceInfo_V2` |
| 키 위치 | 주소 경로 | `apiKey=` | `serviceKey=` |
| 한 번에 받는 최대 행 | 1000 | 지정 없음 | 1000 |
| 정상 응답 모양 | `StatisticSearch.row` | 행 배열 | `response.body.items.item` |
| 오류 응답 모양 | `RESULT{CODE, MESSAGE}` | `{err, errMsg}` 객체 | `OpenAPI_ServiceResponse`, 또는 `resultCode`가 `00`이 아님 |
| 데이터 없음 | `INFO-200` | `30` | `00` + `totalCount 0` |
| 기간 | 시작·끝 포함 | 시작·끝 포함 | 끝 날짜를 포함하지 않아 코드에서 하루를 더함 |

(`b-go-end/internal/source/ecos/ecos.go`, `b-go-end/internal/source/kosis/kosis.go`, `b-go-end/internal/source/data_go_kr/data_go_kr.go`, `b-go-end/internal/source/source.go` `APIError` 주석)

- KOSIS 요청에는 `jsonVD=Y`를 넣는다. 없으면 key에 따옴표가 없는 JSON이 온다(`b-go-end/internal/source/kosis/kosis.go` `dataURL` 주석).
- KOSIS는 요청에 `Y`를 넣어도 응답 `PRD_SE`에는 `A`로 온다(`b-go-end/internal/source/kosis/kosis.go` `Row` 주석).
- KOSIS의 기관 ID는 `101`(통계청)로 고정되어 있다. 조사한 통계표가 모두 통계청이기 때문이다(`b-go-end/internal/source/kosis/kosis.go` `orgID` 주석).
- 공공데이터포털은 인증키 오류를 HTTP 4xx 본문으로 보낸다. 그래서 `StatusError`의 본문을 읽어 `APIError`로 바꾼다(`b-go-end/internal/source/data_go_kr/data_go_kr.go` `Fetch`, `readStatusError`).
- 공공데이터포털 키는 Decoding 키와 Encoding 키를 모두 받는다. Encoding 키는 두 번 인코딩되지 않게 원래 형태로 풀어 쓴다(`b-go-end/internal/source/data_go_kr/data_go_kr.go` `New`, `decodeKey`).
- 공공데이터포털의 `likeSrtnCd`는 부분 일치 검색이지만, 6자리 종목 코드로는 그 종목만 온다(2026-10-25 세 종목 확인)(`b-go-end/internal/source/data_go_kr/data_go_kr.go` `priceURL` 주석).

## 5. 주요 결정 (결정 ID와 한 줄 이유)

이유는 각 결정 기록의 "결정 이유"에 적힌 것만 옮겼다.

### 5.1 b-go-end에 직접 영향을 준 결정

| ID | 결정 | 한 줄 이유 |
| --- | --- | --- |
| KEDP001-D021 | 공통 최소 부분을 먼저 두고, 제공처를 ECOS → KOSIS → 공공데이터포털 순으로 하나씩 붙이며 공통을 다듬는다 | 실제로 붙여 보면서 겹치는 부분을 확인한 뒤 공통으로 만들고, 첫 구현에 공통 틀이 있어야 다음 제공처도 같은 틀로 만든다 |
| KEDP001-D004 | 실제 API 응답을 확인한 뒤 DB 스키마를 정한다 | 실제 데이터를 근거로 저장 구조를 비교하는 것이 안전하다 |
| KEDP001-D012 | API 키 보호 5겹, 그중 2겹은 Go 코드에서 기록할 때 키를 `***`로 가리기 | 2겹은 Git 밖(로그, 화면, 공유)으로 새는 길을 막는다 |
| KEDP001-D001 | 제공처를 ECOS, KOSIS, 공공데이터포털로 제한 | 공식 기관 데이터이고, 무료이고, 장기 운영이 가능하고, 출처가 명확하다 |
| KEDP001-D013 | 1단계 지표에 주가지수와 KODEX 3종 추가 | 지수와 ETF를 나란히 보면 움직임 차이가 보인다 |
| KEDP001-D016 | Go 수집·API는 Go 기본 기능(`net/http`) | 조회 API 정도는 추가 도구 없이 충분하다 |
| KEDP001-D002 | Go는 외부 API 수집과 서비스 API 제공 | 외부 API 수집에 적합하고 서비스 API 구현이 단순하다 |

(`records/07.decision/decisions/` 각 파일)

### 5.2 배경이 되는 결정

| ID | 결정 | 한 줄 이유 |
| --- | --- | --- |
| KEDP001-D015 | 기준금리를 첫 대상으로 Go 수집 → Rust 검증 → DB 저장 → Go 조회 API를 먼저 만든다 | API가 정해지면 화면이 받을 데이터 모양이 확정된다 |
| KEDP001-D018 | 지표 설정은 관리자 화면에서 하고 DB에 저장한다 | 나중에 다시 보고 수정할 수 있다 |
| KEDP001-D019 | 수집은 하루 1회, 기본 06:00(Asia/Seoul), Go 안의 예약 기능이 실행 | MVP는 하루 1회로 단순하게 시작한다 |
| KEDP001-D005 | AI Agent 역할을 다섯 영역으로 나눈다 | 기술 Agent는 담당 영역 구현에 집중한다 |
| KEDP001-D006 | 한 저장소, `b-go-end/` 같은 폴더 이름 | 맞물리는 변경을 commit 하나로 처리하고, 폴더 이름을 Docker 서비스 이름과 같게 한다 |
| KEDP001-D008 | Linux 개발 서버에 도구를 직접 설치하고, 버전은 `go.mod`로 고정 | 명령이 빠르고 편집기 자동완성이 잘 동작한다 |
| KEDP001-D010 | 작업마다 branch를 만들고 PR로 합친다. merge는 소유자가 한다 | 작업 단위를 나누고 검토 과정을 유지한다 |

(`records/07.decision/decisions/` 각 파일)

## 6. 진행 과정 (Issue와 PR 순서)

| 순서 | Issue | PR | 내용 | 출처 |
| --- | --- | --- | --- | --- |
| 1 | - | #3 | 줄바꿈을 LF로 통일(`.gitattributes`), Jira 연결 시험 겸 | PR #3 |
| 2 | - | #4 | KEDP 문서 v0.1을 `docs/`로 가져옴 | PR #4 |
| 3 | #5 | #6 | Agent 규칙 `.claude/CLAUDE.md`, `api.md`, `git.md`, `environment.md` | Issue #5, PR #6 |
| 4 | #7 | #8 | 2026-10-22~24 합의를 README, 명세, 결정 기록, 키 보호 가이드에 반영 | Issue #7, PR #8 |
| 5 | #9 | #10, #11 | D015~D020 추가, 화면 명세 `specification/50.web` | Issue #9, PR #10, PR #11 |
| 6 | #12 | #13 | D021 Go 수집 모듈 구조 결정 기록 | Issue #12, PR #13 |
| 7 | #14 | #15 | 수집 모듈 1단계: 공통 최소 부분과 ECOS 기준금리 | Issue #14, `git log`(7ba5808) |

- Issue #9에 PR이 두 개(#10, #11) 연결된 이유는 기록에 없다(확인 필요).

### 6.1 Issue #14: 수집 모듈 1단계

Issue #14의 "끝났다는 기준"은 다섯 가지다(Issue #14).

1. D021의 폴더 구조대로 `b-go-end/`를 만든다.
2. 기준금리 정상 응답이 `samples/ecos/`에 저장된다(키 없음).
3. 샘플로 읽기 테스트를 만들고 `go test`가 통과한다.
4. `10.indicators`의 기준금리 칸을 채운다.
5. 로그와 출력에 키가 보이지 않는다.

Issue #14에서 하지 말라고 한 것은 KOSIS·공공데이터포털 구현, DB·Rust·화면 작업, commit이다(Issue #14).

작업 결과는 다음과 같다(session c8dd3c10 결과 보고).

- D021 폴더 구조대로 `b-go-end/`를 만들었다. `kosis/`, `data_go_kr/`는 패키지 선언만 있는 빈 자리로 두었다.
- 공통 부분은 `internal/config`(키와 `indicators.yaml` 읽기)와 `internal/fetch`(요청, 시간 제한, `***` 가리기, 응답 저장)에 두었다.
- ECOS 722Y001의 항목 목록을 실제로 조회해 기준금리 항목 코드 `0101000`을 찾았고, 주기 `M`으로 `indicators.yaml`에 등록했다.
- 정상 응답 샘플(2024-01~2026-09, 33행)을 이 Go 프로그램으로 받아 저장했다(`samples/ecos/README.md`에도 같은 내용).
- 실제로 확인한 오류 응답 `INFO-100`(인증키 오류)과 `INFO-200`(데이터 없음)도 샘플로 저장했다(`samples/ecos/README.md`).
- `specification/10.indicators/README.md`의 기준금리 칸을 채우고 상태를 `verified`로 바꿨다.
- 테스트: `go vet ./...`, `gofmt -l .` 문제 없음. `go test ./...` config·fetch·ecos 통과.
- 키 노출 검사: `.env`의 키 세 개로 변경 파일을 grep한 결과 0건, `gitleaks dir`(b-go-end, samples) 결과 no leaks found.

이 문서는 Issue #14까지만 다룬다. 2·3단계 코드(`kosis/`, `data_go_kr/`, 공통 도구 추가)도 지금 `main`에 있지만, 그 진행 과정은 이 문서의 범위가 아니다(KEDP001-D021 4절, `b-go-end/` 소스).

## 7. 부딪힌 문제와 해결

Issue #14 작업 session 기록에 남은 것만 적는다.

| 문제 | 해결 또는 상태 | 출처 |
| --- | --- | --- |
| 첫 요청에서 Bash 명령이 세 번 연속 실행되지 않았다. auto mode 안전 검사 서버가 판정을 돌려주지 않는 일시적 오류였다 | Sun&Moon이 같은 요청을 다시 보냈고, 그 뒤 정상 진행됐다 | session c8dd3c10 (06:15, 06:16) |
| `main`에 Issue #14 제목과 같은 commit(`d73d19b`)이 이미 있었다 | 확인해 보니 D021 문서만 추가한 commit이었고 코드는 없었다. 그래서 코드를 새로 만들었다 | session c8dd3c10 (06:16), `git log` |
| 기준금리 항목 코드가 정해져 있지 않았다 | ECOS `StatisticItemList`로 722Y001 항목 목록을 조회해 `0101000`을 찾았다 | session c8dd3c10 결과 보고, `b-go-end/config/indicators.yaml` 머리 주석 |
| branch를 만들지 않고 `main` 위에서 작업했다. 전역 규칙(branch는 요청이 있을 때만)과 프로젝트 `git.md`(작업마다 branch)가 서로 달랐다 | `go/14-ecos-base-rate` branch를 제안했고, 그 이름으로 PR #15가 만들어졌다 | session c8dd3c10 결과 보고, `.claude/git.md`, `git branch -a` |
| YAML을 읽으려고 외부 패키지 `gopkg.in/yaml.v3`를 추가했다. D016은 `net/http`만 정했다 | 확인 필요로 남겼다(8장) | session c8dd3c10 결과 보고, KEDP001-D016, `b-go-end/go.mod` |
| 페이지 나누기가 없다. 한 번에 최대 1000행을 받고, 넘으면 오류를 낸다 | 남은 문제로 보고했다(8장) | session c8dd3c10 결과 보고, `b-go-end/internal/source/ecos/ecos.go` `pageSize` |
| `docs/key-protection.md` 4장의 "따라하기" 보강이 필요했다 | Go 담당 폴더 밖이라 하지 않고 보고했다 | session c8dd3c10 결과 보고, `.claude/CLAUDE.md` 2절 |

## 8. 확인 필요

| 항목 | 지금 상태 | 출처 |
| --- | --- | --- |
| `gopkg.in/yaml.v3` 의존성 | D016은 `net/http`만 정했고, 외부 패키지 추가를 승인한 결정 기록은 없다 | KEDP001-D016, `b-go-end/go.mod`, session c8dd3c10 |
| 페이지 나누기 | ECOS·공공데이터포털은 1000행이 넘으면 오류를 낸다. ECOS 일 주기 전체 기간은 약 1만 행이다 | `b-go-end/internal/source/source.go` `RequireAllRows`, session c8dd3c10 |
| 수집 주기와 논리 key | 기준금리 수집 주기(지금은 `M`)와 논리 key는 정해지지 않았다 | session c8dd3c10 결과 보고, KEDP001-D004 |
| 예약 수집 | D019는 하루 1회 06:00 예약 수집을 정했지만, 코드에는 아직 없다 | KEDP001-D019, `b-go-end/cmd/server/main.go` |
| DB 저장, Rust gRPC 호출, 조회 API | D015의 순서에는 있지만, 코드에는 아직 없다 | KEDP001-D015, `b-go-end/` 소스 |
| KOSPI·KOSPI200 제공처 | D013 후속 작업에 "제공처 확인"으로 남아 있고, `indicators.yaml`에 없다 | KEDP001-D013, `b-go-end/config/indicators.yaml` |
| Issue #9의 PR이 두 개인 이유 | 기록에 없다 | PR #10, PR #11 |

## 부록. 조회·작업 중 Python을 많이 쓴 이유

2026-10-25 session 기록 5개(`c8dd3c10`, `dec5b669`, `45d7bdf6`, `da1e199e`, `ebacdeb8`)에서 Agent가 Bash로 실행한 명령을 확인했다.

### 무엇에 썼나

모두 19번 썼고, 용도는 두 가지다(위 session 기록의 Bash 명령과 설명).

| 용도 | 횟수 | 예 |
| --- | --- | --- |
| 받은 JSON 응답·샘플 들여다보기 | 8 | `python3 -m json.tool`로 ECOS 항목 목록 보기, `json.load`로 행 수·첫 행·끝 행·key 목록 요약 |
| 파일 고치기 | 11 | Go 소스·테스트·`specification/10.indicators/README.md`를 문자열 바꾸기로 수정 |

- 들여다보기 8번은 session `c8dd3c10` 2번, `dec5b669` 2번, `45d7bdf6` 1번, `da1e199e` 1번, `ebacdeb8` 2번이다.
- 파일 고치기 11번은 session `dec5b669` 3번, `45d7bdf6` 8번이다.
- 이 문서의 범위인 Issue #14 session(`c8dd3c10`)에서는 들여다보기에만 2번 썼다.

### 이유

- **기록된 이유는 없다.** 다섯 session의 Agent 글에 Python을 고른 이유를 설명한 곳은 없다(위 session 기록에서 `python`, `jq` 검색 결과).
- `jq`는 서버에 설치되어 있다(`/usr/bin/jq`). 그러니 "다른 도구가 없어서"는 이유가 될 수 없다(이 문서 작성 시 `command -v jq`).
- 같은 다섯 session에서 파일 수정 도구 Edit은 7번, Bash는 모두 75번 썼다(위 session 기록의 도구 사용 집계).
- 명령 설명에서 볼 수 있는 것: 파일 고치기에 쓴 명령 중 여러 개는 "여러 파일 수정 + 테스트 실행"을 한 명령으로 묶었다. 예를 들어 설명이 `Refactor ECOS/KOSIS to use common Base, run tests`인 명령이 있다(session `45d7bdf6`).
- 그 밖의 이유("Python이 더 편해서" 등)는 기록으로 확인할 수 없다(확인 필요).
