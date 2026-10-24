# TODO

## 1. 시작 전 확인

- [ ] `KEDP001`을 프로젝트 ID로 확정한다.
- [ ] 한글·영문 프로젝트명을 확정한다.
- [ ] 저장소 이름을 확정한다.
- [ ] Common Specification의 Tag 또는 전체 Commit 해시를 `project.yaml`에 기록한다.
- [ ] 초기 Decision 초안을 검토하고 필요한 항목을 `decided`로 확정한다.

## 2. API 접근 준비

- [ ] ECOS API 키 준비
- [ ] KOSIS API 키 준비
- [ ] `.env.example` 작성
- [ ] 실제 비밀값 Git 제외

## 3. 대표 지표 조사

### ECOS

- [ ] 한국은행 기준금리
- [ ] 소비자물가지수
- [ ] 실질 국내총생산
- [ ] 외환보유액

### KOSIS

- [ ] 고용률
- [ ] 출생아 수
- [ ] 총인구
- [ ] 전산업 사업체 수

각 지표에서 확인:

- [ ] 공식 API와 요청 파라미터
- [ ] 실제 정상 응답 샘플
- [ ] 기간 key
- [ ] 값 key
- [ ] 단위 key
- [ ] 항목·분류 key
- [ ] 논리 key 후보

## 4. API 조사 이후

- [ ] ECOS와 KOSIS 구조 비교
- [ ] 저장 방식 결정
- [ ] DB 논리 모델 작성
- [ ] DB 물리 스키마와 migration 작성
- [ ] Go-Rust 처리 방식 결정

## 5. 첫 수직 기능

대상: 한국은행 기준금리

- [ ] Go 수집
- [ ] Rust 파싱·검증
- [ ] DB 저장
- [ ] Go 조회 API
- [ ] TypeScript 차트와 표
- [ ] 출처와 수집 시각 표시
- [ ] Docker 빌드
- [ ] 전체 흐름 테스트
