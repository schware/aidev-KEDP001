# KEDP Docker 배포 안내서

- 상태: draft
- 대상: Sun&Moon(소유자)
- 근거: KEDP001-D002(Docker), D012(키 보호), D018(관리자 접근), D020(SSH 배포, 백업), D023(배포 방식)

이 문서 하나로 **image 만들기부터 실행 서버 배포, 백업, 다음 배포까지** 따라 할 수 있다. 단계마다 "왜 하는지"를 먼저 쓰고 명령을 쓴다.

- 서버 이름: **dell-debian** = 개발 서버, **macpro-debian** = 실행 서버
- `macpro-debian`은 dell-debian `~/.ssh/config`의 Host 이름이다. 실제 주소는 이 문서에 쓰지 않는다.
- `<꺾쇠>`는 그 자리에 실제 값을 넣으라는 뜻이다. 꺾쇠는 쓰지 않는다. 예: `<tag>` → `17a8351`
- 실제 주소, 계정, 비밀번호, 키는 이 문서에 쓰지 않는다.

---

# 1. 큰 그림

```text
 dell-debian (개발 서버)                         macpro-debian (실행 서버)
 ───────────────────────                         ──────────────────────────
 소스 코드                                       소스 코드 없음
 /srv/projects/aidev-kedp001                     /srv/projects/kedp/
    │                                              ├─ compose.yaml  무엇을 어떻게 띄울지
    │ ① deploy/build.sh                            ├─ .env          키, 비밀번호, 설정
    ▼                                              ├─ admin         관리자 계정 파일
 image 3개                                         └─ backup.sh     DB 백업
  kedp-go:<tag>     ── ② deploy/send.sh ──►    /srv/projects/kedp-backup/  백업 파일
  kedp-rust:<tag>      docker save                      │
  kedp-nginx:<tag>     → SSH → docker load              │ ③ docker compose up -d
                                                        ▼
                                                 container 4개가 돈다
                                                  nginx · go · rust · postgres
```

- dell-debian은 **만드는 곳**이다. 소스에서 image를 만든다.
- macpro-debian은 **돌리는 곳**이다. 받은 image를 실행만 한다. Go, Rust, Node를 설치하지 않는다.
- 둘 사이는 SSH로 image를 보낸다. GitHub 같은 바깥 저장소(registry)는 쓰지 않는다.

---

# 2. 꼭 알아야 할 낱말

| 낱말 | 쉽게 말하면 | KEDP에서 |
| --- | --- | --- |
| **image** | 프로그램과 실행에 필요한 것을 묶은 **설치 파일**. 한 번 만들면 바뀌지 않는다 | `kedp-go`, `kedp-rust`, `kedp-nginx`, `postgres:17` |
| **container** | image를 실행한 것. **돌고 있는 프로그램** 하나 | `kedp-go-1` 등 4개 |
| **tag** | image의 **버전 이름** | `17a8351`(그때의 git commit 번호) |
| **Dockerfile** | image를 만드는 **조리법** | `b-go-end/Dockerfile`, `b-rust-processor/Dockerfile`, `f-ts-end/Dockerfile` |
| **compose** | container 여러 개를 **한 번에** 띄우는 설정과 명령 | `compose.yaml`, `docker compose ...` |
| **volume** | container를 지워도 **남는 저장 공간** | `kedp_pgdata`(DB 데이터) |
| **port** | 바깥에서 들어오는 **문 번호** | nginx 하나만 열려 있다(지금 8090) |

image와 container는 "설치 파일"과 "실행 중인 프로그램"의 관계다. container는 지웠다 다시 만들어도 똑같이 돈다. **데이터는 container가 아니라 volume에 있기 때문이다.** 그래서 container를 다시 만드는 것을 겁낼 필요가 없다.

---

# 3. container 4개와 요청이 흐르는 길

```text
       브라우저 (Windows PC)
           │  http://<macpro-debian 주소>:8090/
           ▼
 ┌──────────────────────┐
 │ nginx                │  문지기. 바깥에 열린 유일한 문
 │  · 화면 파일을 준다  │  (/, /indicator/..., /admin/...)
 │  · /api 는 go로 넘김 │
 └─────────┬────────────┘
 ══════════╪═══════════════════ 여기부터는 바깥에서 들어올 수 없다 (내부 network)
           ▼
 ┌──────────────────────┐   gRPC    ┌──────────────────────┐
 │ go                   │ ────────► │ rust                 │
 │  · 조회/관리자 API   │           │  · 응답 정리·검증    │
 │  · 매일 수집         │           │  · 관측값 저장       │
 └─────────┬────────────┘           └─────────┬────────────┘
           ▼                                  ▼
 ┌─────────────────────────────────────────────────────────┐
 │ postgres   데이터는 volume(kedp_pgdata)에 있다          │
 └─────────────────────────────────────────────────────────┘
```

