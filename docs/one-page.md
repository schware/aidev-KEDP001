# KEDP: 한국 경제 데이터 플랫폼

**공식 경제 통계를 하루 한 번 모아 한 화면에 보여 주는 웹 서비스.** AI Agent(Claude Code)와 함께 열흘 만에 MVP를 만들고, 실행 서버에서 운영 중이다.

- 저장소: https://github.com/schware/aidev-KEDP001
- 기간: 2026-10-22 ~ 2026-11-01 MVP, 이후 지표 확장
- 자세한 사례: [case-study.md](case-study.md)

## 맡은 역할

프로젝트 소유자 1명. 범위와 결정 확정, Issue 설계, AI Agent 작업 분배와 검토, merge, 서버 운영과 배포를 맡았다. 코드는 AI Agent가 Issue 단위로 구현했다.

## 기술

TypeScript(React, Vite, Tailwind CSS, ECharts) · Go · Rust(tonic gRPC) · PostgreSQL 17 · Docker, nginx · GitHub Actions(gitleaks)

## 숫자 (2026-11-08)

| 지표 | 결정 기록 | Issue | merge된 PR |
| --- | --- | --- | --- |
| 42개 (제공처 3곳, 구분 9개) | 24개 | 32개 | 38개 |

## 핵심 성과

1. **AI와 일하는 방식을 규칙으로 만들었다.** 역할 5개로 나누고, 규칙 파일, 다섯 칸 Issue(할 일, 고쳐도 되는 곳, 정해진 것, 끝 기준, 하지 말 것), 결정 기록을 두었다. 사람은 결정하고 Agent는 구현한다. PR 충돌은 한 번도 없었다.
2. **실제 데이터를 먼저 보고 설계했다.** 샘플 수집 도구로 ECOS, KOSIS, 공공데이터포털의 실제 응답을 모은 뒤 DB를 정했다. 지표를 7개에서 42개로 늘릴 때도 id와 화면 주소는 바뀌지 않았다.
3. **처음부터 안전하게 운영했다.** API 키를 5겹으로 보호해 저장소에 키가 들어간 적이 없다. 관리자는 허용 IP와 계정으로 막고, 방문자 수는 IP를 저장하지 않고 센다. 개발 서버와 실행 서버를 나눠 Docker로 배포한다.

---

## English summary

- **KEDP** collects official Korean economic statistics (Bank of Korea ECOS, Statistics Korea KOSIS, data.go.kr) once a day and shows 42 indicators on one dashboard.
- Built as a solo project owner with AI coding agents (Claude Code): an MVP in 10 days, now running on a separate server with Docker (nginx, Go, Rust gRPC, PostgreSQL).
- I set the scope, made 24 recorded decisions, defined 32 structured issues, reviewed and merged every PR, and ran the deployment. The agents implemented each issue in its own role and folder.
- Data shapes were verified with real API samples before the database was designed. API keys never entered the repository (pre-commit hooks and CI secret scanning).
