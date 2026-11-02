# f-ts-end

사용자 화면과 관리자 화면. React + Vite, Tailwind CSS, Apache ECharts(KEDP001-D016).
데이터는 Go 조회 API와 관리자 API(`specification/40.api/README.md`)에서만 받는다.

| 주소 | 화면 |
| --- | --- |
| `/` | 메인 화면: 구분별로 묶은 지표 카드(50.web 2장) |
| `/indicator/{id}` | 지표 상세(50.web 3장) |
| `/admin/login` | 관리자 로그인 |
| `/admin/dashboard` | Dashboard 관리: 노출·순서, 지표 편집, 지표 추가, 구분(50.web 5장) |
| `/admin/api` | API 관리: 수집 시각, 제공처 상태(50.web 6장) |
| `/admin/visits` | 접속 통계: 오늘·전체·시작값, 최근 30일 막대 차트, 시작값 바꾸기, 0부터 다시 시작(Issue #50) |

- 메인 화면은 API가 준 순서대로 구분 제목 아래에 카드를 묶는다(`src/category.ts`). 다시 정렬하지 않고, 카드가 없는 구분은 보이지 않는다.
- 숫자는 지표의 소수 자릿수(`decimals`)로 보인다. `null`이면 받은 값 그대로(소수 4자리까지)다.

## 방문자 표시 (사용자 화면, 40.api 6장)

- 메인과 상세 화면 아래쪽(footer)에 "오늘 N명 · 전체 N명"을 보인다. 숫자는 천 단위 쉼표다. 코드는 `src/visits.ts`, `src/components/VisitFooter.tsx`다.
- 페이지가 처음 열릴 때 `POST /api/visits`를 한 번 보내고, 그다음 `GET /api/visits/summary`를 읽는다(오늘 숫자에 자기가 들어간다).
- `localStorage`의 `kedp-visit-date`에 Asia/Seoul 오늘 날짜(`YYYY-MM-DD`)가 있으면 POST를 보내지 않는다. POST가 성공한 뒤 날짜를 적는다. `localStorage`에는 이 날짜만 둔다.
- `localStorage`를 쓸 수 없으면(사생활 보호 모드 등) 그냥 보낸다. 같은 사람은 Go가 거른다.
- 화면 안에서 메인과 상세를 오가도 다시 보내지 않는다. 새로고침하면 summary를 다시 읽는다(POST는 날짜를 보고 정한다).
- 방문자 API가 실패하면(POST, summary 어느 쪽이든) footer 숫자만 보이지 않는다. 화면의 다른 부분은 그대로다. POST가 실패하면 날짜를 적지 않아, 다음에 열 때 다시 보낸다.
- 관리자 화면(`/admin/...`)은 이 footer를 쓰지 않으므로 POST를 보내지 않는다.

## 관리자 화면

- 사용자 화면에는 관리자 링크가 없다. 주소(`/admin/...`)로 바로 들어간다. `/admin`은 `/admin/dashboard`로 간다.
- 들어올 때 `GET /api/admin/session`으로 로그인 상태를 본다. 401이면 로그인 화면으로, 403(허용 IP 아님)이면 "허용된 주소가 아닙니다"만 보인다. 화면을 쓰는 중에 401/403을 받아도 같다.
- session은 서버가 준 cookie(`HttpOnly`)만 쓴다. 비밀번호와 session을 브라우저 저장소에 두지 않는다.
- 허용 IP는 Go 서버의 `.env`(`KEDP_ADMIN_ALLOW_IPS`)로 정한다. 개발 서버의 `/api` proxy는 `xfwd: true`라서 브라우저 주소가 Go로 넘어간다(40.api 5.1).
- 지표 [편집]과 [지표 추가]에서 구분을 꼭 고른다. 목록은 `GET /api/admin/categories`다. 이 목록을 읽지 못하면 Dashboard 관리는 "다시 읽기"만 보인다.
- Dashboard 관리의 노출·순서는 [저장]을 눌러야 저장된다. 지표 [편집]과 [지표 추가]는 창에서 따로 저장한다.
- API 400의 칸 오류는 그 입력칸 아래에 보인다. 입력칸이 없는 칸의 오류는 창 아래에 모아 보인다.
- 지표 삭제, 재수집, 연결 시험, API 키 입력은 없다(50.web 6장 향후 확장).
- 접속 통계(`/admin/visits`): 오늘, 전체(시작값 + 일별 합), 시작값과 최근 30일 일별 막대 차트(ECharts)를 보인다. 시작값은 0 이상 정수(쉼표 없이)로 바꾼다. [0부터 다시 시작]은 "지금까지의 일별 기록이 지워집니다" 확인을 받은 뒤 일별 기록을 지우고 시작값을 0으로 한다. 되돌릴 수 없다.

명령은 모두 `f-ts-end` 폴더에서 실행한다. Node 24에서 확인했다.

## 설치

```sh
npm ci
```

## 개발 서버

```sh
npm run dev        # http://127.0.0.1:5173, /api는 Go 조회 API(127.0.0.1:8080)로 넘긴다
npm run dev:mock   # Go 서버 없이 mock/의 가짜 데이터로 띄운다
```

- Go 조회 API를 먼저 띄운다(`b-go-end/README.md`의 `cmd/server`).
- `dev:mock`의 데이터는 40.api 응답 예시로 만들었다. `/indicator/fake-500`은 500 오류 화면, 없는 id는 404 화면을 보여준다.
- `dev:mock`의 관리자 API(`mock/admin.ts`)는 계정·비밀번호를 확인하지 않는다. 둘 다 넣으면 로그인되고, 계정이 `fail`이면 로그인 실패다.
  관리자 화면에서 바꾼 것은 사용자 화면에도 보인다(dev 서버를 다시 켜면 처음으로 돌아간다).
- `MOCK_ADMIN=forbidden npm run dev:mock`으로 띄우면 관리자 API가 모두 403이다(허용 IP가 아닐 때 화면).
- `dev:mock`의 방문자 API(`mock/visits.ts`)는 같은 사람을 거르지 않는다. POST가 올 때마다 오늘 숫자가 1 늘어난다. `MOCK_VISITS=error npm run dev:mock`이면 방문자 API가 500이다(footer 숫자가 숨는 화면).

## 빌드

```sh
npm run build      # dist/에 만든다
npm run preview    # 만든 결과를 띄워 본다(http://127.0.0.1:4173, /api proxy는 dev와 같다)
```

## 검사와 테스트

```sh
npm run lint
npm test           # 기간·숫자(소수 자릿수)·시각 표시, 차트 기간 변환, 제공처별 입력칸 검사, 401/403 처리,
                   # 방문 날짜 판단(Asia/Seoul 자정), localStorage를 못 쓸 때, 방문자 API 실패 때 숨김, 시작값 입력 검사
```