| container | 하는 일 | 바깥에서 접속 |
| --- | --- | --- |
| nginx | 화면을 주고, `/api` 요청을 go로 넘긴다 | **된다**(포트 하나) |
| go | API, 매일 수집, 관리자 기능. API 키는 여기에만 있다 | 안 된다 |
| rust | 받은 응답을 정리해서 DB에 저장 | 안 된다 |
| postgres | DB | 안 된다 |

개발할 때는 Vite가 nginx 자리에 있었다. Vite도 화면을 주고 `/api`를 go로 넘겼다. **배포에서는 같은 일을 nginx가 한다.** 그래서 화면 코드는 바뀌지 않았다.

---

# 4. 파일 소개

저장소 `deploy/` 폴더:

| 파일 | 하는 일 | 어디서 쓰나 |
| --- | --- | --- |
| `build.sh` | image 3개를 **하나씩** 만든다 | dell-debian |
| `send.sh` | image 3개를 SSH로 보낸다 | dell-debian |
| `compose.yaml` | container 4개를 어떻게 띄울지 | macpro-debian으로 복사 |
| `backup.sh` | DB 백업, 14일 지난 백업 지우기 | macpro-debian으로 복사 |
| `.env.example` | `.env`에 넣을 이름 목록(값 없음) | 참고용 |
| `nginx/default.conf` | nginx 설정 | `kedp-nginx` image 안에 들어간다 |

macpro-debian에만 있는 파일(저장소에 없다):

| 파일 | 내용 | 권한 |
| --- | --- | --- |
| `/srv/projects/kedp/.env` | 키, DB 비밀번호, 설정 | `600`(나만 읽고 쓰기) |
| `/srv/projects/kedp/admin` | 관리자 계정과 비밀번호 hash | 소유자 `65532`, `400` |

---

# 5. 처음 배포하기

2026-10-31에 이 순서로 옮겼다. 서버를 새로 바꿀 때도 같은 순서다.

## 5.1 macpro-debian 준비

**왜:** container를 띄우려면 Docker가 있어야 하고, compose 파일과 `.env`를 둘 폴더가 있어야 한다. 백업 폴더는 compose 폴더 밖에 따로 둔다.

```bash
# macpro-debian
docker compose version          # 버전이 나오면 Docker와 compose가 있다
docker ps                       # 오류 없이 표가 나오면 docker group에 들어 있다
timedatectl                     # Time zone이 Asia/Seoul인지(백업 03:00 기준)

sudo mkdir -p /srv/projects/kedp /srv/projects/kedp-backup
sudo chown "$USER": /srv/projects/kedp /srv/projects/kedp-backup
chmod 700 /srv/projects/kedp-backup
```

## 5.2 .env 만들기

**왜:** 키와 비밀번호는 image에 넣지 않는다(KEDP001-D012). image가 새어도 키는 새지 않게 하려는 것이다. 대신 실행할 때 compose가 `.env`에서 읽어 container에 넣어 준다.

dell-debian `.env`를 통째로 복사하고 모자란 줄만 더한다. dell-debian `.env`의 주소 값(`KEDP_DB_HOST` 등)은 compose가 덮어쓰므로 그대로 두어도 된다(9.2).

```bash
# dell-debian
scp /srv/projects/aidev-kedp001/.env macpro-debian:/srv/projects/kedp/.env

# macpro-debian
chmod 600 /srv/projects/kedp/.env
nano /srv/projects/kedp/.env
```

`KEDP_ADMIN_ALLOW_IPS=`는 비워 두고(5.10에서 넣는다), 맨 아래에 더한다:

```text
KEDP_PG_SUPERUSER=<새 관리 계정 이름>
KEDP_PG_SUPERUSER_PASSWORD=<새 비밀번호>
KEDP_NET_SUBNET=172.30.46.0/24
KEDP_HTTP_PORT=8090
KEDP_IMAGE_TAG=
KEDP_BACKUP_DIR=/srv/projects/kedp-backup
```

