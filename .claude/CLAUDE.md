# KEDP001 프로젝트 규칙

이 파일은 KEDP001(한국 경제 데이터 플랫폼) 저장소에서 일하는 AI Agent가 따르는 규칙이다.
v0.1의 `docs/AGENTS.md`를 대신한다. 사람(팀원)도 같은 규칙을 따른다.

## 1. 문서와 기준

| 무엇 | 어디 |
| --- | --- |
| 프로젝트 기본 정보 | `project.yaml` |
| 명세 | `specification/` |
| 확정된 결정 | `records/07.decision/decisions/` |
| 안내서·참고 자료 | `docs/` |
| 작업 목록 | GitHub Issues |
| API 응답 샘플 | `samples/` |

- `docs/AGENTS.md`, `docs/TODO.md`는 v0.1 기록이다. 현재 기준으로 쓰지 않는다.
- `proposed`, `deferred` 상태의 결정은 실행 기준으로 쓰지 않는다.
- 실제 API 응답을 확인하기 전에는 DB 구조를 확정하지 않는다.
- 확인되지 않은 지표 코드를 사실처럼 기록하지 않는다.
- 모르는 것은 먼저 `samples/`에서 찾는다. 조사 샘플은 `samples/repository/`에 있다.
- 새 샘플이 필요하면 샘플 수집 도구(`b-go-end/cmd/sample`) 명령을 그대로 실행할 수 있게 적고, 무엇을 확인하려는지 한 줄과 함께 소유자에게 요청한다.
- 소유자에게는 소유자만 할 수 있는 결정만 묻는다. 질문마다 배경, 선택지, 추천, 영향을 적는다.
- 이미 결정된 것(`records/07.decision/decisions/`)은 다시 묻지 않는다.
- 결정 확정, 위험 수용, 최종 merge는 프로젝트 소유자만 한다.

## 2. 역할과 담당 폴더

역할은 사람이 지정한다. 한 번에 한 역할로 일한다.

| 역할 | 폴더 | 하는 일 | 하지 않는 일 |
| --- | --- | --- | --- |
| Common | (문서) | 설계 정리, 작업 나누기, 누락·충돌 확인, Decision 초안 | 코드 작성, 최종 결정 |
| TypeScript | `f-ts-end/` | 화면, 차트, 표 | ECOS·KOSIS 직접 호출, DB 직접 접근 |
| Go | `b-go-end/` | ECOS·KOSIS·공공데이터포털 호출, 샘플 확보, 수집, REST API, Rust 서버 호출(gRPC) | DB 구조·집계 규칙 확정 |
| Rust | `b-rust-processor/` | 응답 분석, 정규화, 검증, gRPC 서버 | 외부 API 정책·DB 구조 변경 |
| Database | `b-sql-db/` | API 조사 전에는 구조 비교와 질문, 조사 뒤에는 스키마·migration | API 조사 전 DDL 작성 |

- `b-proto/`(Go–Rust 약속서)는 변경안만 제시하고, 소유자가 승인한 뒤 바꾼다.
- 담당 폴더 밖을 고쳐야 하면 직접 고치지 않고 보고한다.
- 소유자가 직접 맡겠다고 한 파일이나 분야는 수정하지 않는다. 요청받으면 설명과 검토만 한다.

## 3. 세부 규칙

항상 지키는 규칙:

@api.md
@git.md

필요할 때 읽는 정보:

- 개발 환경(서버, 도구, 메모리 주의): `.claude/environment.md`
