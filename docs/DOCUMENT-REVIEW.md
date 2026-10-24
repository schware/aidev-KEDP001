# 문서 구성 비판적 검토

## 검토 결론

문서 체계를 너무 줄이면 AI Agent 협업에서 공통 기준과 결정 이력이 사라지고, 너무 늘리면 기능보다 문서 관리가 커진다. 이번 구성은 두 문제 사이의 최소 균형을 목표로 한다.

## 유지해야 하는 것

- Common Specification 적용 기준
- Decision Specification과 개별 결정 이력
- 프로젝트 메타데이터 SSoT
- Agent 역할 경계
- 지표별 API 조사 결과
- 현재 실행 작업

## 제거한 것

- 내용이 없는 계약 placeholder
- Agent별 장문의 독립 명세
- 별도 Roadmap과 Phase Gate 문서
- 복잡한 검토·Git 절차 문서
- API 조사 전 DB·집계 상세 문서
- 지표 25개별 빈 파일

## 중복 방지 원칙

- 프로젝트 방향은 `README.md`에서만 요약한다.
- 프로젝트 메타데이터는 `project.yaml`만 수정한다.
- Agent 역할은 `AGENTS.md`에서 관리한다.
- 지표 사실은 Indicator Specification에서 관리한다.
- 중요한 선택과 이유는 Decision 기록에서 관리한다.
- 작업 진행은 `TODO.md`에서만 관리한다.

## 남아 있는 위험

- `KEDP001`과 프로젝트명은 아직 후보값이다.
- 외부 Common Specification의 실제 Git 참조가 비어 있다.
- 초기 Decision은 `proposed`이며 사람의 상태 확정이 필요하다.
- 실제 API 조사 전이므로 저장 방식과 처리 계약은 의도적으로 문서화하지 않았다.

## 문서 확장 조건

다음 중 하나가 실제로 발생할 때만 새 Specification을 추가한다.

1. 기존 문서가 커져 정보를 찾기 어렵다.
2. 둘 이상의 Agent가 반복적으로 같은 인터페이스를 해석한다.
3. 같은 문제가 두 번 이상 발생한다.
4. 되돌리기 어려운 새 결정이 필요하다.
5. 실제 코드 경계에 명시적 계약이 필요하다.

예상 확장 순서:

```text
20.database     API 조사와 저장 방식 결정 이후
30.processing   Go-Rust 처리 방식 결정 이후
40.api          공개 REST API 설계 시 OpenAPI 중심으로 추가
```