| 이름 | 뜻 |
| --- | --- |
| `ECOS_API_KEY` 외 2개 | 제공처 API 키. dell-debian과 같다 |
| `kedp_go`, `kedp_rust` | go, rust가 DB에 접속할 비밀번호. DB 계정을 그대로 옮기므로 dell-debian과 같다. 사람이 외울 필요 없다 |
| `KEDP_PG_SUPERUSER`, `..._PASSWORD` | postgres 관리 계정. **DB를 처음 만들 때만** 쓰인다. 그 뒤에 바꿔도 DB에 반영되지 않는다 |
| `KEDP_ADMIN_ALLOW_IPS` | 관리자 화면을 열 수 있는 PC 주소(5.10) |
| `KEDP_NET_SUBNET` | container끼리 쓰는 내부 network 대역. `ip route`로 본 서버의 다른 대역과 겹치지 않게 골랐다 |
| `KEDP_HTTP_PORT` | 바깥에 여는 포트. 비우면 80. 80은 Windows PC에서 닿지 않아 8090으로 했다 |
| `KEDP_IMAGE_TAG` | 띄울 image 버전. 5.5에서 넣는다 |
| `KEDP_BACKUP_DIR` | 백업 파일 폴더 |

## 5.3 관리자 계정 파일

**왜:** 관리자 로그인은 서버의 파일 하나로 관리한다(KEDP001-D018). 파일에는 비밀번호 원문이 아니라 **hash**(되돌릴 수 없게 바꾼 값)만 들어간다. hash를 만드는 도구가 Go 프로그램이라서 dell-debian에서 만든다. 이 비밀번호는 사람이 입력하므로 기억할 수 있는 것으로 정한다(12자 이상).

```bash
# dell-debian
cd /srv/projects/aidev-kedp001/b-go-end
go run ./cmd/adminpass -account <계정> > ~/kedp-admin    # 비밀번호를 두 번 입력. 화면에 안 보인다
chmod 600 ~/kedp-admin
scp ~/kedp-admin macpro-debian:/srv/projects/kedp/admin
rm ~/kedp-admin

# macpro-debian: go container만 읽게 한다
sudo chown 65532:65532 /srv/projects/kedp/admin
sudo chmod 400 /srv/projects/kedp/admin
```

`65532`는 go container가 쓰는 사용자 번호다. container는 관리자(root)가 아닌 이 번호로 돈다.

## 5.4 image 만들기 (dell-debian)

**왜:** 실행 서버에는 소스와 도구가 없으므로, 실행에 필요한 것만 묶은 image를 개발 서버에서 만든다.

**image는 2단계로 만들어진다(multi-stage).** Dockerfile 세 개가 모두 이 방식이다.

```text
 1단계: 만드는 곳(build)                  2단계: 실행할 것만 담는 곳
 ─────────────────────────                ─────────────────────────
 Go/Rust/Node 도구 + 소스      ──복사──►   실행 파일 하나 (+ 꼭 필요한 것)
 (크다: 수백 MB~GB)                        (작다: 수 MB)
```

| image | 1단계 | 2단계에 들어가는 것 | 크기(이번) |
| --- | --- | --- | --- |
| kedp-go | Go로 `cmd/server` 컴파일 | 실행 파일, HTTPS 인증서 목록 | 약 25MB |
| kedp-rust | Rust로 처리 서버 컴파일 | 실행 파일 | 약 5MB |
| kedp-nginx | Node로 `npm run build` | nginx + 화면 파일 + nginx 설정 | 약 94MB |

go와 rust의 2단계는 `scratch`(빈 image)다. 셸도 없어서 container 안에 들어가 명령을 칠 수 없다. 대신 공격받을 곳이 적다.

**왜 하나씩 만드나:** Rust 컴파일은 메모리를 많이 쓴다. dell-debian에서 여러 작업(Claude Code 여러 개, 다른 Docker 작업)을 함께 돌리면 서버가 멈춘다. `build.sh`는 go → rust → nginx를 차례로 하나씩 만든다. 처음에는 7분쯤 걸린다.

```bash
# dell-debian. 다른 Claude Code, Docker 작업을 멈춘 뒤
cd /srv/projects/aidev-kedp001
git switch main
git pull
git status --short              # 아무것도 안 나와야 한다. 바뀐 파일이 있으면 tag 끝에 -dirty가 붙는다
deploy/build.sh                 # 마지막 줄 "== done: tag 17a8351"의 tag를 적어 둔다
```

