# Project Decision Specification

- 상태: draft
- 적용 기준: 프로젝트가 채택한 Decision Specification

## 1. 원칙

- 중요한 결정 하나당 Markdown 파일 하나를 사용한다.
- 제안과 확정 결정을 구분한다.
- AI Agent는 초안을 작성할 수 있지만 사람이 최종 확정한다.
- 결정 파일 이름은 결정 ID와 일치시킨다.
- 확정된 결론을 바꾸려면 기존 파일을 덮어쓰지 않고 새 결정으로 기록한다.

## 2. 기록 대상

다음 중 하나에 해당하면 Decision으로 기록한다.

- 여러 대안 중 하나를 선택한다.
- 되돌리기 어렵거나 영향 범위가 크다.
- 여러 Agent가 같은 결정을 알아야 한다.
- 나중에 선택 이유를 다시 확인할 가능성이 크다.
- 기존 결정을 대체한다.

사소한 파일명, 함수명, 화면 배치, 단순 TODO는 Decision으로 만들지 않는다.

## 3. 프로젝트 적용 필드

```yaml
---
id: KEDP001-D001
title: "결정 제목"
status: proposed
drafted_by: mixed
created_at: "2026-10-22"
risk_level: low
replaces: []
issues: []
---
```

`decided` 상태에서는 다음 값을 추가한다.

```yaml
decided_at: "YYYY-MM-DD"
decided_by: "person-id"
decision_owner: "project-owner"
```

## 4. 본문 최소 구조

```markdown
# 결정 제목

## 결론

## 배경

## 검토한 대안

## 결정 이유

## 영향

## 미해결 질문
```

내용이 없는 구역은 억지로 채우지 않는다. 다만 `결론`, `배경`, `결정 이유`, `영향`은 확정 결정에서 유지한다.

## 5. 상태

- `proposed`
- `decided`
- `deferred`
- `rejected`
- `superseded`

AI Agent는 사람 승인 없이 `decided`, `rejected`, `superseded`로 변경하지 않는다.
