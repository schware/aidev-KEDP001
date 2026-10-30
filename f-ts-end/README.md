# f-ts-end

사용자 화면과 관리자 화면. React + Vite, Tailwind CSS, Apache ECharts(KEDP001-D016).
데이터는 Go 조회 API와 관리자 API(`specification/40.api/README.md`)에서만 받는다.

| 주소 | 화면 |
| --- | --- |
| `/` | 메인 화면: 지표 카드(50.web 2장) |
| `/indicator/{id}` | 지표 상세(50.web 3장) |
| `/admin/login` | 관리자 로그인 |
| `/admin/dashboard` | Dashboard 관리: 노출·순서, 지표 편집, 지표 추가(50.web 5장) |
| `/admin/api` | API 관리: 수집 시각, 제공처 상태(50.web 6장) |

- 숫자는 지표의 소수 자릿수(`decimals`)로 보인다. `null`이면 받은 값 그대로(소수 4자리까지)다.

## 관리자 화면

- 사용자 화면에는 관리자 링크가 없다. 주소(`/admin/...`)로 바로 들어간다. `/admin`은 `/admin/dashboard`로 간다.
- 들어올 때 `GET /api/admin/session`으로 로그인 상태를 본다. 401이면 로그인 화면으로, 403(허용 IP 아님)이면 "허용된 주소가 아닙니다"만 보인다. 화면을 쓰는 중에 401/403을 받아도 같다.
- session은 서버가 준 cookie(`HttpOnly`)만 쓴다. 비밀번호와 session을 브라우저 저장소에 두지 않는다.
- 허용 IP는 Go 서버의 `.env`(`KEDP_ADMIN_ALLOW_IPS`)로 정한다. 개발 서버의 `/api` proxy는 `xfwd: true`라서 브라우저 주소가 Go로 넘어간다(40.api 5.1).
- Dashboard 관리의 노출·순서는 [저장]을 눌러야 저장된다. 지표 [편집]과 [지표 추가]는 창에서 따로 저장한다.
- API 400의 칸 오류는 그 입력칸 아래에 보인다. 입력칸이 없는 칸의 오류는 창 아래에 모아 보인다.
- 지표 삭제, 재수집, 연결 시험, API 키 입력은 없다(50.web 6장 향후 확장).

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

## 빌드

```sh
npm run build      # dist/에 만든다
npm run preview    # 만든 결과를 띄워 본다(http://127.0.0.1:4173, /api proxy는 dev와 같다)
```

## 검사와 테스트

```sh
npm run lint
npm test           # 기간·숫자(소수 자릿수)·시각 표시, 차트 기간 변환, 제공처별 입력칸 검사, 401/403 처리
```