- tag는 그때의 git commit 번호다. 그래서 "어느 코드로 만든 image인지"를 tag만 보고 알 수 있다.
- 하나만 다시 만들 때: `deploy/build.sh <tag> rust`

## 5.5 image 보내기

**왜:** `docker save`는 image를 파일 하나로 묶고, SSH가 그 파일을 보내고, `docker load`가 받는 쪽 Docker에 풀어 넣는다. `send.sh`가 이 세 가지를 한 번에 한다.

```bash
# dell-debian
deploy/send.sh macpro-debian <tag>
scp deploy/compose.yaml deploy/backup.sh macpro-debian:/srv/projects/kedp/

# macpro-debian
chmod 700 /srv/projects/kedp/backup.sh
nano /srv/projects/kedp/.env            # KEDP_IMAGE_TAG=<tag>
docker images 'kedp-*'                  # kedp-nginx, kedp-go, kedp-rust가 같은 tag로 보이면 된다
```

## 5.6 DB 꺼내기 (dell-debian)

**왜:** 지금까지 모은 데이터를 그대로 가져가려고 한다. 두 가지를 따로 꺼낸다.

- **계정(`roles.sql`):** `pg_dump`는 DB 안의 표만 담고 DB 계정(`kedp_go`, `kedp_rust`)은 담지 않는다. 그래서 계정은 따로 꺼낸다. 비밀번호 hash가 들어 있으니 내용을 남에게 보이지 않는다.
- **DB(`kedp.dump`):** 표, 데이터, 권한.

먼저 수집을 멈춘다. 꺼내는 동안 값이 바뀌지 않게 하려는 것이다. 행 수는 옮긴 뒤 같은지 비교하려고 적어 둔다.

```bash
# dell-debian
# 1) 수집 멈춤: Go 서버를 띄운 터미널에서 Ctrl+C
ss -ltn | grep ':8080 ' || echo "go server stopped"

# 2) 옮길 파일 폴더
mkdir -p ~/kedp-move && chmod 700 ~/kedp-move

# 3) 계정 2개. 결과가 4여야 한다(CREATE ROLE, ALTER ROLE × 2)
docker exec kedp-postgres pg_dumpall -U sunmoon --roles-only | grep -E '^(CREATE|ALTER) ROLE kedp_(go|rust)[ ;]' > ~/kedp-move/roles.sql
wc -l < ~/kedp-move/roles.sql

# 4) DB
docker exec kedp-postgres pg_dump -U sunmoon -d kedp -Fc --no-owner > ~/kedp-move/kedp.dump

# 5) 행 수 적어 두기
docker exec kedp-postgres psql -X -U sunmoon -d kedp -At -c "select 'indicator', count(*) from indicator union all select 'observation', count(*) from observation union all select 'collection_run', count(*) from collection_run union all select 'raw_response', count(*) from raw_response"

# 6) 보내기
scp ~/kedp-move/roles.sql ~/kedp-move/kedp.dump macpro-debian:/srv/projects/kedp-backup/
```

2026-10-31 행 수: indicator 7, observation 1003, collection_run 7, raw_response 7

## 5.7 DB 되살리기 (macpro-debian)

**왜 postgres만 먼저 띄우나:** 빈 DB가 먼저 있어야 되살릴 수 있다. go가 먼저 뜨면 빈 DB에 접속해 오류를 낸다.

**왜 계정을 먼저 넣나:** DB의 권한(GRANT)이 `kedp_go`, `kedp_rust`를 가리킨다. 계정이 없으면 권한을 넣다가 실패한다.

```bash
# macpro-debian
cd /srv/projects/kedp
docker compose up -d postgres
docker compose ps postgres              # STATUS에 healthy가 보일 때까지 몇 번 본다

# 계정 → DB 순서. 둘 다 아무것도 안 나오면 성공
docker compose exec -T postgres sh -c 'psql -X -q -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d kedp' < /srv/projects/kedp-backup/roles.sql
docker compose exec -T postgres sh -c 'pg_restore -U "$POSTGRES_USER" -d kedp --no-owner --exit-on-error' < /srv/projects/kedp-backup/kedp.dump

# 행 수 비교: 5.6과 같아야 한다
docker compose exec -T postgres sh -c 'psql -X -At -U "$POSTGRES_USER" -d kedp' <<'SQL'
select 'indicator', count(*) from indicator union all select 'observation', count(*) from observation
union all select 'collection_run', count(*) from collection_run union all select 'raw_response', count(*) from raw_response;
SQL

rm /srv/projects/kedp-backup/roles.sql   # 비밀번호 hash가 들어 있다
```

