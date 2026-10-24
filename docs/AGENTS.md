# AI Agent 작업 규칙

## 공통

- 모든 Agent는 `project.yaml`에 고정된 Common Specification을 따른다.
- 확정된 결정은 `records/07.decision/decisions/`에서 확인한다.
- `proposed`와 `deferred` 상태를 현재 실행 기준으로 사용하지 않는다.
- 실제 API 확인 전 DB 구조를 확정하지 않는다.
- 담당 영역 밖 변경이 필요하면 임의 수정하지 않고 보고한다.
- 프로젝트 소유자만 결정 확정, 위험 수용, 최종 병합을 수행한다.

## Common Specification Agent

- 사용자 설계를 간단히 정리한다.
- 기술별 작업을 나눈다.
- 누락, 중복과 충돌을 확인한다.
- Decision 초안을 작성할 수 있다.
- 코드 작성과 최종 결정은 하지 않는다.

## TypeScript Agent

- 공개 웹 화면, 차트와 데이터 표를 담당한다.
- ECOS·KOSIS를 직접 호출하지 않는다.
- DB에 직접 접근하거나 통계 계산 규칙을 결정하지 않는다.

## Go Agent

- ECOS·KOSIS API 호출과 실제 응답 샘플 확보를 담당한다.
- 수집 흐름과 REST API를 담당한다.
- DB 구조와 집계 규칙을 독자적으로 확정하지 않는다.

## Rust Agent

- 샘플 응답의 key, type, 기간, 값, 단위와 차원을 분석한다.
- 이후 정규화, 검증, 중복·수정 감지와 승인된 집계를 담당한다.
- 외부 API 정책과 DB 구조를 독자적으로 변경하지 않는다.

## Database Agent

- API 조사 전에는 구조 비교, 논리 key 검토와 설계 질문만 작성한다.
- API 조사 후 DB 스키마, migration, 인덱스와 조회 SQL을 담당한다.
- API 조사 전 DDL과 migration을 작성하지 않는다.

## 결과 보고

```markdown
## 작업 결과

- 변경 내용:
- 변경 파일:
- 테스트:
- 남은 문제:
```
