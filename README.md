# 국내 경제 데이터 플랫폼

- 상태: draft
- 현재 단계: API 조사 전 준비
- 프로젝트 ID: `KEDP001` 후보

## 목적

한국은행 ECOS와 KOSIS가 무상으로 제공하는 국내 공식 통계를 수집하여 사용자가 지표를 조회하고 비교할 수 있는 웹 플랫폼을 개발한다.

## 기본 원칙

1. 데이터 제공처는 ECOS와 KOSIS로 제한한다.
2. 유료 데이터, 비공식 API 및 웹 스크래핑은 사용하지 않는다.
3. 실제 API 응답을 확인한 뒤 데이터베이스를 설계한다.
4. 공식기관이 제공하는 값을 우선 사용한다.
5. 근거가 확인되지 않은 합계, 평균, 보간 또는 가공은 하지 않는다.
6. 일반 사용자 회원 기능과 다운로드 기능은 만들지 않는다.
7. 배포는 Docker를 기준으로 한다.
8. 필요한 문서만 만들고, 실제 반복 문제나 변경 위험이 확인될 때 확장한다.

## 기술

- TypeScript: 공개 웹 화면과 이후 필요한 최소 관리자 화면
- Go: ECOS·KOSIS 호출, 수집 흐름 및 REST API
- Rust: 응답 구조 분석, 정규화, 검증 및 승인된 데이터 처리
- PostgreSQL: API 조사 이후 설계할 데이터 저장소

## 개발 순서

```text
API 키 준비
→ 대표 지표 8개 조사
→ key·value·단위·기간·차원 확인
→ 저장 방식 결정
→ DB 설계
→ 기준금리 수직 기능 구현
→ 대표 지표 확대
```

## 문서 구조

- `project.yaml`: 프로젝트 공통 메타데이터
- `AGENTS.md`: AI Agent 역할과 최소 공통 규칙
- `TODO.md`: 현재 실행 작업
- `specification/00.common/README.md`: 프로젝트의 Common Specification 적용 규칙
- `specification/10.indicators/README.md`: 지표와 API 조사 결과
- `records/07.decision/README.md`: Decision Specification 적용 규칙
- `records/07.decision/decisions/`: 개별 결정 기록

## 아직 확정하지 않은 것

- 프로젝트 ID와 정식 명칭
- DB 물리 스키마
- ECOS·KOSIS 저장 방식
- Go와 Rust의 물리적 연결 방식
- 관리자 화면 상세 범위
- 첫 공개 지표 수