`$POSTGRES_USER`는 container 안에서 `.env`의 `KEDP_PG_SUPERUSER` 값이다. 그래서 명령에 계정 이름을 직접 쓰지 않아도 된다.

## 5.8 전체 띄우기와 서버 안에서 확인

```bash
# macpro-debian (/srv/projects/kedp)
docker compose up -d
docker compose ps                       # 4개 모두 Up, postgres는 healthy
docker compose logs --tail 20 go rust
```

go 기록에서 볼 것:

- `processor check ok`: go가 rust에 닿았다.
- `server started (API and schedule)`: API가 열리고 매일 수집 예약이 시작됐다.

```bash
curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:8090/              # 200
curl -s http://127.0.0.1:8090/api/indicators | head -c 300; echo             # 지표 JSON
curl -s -w ' %{http_code}\n' http://127.0.0.1:8090/api/admin/session         # {"error":"forbidden"} 403
```

마지막이 403인 것이 **정상**이다. 서버 안에서 부른 요청은 허용 IP가 아니기 때문이다(5.10).

## 5.9 바깥에서 확인 (Windows PC)

**왜:** 서버 안에서 되는 것과 바깥에서 들어올 수 있는 것은 다르다. 그 사이에 방화벽, 공유기, 회선이 있다.

```powershell
# Windows PowerShell
Test-NetConnection <macpro-debian 주소> -Port 8090
```

`TcpTestSucceeded : True`이면 브라우저에서 `http://<macpro-debian 주소>:8090/`에 들어간다.

1. 메인 화면에 카드가 보인다.
2. 카드를 눌러 상세 화면을 열고, **그 화면에서 새로고침**한다. 새로고침해도 화면이 나와야 한다(nginx가 없는 주소에 `index.html`을 준다).

`Waiting for response`에서 멈추면 그 포트가 중간에서 막힌 것이다. 처음에 80이 이랬고, 8090으로 바꿨다. 포트를 바꾸면:

```bash
nano .env                       # KEDP_HTTP_PORT=<새 포트>
docker compose up -d nginx
docker compose ps nginx         # PORTS에 0.0.0.0:<새 포트>->80/tcp
```

## 5.10 관리자 허용 IP 넣기

관리자 화면은 **허용 IP + 로그인** 둘 다 통과해야 열린다(KEDP001-D018).

**문제: go는 브라우저 주소를 직접 볼 수 없다.**

```text
 브라우저 ──► nginx ──► go
 (PC 주소)    nginx는 PC 주소를 안다
              go가 보는 것은 "nginx가 보냈다"뿐이다
```

그래서 nginx가 편지에 메모를 붙이듯 `X-Forwarded-For: <PC 주소>`를 붙여 go에 넘긴다. go는 이 메모를 허용 IP와 비교한다.

- 아무나 메모를 꾸며 보낼 수 있으므로, go는 **내부 network(nginx)에서 온 요청의 메모만 믿는다.** compose가 go에 `KEDP_TRUSTED_PROXIES = KEDP_NET_SUBNET`을 넣어 준다.
- nginx는 브라우저가 보낸 메모를 버리고 **자기가 본 주소로 덮어쓴다.** 그래서 꾸민 메모는 쓰이지 않는다.

**PC 주소는 nginx 기록에서 찾는다.** 서버가 보는 PC 주소는 PC 설정 화면의 주소와 다를 수 있다(공유기, 회선, VPN을 거치면 바뀐다). 서버가 실제로 본 주소가 정답이다.

1. Windows 브라우저로 사용자 화면을 한 번 연다.
2. 기록을 본다.

```bash
docker compose logs --tail 5 nginx
```

```text
nginx-1  | <PC 주소> - - [날짜] "GET / HTTP/1.1" 200 ... "Mozilla/5.0 (Windows NT 10.0 ...
           ▲ 끝에 Windows가 붙은 줄의 맨 앞 주소
```

3. 그 주소를 넣고 go를 다시 만든다.

```bash
nano .env                       # KEDP_ADMIN_ALLOW_IPS=<PC 주소>
docker compose up -d go
```

4. 브라우저에서 `http://<macpro-debian 주소>:8090/admin/login`으로 로그인한다. Dashboard 관리, API 관리가 보이면 된다.

넣으면 안 되는 주소:

