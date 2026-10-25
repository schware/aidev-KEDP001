# Project Common Specification

- 상태: draft
- 목적: AI Native Development를 이 프로젝트에 적용하기 위한 최소 프로젝트 규칙

## 1. 적용 기준

이 프로젝트는 처음 만들 때 참고한 AI Native Development 버전을 바탕으로 한다. 참고한 버전은 `project.yaml`의 `common_spec`에 기록한다.

- Branch 이름이 아니라 Git Tag 또는 전체 Commit 해시를 기록한다.
- 새 버전은 소유자가 다시 확인하라고 요청할 때만 읽는다. 바뀐 점과 반영안을 보여주고, 소유자 승인 뒤 반영한다.
- 충돌이 발견되면 AI Agent가 임의 해석하지 않고 프로젝트 소유자에게 보고한다.

## 2. 프로젝트 SSoT

- 프로젝트 메타데이터: `project.yaml`
- AI Agent 규칙: `.claude/CLAUDE.md`
- 현재 공통 규칙: 이 문서
- 지표 정의와 조사 결과: `specification/10.indicators/README.md`
- 중요한 결정: `records/07.decision/decisions/`
- 안내서: `docs/`
- 현재 실행 작업: GitHub Issues
- 변경 이력: Git

## 3. 사람과 AI Agent

AI Agent가 할 수 있는 일:

- 조사, 분석, 대안 제시
- 문서와 코드 초안 작성
- 테스트와 누락·충돌 확인
- Decision 초안 작성

AI Agent가 독자적으로 할 수 없는 일:

- 결정을 `decided` 또는 `rejected`로 확정
- 위험 수용
- 확정된 결정의 의미 변경
- 현재 적용 규칙 변경
- 복구하기 어려운 작업 실행
- commit, push, PR 생성, merge (사람이 요청하면 PR 생성까지 가능, merge는 하지 않음)
- `b-proto/` 약속서 변경 (소유자 승인 후 가능)
- 개발 도구 설치 (안내만 함)

## 4. 문서 원칙

- YAML 필드와 구조화 값은 영어를 사용한다.
- 본문은 한국어를 기본으로 한다.
- 같은 정보를 여러 문서에서 독립적으로 관리하지 않는다.
- 문서는 필요할 때만 추가한다.
- 새로운 문서는 기존 문서로 관리하기 어려워졌을 때 생성한다.

## 5. 최소 상태

일반 프로젝트 문서:

```yaml
status: draft | approved | retired
```

지표 조사:

```yaml
status: pending | verified | excluded
```

결정 상태는 Decision Specification을 따른다.

## 6. 중요한 제한

- API 조사 전 물리 DB 스키마를 확정하지 않는다.
- 확인되지 않은 지표 코드를 사실처럼 기록하지 않는다.
- 원자료와 서비스 계산값을 구분한다.
- 공식 근거 없는 집계를 구현하지 않는다.
- API 키는 로그, 에러, 샘플에 기록할 때 가린다.
- 키가 노출되면 재발급이 먼저이고, 기록 정리는 그다음이다.
