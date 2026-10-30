# KEDP 호출 흐름 (Vite → Go → Rust)

화면에서 서버까지 요청이 어떻게 지나가는지 흐름만 정리한다. 업무 규칙과 DB 구조는 다루지 않는다.

- 자세한 API: `specification/40.api/README.md`
- Go–Rust 약속: `b-proto/README.md`

---

## 1. 한눈에 보기

```text
 브라우저
   │  http://<서버>:5173
   ▼
┌──────────────────────────────┐
│ Vite 개발 서버  :5173        │  f-ts-end
│  - 화면 파일(React) 전달      │
│  - /api/* 는 Go로 넘김(proxy) │
└──────────────┬───────────────┘
               │  /api/*  →  http://127.0.0.1:8080
               ▼
┌──────────────────────────────┐
│ Go 서버  :8080               │  b-go-end (cmd/server)
│  - 조회 API   /api/...        │
│  - 관리자 API /api/admin/...  │
│  - 예약 수집(하루 1회)        │
└──────┬────────────────┬──────┘
       │ gRPC           │ HTTPS
       ▼                ▼
┌──────────────┐   ┌──────────────────────────┐
│ Rust :50051  │   │ ECOS · KOSIS · 공공데이터 │
│ 응답 정리·저장│   │ (외부 제공처)             │
└──────┬───────┘   └──────────────────────────┘
       ▼
      DB
```

| 프로그램 | 폴더 | 주소 | 하는 일 |
| --- | --- | --- | --- |
| Vite | `f-ts-end` | `:5173` | 화면 파일을 브라우저에 주고, `/api` 요청을 Go로 넘긴다 |
| Go | `b-go-end` | `127.0.0.1:8080` | API(조회, 관리자)와 수집 |
| Rust | `b-rust-processor` | `127.0.0.1:50051` | Go가 받은 제공처 응답을 정리해 저장 |

Go와 Rust는 서버 안(`127.0.0.1`)에서만 받는다. 밖에서 들어오는 문은 Vite 하나다.

---

## 2. Vite는 Tomcat인가?

아니다. 개발할 때 쓰는 **화면 파일 전달 + 요청 중계** 서버다.

| | Vite (개발 서버) | Tomcat |
| --- | --- | --- |
| 무엇을 돌리나 | 화면 코드(React, TypeScript)를 브라우저가 읽을 수 있게 바꿔서 준다 | 서버 쪽 프로그램(Servlet, JSP)을 실행한다 |
| 업무 처리 | 하지 않는다. DB도 모른다 | 한다 |
| `/api` 요청 | Go로 그대로 넘긴다(proxy) | 직접 처리한다 |
| 운영(배포)에서 | **쓰지 않는다** | 운영에서도 쓴다 |

KEDP에서 Tomcat과 비슷한 자리는 **Go 서버**다. 요청을 받아 처리하고 응답을 만든다.

### 개발과 운영의 차이

```text
개발:  브라우저 → Vite(:5173) ─ 화면 파일 (고치면 바로 반영)
                        └─ /api → Go(:8080)

운영:  npm run build → dist/ (HTML, JS, CSS 정적 파일)
       브라우저 → 웹 서버(예: nginx) ─ dist/ 파일
                              └─ /api → Go(:8080)
```

- 개발 중에는 Vite가 코드가 바뀔 때마다 브라우저를 바로 새로 그려 준다(HMR).
- 운영에서는 `npm run build`로 만든 정적 파일만 있으면 된다. Vite 프로그램은 돌지 않는다. 배포 방법은 배포 Issue에서 정한다.

### Vite proxy 설정

`f-ts-end/vite.config.ts`:

```ts
proxy: { '/api': { target: 'http://127.0.0.1:8080', xfwd: true } }
```

- `/api`로 시작하는 요청만 Go로 넘긴다. 나머지는 Vite가 화면 파일로 답한다.
- 브라우저는 Vite 주소 하나만 보므로 CORS 설정이 필요 없다.
- `xfwd: true`: 요청한 쪽의 IP를 `X-Forwarded-For`에 담아 넘긴다. Go 관리자 API가 허용 IP를 볼 때 쓴다.
- `npm run dev:mock`으로 띄우면 Go 없이 Vite가 가짜 데이터로 답한다.

---

## 3. 흐름 1: 사용자가 화면을 볼 때

```text
1. 브라우저 → Vite      GET /                       화면 파일(HTML, JS)을 받는다
2. 브라우저(React)가 화면을 그리면서 데이터를 요청한다
3. 브라우저 → Vite      GET /api/indicators
4. Vite → Go            GET /api/indicators          (proxy)
5. Go → DB              지표와 최신값을 읽는다
6. Go → Vite → 브라우저  JSON
7. 카드를 누르면         GET /api/indicators/{id}     같은 길로 상세 데이터
```

- 화면 주소(`/indicator/ecos-base-rate`)는 브라우저 안에서 React가 바꾼다. 서버에 새 페이지를 요청하지 않는다.
- 이 흐름에서는 Rust를 부르지 않는다.

## 4. 흐름 2: 관리자

```text
1. 브라우저 → Vite → Go   POST /api/admin/login
2. Go: 허용 IP 확인 → 계정 파일로 비밀번호 확인 → session을 만들고 cookie를 준다
3. 이후 요청(지표 수정, 순서 저장, 수집 시각 등)에 브라우저가 cookie를 붙여 보낸다
4. Go: 허용 IP 확인 → cookie로 로그인 확인 → 처리
```

- 허용 IP가 아니면 403, 로그인하지 않았으면 401이다.
- 바꾼 설정은 DB에 들어가고, 사용자 화면과 수집이 다음부터 따른다.

## 5. 흐름 3: 수집 (화면과 관계없음)

브라우저 요청이 없어도 Go가 스스로 돈다.

```text
Go 시작 → Rust Check(gRPC)     Rust가 없으면 Go가 시작하지 않는다
   │
   └ 30초마다 수집 시각(기본 06:00)이 됐는지 확인
        │ 됐으면 지표마다:
        ├ Go → 제공처(ECOS·KOSIS·공공데이터포털)   HTTPS, 응답 받기
        ├ Go → Rust  ProcessPage(gRPC)            받은 응답을 그대로 넘김
        ├ Rust: 응답 정리·검증 → DB 저장 → 결과(건수, 상태)를 Go에 돌려줌
        └ Go: 수집 기록을 남김
```

- 외부 제공처를 부르는 것은 Go뿐이다. Rust는 외부를 부르지 않는다.
- 화면(Vite)은 제공처나 Rust를 직접 부르지 않는다. 항상 Go를 거친다.

---

## 6. 띄우는 순서

```text
1. DB          docker start kedp-postgres
2. Rust        cd b-rust-processor && CARGO_BUILD_JOBS=2 cargo run --release
3. Go          cd b-go-end && go run ./cmd/server
4. Vite        cd f-ts-end && npm run dev           (외부 접속: npm run dev -- --host 0.0.0.0)
```

- Go는 시작할 때 Rust를 확인하므로 Rust를 먼저 띄운다.
- 화면만 볼 때는 `npm run dev:mock`으로 Vite만 띄워도 된다.