- **내부 network 주소(`172.30.46.x`)**: 서버 안에서 `curl`로 부르면 이 주소로 보인다. 넣으면 서버 안의 모든 요청이 허용된다.
- **macpro-debian 자신의 주소**: PC 주소가 아니다.

**Cloudflare WARP:** WARP를 켜면 서버에는 PC 주소 대신 Cloudflare 주소가 보인다. 이 주소는 다른 WARP 사용자와 함께 쓰고 가끔 바뀐다. 그래서 WARP를 끈 상태의 주소를 넣었다. WARP를 켠 채로는 관리자 화면이 403이다(사용자 화면은 상관없다). 주소가 바뀌어 403이 나오면 1~3을 다시 한다.

## 5.11 정리

옮길 때 만든 파일을 지운다. 데이터가 들어 있는 파일이다.

```bash
# macpro-debian
rm /srv/projects/kedp-backup/kedp.dump

# dell-debian
rm -r ~/kedp-move
```

dell-debian의 개발용 Go 서버를 다시 켜면 dell-debian DB에도 수집한다(두 곳에서 수집). 개발할 때만 켠다.

---

# 6. 백업

**왜:** DB가 망가지거나 실수로 지워져도 되살릴 수 있게, 매일 DB를 파일로 꺼내 둔다(KEDP001-D020).

`backup.sh`가 하는 일:

1. postgres container 안에서 `pg_dump`로 kedp DB를 꺼낸다.
2. `/srv/projects/kedp-backup/kedp-YYYYMMDD.dump`로 둔다(하루 1개).
3. 14일보다 오래된 백업 파일을 지운다. 그래서 백업 폴더가 끝없이 커지지 않는다.

먼저 손으로 한 번 돌려 본다:

```bash
/srv/projects/kedp/backup.sh
# backup: /srv/projects/kedp-backup/kedp-20261031.dump (… bytes)
```

**crontab**은 정해진 시각에 명령을 자동으로 실행해 주는 Linux 예약표다. 매일 03:00에 돌게 등록한다(사람이 안 쓰는 시각, 수집 06:00 전).

```bash
crontab -e
```

맨 아래에 한 줄을 더하고 저장한다:

```text
0 3 * * * /srv/projects/kedp/backup.sh >> /srv/projects/kedp-backup/backup.log 2>&1
```

| 칸 | 뜻 |
| --- | --- |
| `0 3 * * *` | 분 0, 시 3, 매일, 매월, 모든 요일 = 매일 03:00 |
| `/srv/projects/kedp/backup.sh` | 실행할 명령 |
| `>> .../backup.log 2>&1` | 결과와 오류를 `backup.log`에 이어 적는다 |

다음 날 확인:

```bash
ls -l /srv/projects/kedp-backup/
tail /srv/projects/kedp-backup/backup.log
```

백업에는 DB 계정(`kedp_go`, `kedp_rust`)이 들어 있지 않다. 계정은 postgres volume에 남아 있다. volume까지 잃었다면 5.6~5.7처럼 계정을 먼저 만든다.

### 백업에서 되살리기

kedp DB를 지우고 백업 파일로 다시 만든다. **지금 DB 값은 사라진다.**

```bash
cd /srv/projects/kedp
F=/srv/projects/kedp-backup/kedp-<YYYYMMDD>.dump
docker compose stop nginx go rust
docker compose exec -T postgres sh -c 'dropdb -U "$POSTGRES_USER" kedp && createdb -U "$POSTGRES_USER" kedp'
docker compose exec -T postgres sh -c 'pg_restore -U "$POSTGRES_USER" -d kedp --exit-on-error' < "$F"
docker compose up -d
```

---

# 7. 새 버전 올리기

코드가 바뀌어 main에 merge되면:

```text
 dell-debian                                  macpro-debian
 ───────────                                  ─────────────
 git pull (main)
 deploy/build.sh            → 새 tag
 deploy/send.sh macpro-debian <새 tag>  ──►   .env의 KEDP_IMAGE_TAG를 새 tag로
                                              docker compose up -d
```

```bash
# dell-debian
cd /srv/projects/aidev-kedp001 && git switch main && git pull
deploy/build.sh
deploy/send.sh macpro-debian <새 tag>
scp deploy/compose.yaml deploy/backup.sh macpro-debian:/srv/projects/kedp/    # 이 두 파일이 바뀌었을 때만

# macpro-debian
cd /srv/projects/kedp
sed -i 's/^KEDP_IMAGE_TAG=.*/KEDP_IMAGE_TAG=<새 tag>/' .env
docker compose up -d
docker compose ps && docker compose logs --tail 20 go
```

