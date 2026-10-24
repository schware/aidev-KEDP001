# 한국 경제 데이터 플랫폼

- 상태: draft
- 현재 단계: API 조사
- 프로젝트 ID: `KEDP001`

## 목적

한국은행 ECOS, KOSIS, 공공데이터포털이 무상으로 제공하는 국내 공식 통계와 ETF 시세를 수집하여, 사용자가 지표를 조회하고 비교할 수 있는 웹 플랫폼을 개발한다.

## 기본 원칙

1. 데이터 제공처는 ECOS, KOSIS, 공공데이터포털(금융위원회_증권상품시세정보)로 제한한다.
2. 유료 데이터, 증권사 API, 비공식 API 및 웹 스크래핑은 사용하지 않는다.
3. 실제 API 응답을 확인한 뒤 데이터베이스를 설계한다.
4. 공식기관이 제공하는 값을 우선 사용한다.
5. 근거가 확인되지 않은 합계, 평균, 보간 또는 가공은 하지 않는다.
6. 일반 사용자 회원 기능과 다운로드 기능은 만들지 않는다.
7. 배포는 Docker를 기준으로 한다.
8. 필요한 문서만 만들고, 실제 반복 문제나 변경 위험이 확인될 때 확장한다.

## 기술

- TypeScript: 공개 웹 화면과 이후 필요한 최소 관리자 화면
- Go: ECOS·KOSIS·공공데이터포털 호출, 수집 흐름, REST API, Rust 서버 호출(gRPC)
- Rust: 응답 구조 분석, 정규화, 검증. Go의 요청을 gRPC로 받는 서버로 동작
- PostgreSQL: API 조사 이후 설계할 데이터 저장소

## 저장소 구조

```text
aidev-kedp001/
├── README.md
├── project.yaml          프로젝트 기본 정보
├── .claude/              AI Agent 규칙 (CLAUDE.md, api.md, git.md, environment.md)
├── docs/                 안내서와 참고 자료 (v0.1 기록 포함)
├── specification/        명세
├── records/              결정 기록
├── samples/              API 응답 샘플
├── b-go-end/             Go: 수집 + REST API
├── b-proto/              Go–Rust 약속서 (소유자 승인 후 변경)
├── b-rust-processor/     Rust: 분석·검증 gRPC 서버
├── b-sql-db/             SQL: 스키마, migration
└── f-ts-end/             TypeScript: 화면
```

코드 폴더(`b-*`, `f-*`)는 해당 작업을 시작할 때 만든다.

## 개발 순서

```text
저장소와 키 보호 구축
→ 대표 지표 조사 (key·value·단위·기간·차원 확인)
→ 저장 방식 결정
→ DB 설계
→ 기준금리 수직 기능 구현
→ 대표 지표 확대
```

## 문서와 작업 관리

| 무엇 | 어디 |
| --- | --- |
| 프로젝트 기본 정보 | `project.yaml` |
| AI Agent 규칙 | `.claude/CLAUDE.md` |
| 공통 규칙 | `specification/00.common/README.md` |
| 지표와 API 조사 결과 | `specification/10.indicators/README.md` |
| 결정 기록 | `records/07.decision/decisions/` |
| 안내서 | `docs/` |
| 작업 목록 | GitHub Issues |

## 처음 시작할 때

clone 뒤 키 보호 설정과 `.env` 준비가 필요하다. `docs/key-protection.md`의 "팀원 시작 안내"를 따른다.

## 아직 확정하지 않은 것

- DB 물리 스키마
- ECOS·KOSIS·공공데이터포털 저장 방식
- 데이터 수집 주기
- 관리자 화면 상세 범위
- 첫 공개 지표 수
