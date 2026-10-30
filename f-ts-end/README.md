# f-ts-end

사용자 화면. React + Vite, Tailwind CSS, Apache ECharts(KEDP001-D016).
데이터는 Go 조회 API(`specification/40.api/README.md`)에서만 받는다.

| 주소 | 화면 |
| --- | --- |
| `/` | 메인 화면: 지표 카드(50.web 2장) |
| `/indicator/{id}` | 지표 상세(50.web 3장) |

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

## 빌드

```sh
npm run build      # dist/에 만든다
npm run preview    # 만든 결과를 띄워 본다(http://127.0.0.1:4173, /api proxy는 dev와 같다)
```

## 검사와 테스트

```sh
npm run lint
npm test           # 기간·숫자·시각 표시(src/format.test.ts)
```