- `up -d`는 image가 바뀐 container만 새로 만든다. DB(volume)는 그대로다.
- 관리자 session은 go 메모리에 있으므로 go가 새로 뜨면 다시 로그인한다.
- 옛 image는 되돌릴 때 쓰므로 바로 지우지 않는다. 몇 번 배포한 뒤 `docker image rm kedp-go:<옛 tag> kedp-rust:<옛 tag> kedp-nginx:<옛 tag>`.

### 되돌리기

tag만 이전 것으로 바꾼다.

```bash
cd /srv/projects/kedp
docker images 'kedp-*'                                       # 남아 있는 tag
sed -i 's/^KEDP_IMAGE_TAG=.*/KEDP_IMAGE_TAG=<이전 tag>/' .env
docker compose up -d
```

### DB 구조가 바뀌는 배포 (migration)

`b-sql-db/migrations/`에 새 파일이 생긴 배포다. migration 파일은 image에 들어 있지 않으므로 따로 보내 적용한다. **적용 전에 백업한다.**

```bash
# dell-debian
scp b-sql-db/migrations/<NNNN_이름>.up.sql macpro-debian:/srv/projects/kedp-backup/

# macpro-debian
cd /srv/projects/kedp
./backup.sh
docker compose stop go
docker compose exec -T postgres sh -c 'psql -X -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d kedp' < /srv/projects/kedp-backup/<NNNN_이름>.up.sql
# .env의 KEDP_IMAGE_TAG를 새 tag로 바꾼 뒤
docker compose up -d
rm /srv/projects/kedp-backup/<NNNN_이름>.up.sql
```

migration 파일은 `BEGIN`/`COMMIT`으로 감싸 있어 실패하면 아무것도 바뀌지 않는다. migration을 적용한 배포를 되돌릴 때는 image만 되돌리면 DB와 맞지 않을 수 있다. 적용 전 백업으로 되살린다(6장).

---

# 8. compose.yaml 읽기

`compose.yaml`은 "container 4개를 이렇게 띄워라"는 설명서다. 중요한 줄만 본다.

### image와 tag

```yaml
  go:
    image: kedp-go:${KEDP_IMAGE_TAG}
```

`${...}`는 `.env`에서 값을 가져온다는 뜻이다. **새 버전, 되돌리기 모두 `.env`의 이 값 하나만 바꾼다.**

### 키와 설정 넣기

```yaml
  go:
    env_file: .env                     # .env 전체를 go에 준다(키 포함)
    environment:
      KEDP_DB_HOST: postgres           # 이 값은 .env보다 먼저다
      KEDP_PROCESSOR_ADDR: rust:50051
```

- `env_file`: `.env`의 값을 container에 넣는다. **go에만** 준다.
- `environment`: compose가 직접 정한 값이다. `.env`에 같은 이름이 있어도 이것이 이긴다. 그래서 dell-debian `.env`를 통째로 복사해도 괜찮았다.
- rust와 postgres에는 `.env` 전체를 주지 않고 필요한 값만 준다. rust는 API 키를 볼 수 없다.

### 이름으로 서로 찾기

`postgres`, `rust`, `go`는 container 이름이자 **내부 network 안의 주소**다. go는 `rust:50051`로 rust를, nginx는 `go:8080`으로 go를 부른다. IP를 몰라도 된다.

### 바깥에 문 열기

```yaml
  nginx:
    ports:
      - "${KEDP_HTTP_PORT:-80}:80"     # 바깥 포트 : container 안 포트
```

- `ports`가 있는 것은 **nginx 하나뿐**이다. 나머지는 바깥에서 접속할 수 없다.
- 왼쪽이 macpro-debian의 바깥 포트(8090), 오른쪽이 nginx 안의 포트(80)다. `:-80`은 비어 있으면 80이라는 뜻이다.

### 관리자 계정 파일 꽂기 (mount)

```yaml
  go:
    volumes:
      - ./admin:/run/kedp/admin:ro     # 서버 파일 : container 안 경로 : 읽기 전용
```

서버의 `admin` 파일을 go container 안에 **읽기 전용으로 꽂아 준다.** image 안에는 없다.

### DB 데이터 (volume)

```yaml
  postgres:
    volumes:
      - pgdata:/var/lib/postgresql/data
```

DB 파일은 `pgdata` volume(실제 이름 `kedp_pgdata`)에 있다. container를 지워도 남는다.

### 내부 network

```yaml
networks:
  kedp:
    ipam:
      config:
        - subnet: ${KEDP_NET_SUBNET}   # 172.30.46.0/24
```

container 4개만 쓰는 사설 network다.

### 다시 띄우기

```yaml
    restart: unless-stopped
```

container가 죽거나 서버가 재부팅되면 Docker가 다시 띄운다. 사람이 `docker compose stop`으로 멈춘 것만 다시 띄우지 않는다.

---

# 9. 매일 쓰는 명령

모두 macpro-debian의 `/srv/projects/kedp`에서 실행한다. compose는 **지금 폴더의** `compose.yaml`과 `.env`를 읽는다. 다른 폴더에 있으면 `docker compose -f /srv/projects/kedp/compose.yaml ...`.

| 하고 싶은 것 | 명령 |
| --- | --- |
| 4개 상태 보기 | `docker compose ps` |
| 기록 보기 | `docker compose logs --tail 20 go` (go 대신 nginx, rust, postgres) |
| 기록 계속 보기 | `docker compose logs -f go` (끝내기: `Ctrl+C`) |
| 수집 기록만 보기 | `docker compose logs --since 24h go \| grep collection` |
| `.env`를 고친 뒤 반영 | `docker compose up -d` |
| 하나만 다시 켜기(설정 그대로) | `docker compose restart go` |
| 모두 멈추기 / 다시 켜기 | `docker compose stop` / `docker compose up -d` |
| 메모리, CPU 사용량 | `docker stats --no-stream` |
| 서버에 있는 image | `docker images 'kedp-*'` |
| DB에 들어가기 | `docker compose exec postgres sh -c 'psql -U "$POSTGRES_USER" -d kedp'` |
| 백업 한 번 돌리기 | `./backup.sh` |

**`restart`와 `up -d`의 차이:** `restart`는 container를 그대로 껐다 켠다. 바뀐 `.env`, 바뀐 tag, 바뀐 포트를 **읽지 않는다.** `up -d`는 설정이 바뀐 container를 새로 만든다. 헷갈리면 `up -d`를 쓴다.

하지 말 것:

- `docker compose down -v`: `-v`가 DB volume까지 지운다.
- `docker volume rm kedp_pgdata`: DB가 사라진다.
- `.env`, `admin` 내용을 chat이나 저장소에 붙이기

---

# 10. 문제가 생기면

| 보이는 것 | 먼저 볼 것 |
| --- | --- |
| 브라우저가 계속 기다리다 실패 | 포트가 막혔다. PC에서 `Test-NetConnection <macpro-debian 주소> -Port 8090` |
| 화면은 뜨는데 카드가 비어 있음 | `docker compose logs --tail 20 go`. DB 접속 오류인지 본다 |
| 관리자 화면 "허용된 주소가 아닙니다" | WARP가 켜져 있는지. 5.10대로 nginx 기록의 PC 주소와 `.env`가 같은지 |
| 로그인이 안 됨 | `admin` 파일 소유자 `65532`, 권한 `400`인지. 파일을 바꿨으면 `docker compose restart go` |
| go가 계속 다시 시작됨 | `docker compose logs go`. 시작할 때 rust나 DB에 닿지 못하면 멈췄다가 다시 뜬다 |
| 수집이 안 됨 | 관리자 화면 API 관리의 상태, `docker compose logs --since 24h go \| grep collection` |
| 백업 파일이 안 생김 | `tail /srv/projects/kedp-backup/backup.log`, `crontab -l` |
| 메모리가 모자란 것 같음 | `free -h`의 available, `docker stats --no-stream`. 2026-10-31 기준 4개 띄운 뒤 available 약 2.1G |

---

# 11. 같은 서버에 다른 프로젝트를 올릴 때

다음 셋만 겹치지 않게 하면 KEDP와 섞이지 않는다.

- **폴더와 이름:** `/srv/projects/<프로젝트>`로 따로 둔다. compose는 `compose.yaml`의 `name:`(없으면 폴더 이름)으로 container와 volume 이름을 나눈다. KEDP는 `name: kedp`이므로 다른 프로젝트는 다른 이름을 쓴다.
- **바깥 포트:** KEDP는 8090. 다른 번호를 쓴다.
- **내부 network 대역:** KEDP는 `172.30.46.0/24`. 예: `172.30.47.0/24`.
