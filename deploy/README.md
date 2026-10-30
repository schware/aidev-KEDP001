# deploy

KEDP를 실행 서버(macpro-debian)에서 돌리는 파일과 순서다(KEDP001-D023, D020, D002, D012, D018).

- image는 개발 서버(dell-debian)에서 **하나씩** 만들고, `docker save` → SSH → `docker load`로 옮긴다. registry는 쓰지 않는다.
- macpro-debian에는 소스를 두지 않는다. image, `compose.yaml`, `.env`, 관리자 계정 파일, `backup.sh`만 둔다.
- 서버 주소, 계정, 비밀번호, 키, 허용 IP는 이 저장소에 쓰지 않는다. SSH 접속은 dell-debian의 `~/.ssh/config` Host 이름으로 한다.

```text
                 바깥 (KEDP_HTTP_PORT, 비우면 80)
                        │
                ┌───────▼────────┐  화면(정적 파일), /api → go
                │ nginx          │  X-Forwarded-For = 실제 접속 주소($remote_addr)
                └───────┬────────┘
     내부 network kedp  │ (KEDP_NET_SUBNET)
                ┌───────▼────────┐  gRPC   ┌──────────┐
                │ go :8080       ├────────►│ rust     │
                └───────┬────────┘  :50051 └────┬─────┘
                        │                       │
                ┌───────▼───────────────────────▼─────┐
                │ postgres:17 :5432 (volume pgdata)    │
                └──────────────────────────────────────┘
```

## 파일

| 파일 | 하는 일 |
| --- | --- |
| `compose.yaml` | container 4개(nginx, go, rust, postgres). 바깥에는 nginx 포트만 연다 |
| `.env.example` | 실행 서버 `.env`에 넣을 이름 목록(값 없음) |
| `nginx/default.conf` | nginx 설정. `kedp-nginx` image에 들어간다 |
| `build.sh` | image 3개를 하나씩 만든다(dell-debian) |
| `send.sh` | image 3개를 SSH로 보낸다(dell-debian → macpro-debian) |
| `backup.sh` | kedp DB 백업, 14일보다 오래된 파일 지우기(macpro-debian, crontab) |

image:

| image | Dockerfile | 들어 있는 것 |
| --- | --- | --- |
| `kedp-nginx:<tag>` | `f-ts-end/Dockerfile` | nginx, `npm run build` 결과, `nginx/default.conf` |
| `kedp-go:<tag>` | `b-go-end/Dockerfile` | `cmd/server` 실행 파일(시간대 정보 포함), CA 인증서 |
| `kedp-rust:<tag>` | `b-rust-processor/Dockerfile` | 처리 서버 실행 파일(정적) |

- go, rust는 multi-stage build이고 실행 image는 `scratch`다. `TZ=Asia/Seoul`이고, 사용자는 `65532`다.
- `.env`와 관리자 계정 파일은 image에 넣지 않는다. `.env`는 go에만 `env_file`로 주고, 계정 파일은 읽기 전용 mount(`/run/kedp/admin`)로 준다.
- API 키는 go container에만 들어간다(KEDP001-D012). rust에는 DB 비밀번호(`kedp_rust`)만, postgres에는 `POSTGRES_USER`, `POSTGRES_PASSWORD`만 준다.

## 1. image 만들기 (dell-debian)

다른 무거운 작업(Claude Code 여러 개, Rust 빌드, 다른 docker build)이 돌고 있지 않을 때 한다.

```sh
cd /srv/projects/aidev-kedp001
git switch main && git pull
deploy/build.sh                 # tag = commit hash(예: 8e2bff6). go → rust → nginx 순서로 하나씩
deploy/build.sh v1              # tag를 직접 준다
deploy/build.sh v1 rust         # 하나만 다시 만든다
docker images 'kedp-*'
```

- 처음 만들 때 시간: go 약 1분, rust 약 5분, nginx 약 1분(dell-debian 시험 때. base image 받는 시간 제외).
- 고친 파일이 있는 채로 tag 없이 만들면 tag 끝에 `-dirty`가 붙는다. 배포용은 commit한 상태로 만든다.

## 2. image 보내기 (dell-debian → macpro-debian)

```sh
deploy/send.sh <SSH Host 이름> <tag>
```

- `<SSH Host 이름>`은 dell-debian `~/.ssh/config`의 Host다. macpro-debian 계정은 `docker` group이어야 한다.
- `compose.yaml`, `backup.sh`가 바뀌었으면 함께 복사한다(4장).

## 3. macpro-debian에 필요한 것

- Docker Engine과 Compose plugin(`docker compose version`)
- 사용하는 계정이 `docker` group에 들어 있다
- 서버 시간대 Asia/Seoul(`timedatectl`). crontab의 03:00이 서버 시각이다
- 인터넷에서 `postgres:17` image를 받을 수 있다. 받을 수 없으면 dell-debian에서 `docker save postgres:17 | gzip -1 | ssh <Host> 'gunzip | docker load'`

폴더 구조(예: `/srv/kedp`. 다른 곳이어도 된다):

```text
/srv/kedp/              compose 폴더. 여기서 docker compose를 실행한다
├── compose.yaml        저장소 deploy/compose.yaml
├── backup.sh           저장소 deploy/backup.sh (실행 권한)
├── .env                chmod 600. 아래 ".env"
└── admin               관리자 계정 파일. 소유자 65532, chmod 400
/srv/kedp-backup/       백업 파일(KEDP_BACKUP_DIR). compose 폴더 밖에 둔다
```

```sh
# macpro-debian
sudo mkdir -p /srv/kedp /srv/kedp-backup
sudo chown "$USER": /srv/kedp /srv/kedp-backup
chmod 700 /srv/kedp-backup
```

### .env

`deploy/.env.example`의 이름을 모두 넣는다. dell-debian의 `.env`에서 가져오는 값과 새로 정하는 값이 있다.

| 이름 | 값 |
| --- | --- |
| `ECOS_API_KEY`, `KOSIS_API_KEY`, `DATA_GO_KR_API_KEY` | dell-debian과 같다 |
| `kedp_go`, `kedp_rust` | dell-debian과 같다(DB 계정을 비밀번호째 옮기므로) |
| `KEDP_PG_SUPERUSER`, `KEDP_PG_SUPERUSER_PASSWORD` | 새로 정한다. postgres container의 관리 계정. volume을 처음 만들 때만 쓰이고, 그 뒤에 바꿔도 DB에 반영되지 않는다 |
| `KEDP_ADMIN_ALLOW_IPS` | 관리자 화면을 쓸 PC의 IP 또는 CIDR. dell-debian 값과 다를 수 있다. `KEDP_NET_SUBNET`은 넣지 않는다(아래) |
| `KEDP_NET_SUBNET` | 새로 정한다. Docker 내부 network 대역(사설 CIDR, 예: `/24` 하나). `docker network inspect`와 `ip route`로 보아 서버의 다른 대역과 겹치지 않게 한다 |
| `KEDP_HTTP_PORT` | 바깥에 열 포트. 비우면 80 |
| `KEDP_IMAGE_TAG` | 띄울 image tag(`build.sh`가 알려 준 tag) |
| `KEDP_BACKUP_DIR` | 백업 폴더 절대 경로(예: `/srv/kedp-backup`) |

dell-debian과 달라지는 주소는 `compose.yaml`이 정한다. `.env`에 있어도 쓰지 않으므로 넣지 않아도 된다.

| 이름 | dell-debian | macpro-debian(compose가 넣는 값) |
| --- | --- | --- |
| `KEDP_DB_HOST`, `KEDP_DB_PORT`, `KEDP_DB_NAME` | kedp-postgres 주소, 포트, `kedp` | `postgres`, `5432`, `kedp` |
| `KEDP_PROCESSOR_ADDR` | Rust 주소(비우면 127.0.0.1:50051) | rust는 `0.0.0.0:50051`에서 열고, go는 `rust:50051`로 부른다 |
| `KEDP_API_ADDR` | Go 주소(비우면 127.0.0.1:8080) | `0.0.0.0:8080`(내부 network에서만 열린다) |
| `KEDP_ADMIN_FILE` | 계정 파일 경로 | `/run/kedp/admin`(compose 폴더의 `admin`을 mount) |
| `KEDP_TRUSTED_PROXIES` | 비움(127.0.0.1 = Vite proxy) | `KEDP_NET_SUBNET` 값(nginx가 있는 내부 network) |

허용 IP:

- go는 내부 network(`KEDP_TRUSTED_PROXIES`)에서 온 요청만 `X-Forwarded-For`를 믿는다. nginx는 이 값을 실제 접속 주소로 덮어쓰므로, 요청한 쪽이 꾸민 `X-Forwarded-For`는 쓰이지 않는다.
- macpro-debian 자신에서 `curl http://127.0.0.1:<포트>`로 부르면 nginx에는 Docker gateway 주소(내부 network 대역)로 보인다. 그래서 `KEDP_ADMIN_ALLOW_IPS`에 `KEDP_NET_SUBNET`을 넣지 않는다(넣으면 서버 안의 모든 요청이 허용된다).

### 관리자 계정 파일

dell-debian에서 만들어 보낸다(macpro-debian에는 Go가 없다).

```sh
# dell-debian
cd /srv/projects/aidev-kedp001/b-go-end
go run ./cmd/adminpass -account <계정> > ~/kedp-admin    # 비밀번호를 두 번 입력한다
chmod 600 ~/kedp-admin
scp ~/kedp-admin <Host>:/srv/kedp/admin && rm ~/kedp-admin

# macpro-debian: go container(사용자 65532)만 읽게 한다
sudo chown 65532:65532 /srv/kedp/admin && sudo chmod 400 /srv/kedp/admin
```

계정 파일을 바꾸면 `docker compose restart go`를 한다.

## 4. 처음 옮기기 (DB 포함)

dell-debian의 kedp DB(계정 `kedp_go`, `kedp_rust` 포함)를 macpro-debian으로 옮긴다. 아래에서 `SU`는 dell-debian `kedp-postgres`의 관리 계정이다(`b-sql-db/README.md`).

### 4.1 dell-debian: image 만들고 보내기

```sh
cd /srv/projects/aidev-kedp001
deploy/build.sh                         # tag를 적어 둔다
deploy/send.sh <Host> <tag>
scp deploy/compose.yaml deploy/backup.sh <Host>:/srv/kedp/
```

### 4.2 dell-debian: 수집 멈추고 dump

```sh
# 1) 수집 멈춤: cmd/server를 띄운 터미널에서 Ctrl+C. 8080이 닫혔는지 본다
ss -ltn | grep ':8080 ' || echo "go server stopped"

# 2) 옮길 파일 폴더
SU=<관리 계정>
mkdir -p ~/kedp-move && chmod 700 ~/kedp-move

# 3) DB 계정 2개(비밀번호 hash 포함). 4줄이 나와야 한다(CREATE ROLE, ALTER ROLE × 2)
docker exec kedp-postgres pg_dumpall -U "$SU" --roles-only \
  | grep -E '^(CREATE|ALTER) ROLE kedp_(go|rust)[ ;]' > ~/kedp-move/roles.sql
wc -l ~/kedp-move/roles.sql

# 4) kedp DB(데이터, 권한 포함). 소유자는 받는 쪽 관리 계정이 된다(--no-owner)
docker exec kedp-postgres pg_dump -U "$SU" -d kedp -Fc --no-owner > ~/kedp-move/kedp.dump
ls -l ~/kedp-move

# 5) 옮긴 뒤 비교할 행 수
docker exec kedp-postgres psql -X -U "$SU" -d kedp -At -c \
  "select 'indicator', count(*) from indicator union all select 'observation', count(*) from observation
   union all select 'collection_run', count(*) from collection_run union all select 'raw_response', count(*) from raw_response"

# 6) 보내기
scp ~/kedp-move/roles.sql ~/kedp-move/kedp.dump <Host>:/srv/kedp-backup/
```

### 4.3 macpro-debian: postgres만 띄워 되살리기

```sh
cd /srv/kedp
chmod 600 .env
docker compose up -d postgres
docker compose ps postgres                              # healthy가 될 때까지 기다린다

# 계정 먼저, 그다음 DB
PSQL='psql -X -q -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d kedp'
docker compose exec -T postgres sh -c "$PSQL" < /srv/kedp-backup/roles.sql
docker compose exec -T postgres sh -c 'pg_restore -U "$POSTGRES_USER" -d kedp --no-owner --exit-on-error' \
  < /srv/kedp-backup/kedp.dump

# 4.2 5)와 같은 행 수인지 본다
docker compose exec -T postgres sh -c "$PSQL -At" <<'SQL'
select 'indicator', count(*) from indicator union all select 'observation', count(*) from observation
union all select 'collection_run', count(*) from collection_run union all select 'raw_response', count(*) from raw_response;
SQL

# 계정 비밀번호가 .env와 맞는지(go, rust가 접속할 수 있는지)는 4.4에서 확인한다
rm /srv/kedp-backup/roles.sql                          # 비밀번호 hash가 들어 있다
```

### 4.4 macpro-debian: 나머지 띄우기와 확인

```sh
docker compose up -d
docker compose ps                                       # 4개 모두 running, postgres는 healthy
docker compose logs --tail 20 go rust                   # go: "processor check ok", "server started"
```

확인(5장)을 모두 한 뒤 `/srv/kedp-backup/kedp.dump`를 지우고, 백업 crontab(6장)을 등록한다. dell-debian의 `~/kedp-move`도 지운다.

- dell-debian의 개발용 수집 서버를 다시 켜면 dell-debian DB에도 수집한다(두 곳에서 수집). 필요할 때만 켠다.

## 5. 확인 명령

macpro-debian에서(`P`는 `KEDP_HTTP_PORT`, 비웠으면 80):

```sh
P=80
# 사용자 화면: index.html, 새로고침 주소도 index.html
curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:$P/                       # 200
curl -s http://127.0.0.1:$P/indicator/ecos-base-rate | grep -c 'id="root"'            # 1
curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:$P/admin/login            # 200

# /api
curl -s http://127.0.0.1:$P/api/indicators | head -c 300; echo
curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:$P/api/indicators/none    # 404

# 관리자 API: 서버 안에서는 허용 IP가 아니므로 403이 맞다
curl -s -w ' %{http_code}\n' http://127.0.0.1:$P/api/admin/session                  # {"error":"forbidden"} 403

# 수집 시각 기록
docker compose exec -T postgres sh -c 'psql -X -At -U "$POSTGRES_USER" -d kedp' <<'SQL'
select value from app_setting where key = 'collect_time';
select trigger, status, count(*), max(finished_at) from collection_run
 where started_at > now() - interval '2 days' group by 1, 2 order by 4 desc;
SQL
docker compose logs --since 24h go | grep -E 'collection|server started' | tail
```

허용 IP의 PC 브라우저에서:

1. `http://<macpro-debian>:<포트>/` 메인 화면 카드, 카드 하나를 눌러 상세 화면, 그 주소에서 새로고침
2. `/admin/login` 로그인 → Dashboard 관리 목록이 보인다
3. API 관리에서 수집 시각과 제공처 상태(마지막 수집)가 보인다
4. 허용 IP가 아닌 기기(휴대폰 등)에서 `/admin/login` → "허용된 주소가 아닙니다"

다음 수집 시각이 지난 뒤 `collection_run`에 `schedule` 기록이 새로 생겼는지 본다.

## 6. 백업 (KEDP001-D020)

`backup.sh`는 postgres container에서 `pg_dump -Fc`로 kedp를 받아 `KEDP_BACKUP_DIR/kedp-YYYYMMDD.dump`(하루 1개)로 두고, 14일보다 오래된 `kedp-*.dump`를 지운다.

```sh
/srv/kedp/backup.sh            # 한 번 돌려 본다. "backup: /srv/kedp-backup/kedp-20261101.dump (… bytes)"
```

crontab(`crontab -e`, 매일 03:00):

```text
0 3 * * * /srv/kedp/backup.sh >> /srv/kedp-backup/backup.log 2>&1
```

- crontab의 PATH에 `docker`가 있어야 한다(`/usr/bin/docker`면 기본 PATH로 된다).
- 백업에는 DB 계정(`kedp_go`, `kedp_rust`)이 들어 있지 않다. 계정은 postgres volume에 남아 있다. volume까지 잃었으면 4.2 3)처럼 계정을 먼저 만든다(또는 `CREATE ROLE kedp_go LOGIN;` 뒤 `\password`로 `.env`와 같은 비밀번호).

### 되살리기

kedp DB를 지우고 백업 파일로 다시 만든다. 지금 값은 사라진다.

```sh
cd /srv/kedp
F=/srv/kedp-backup/kedp-YYYYMMDD.dump
docker compose stop nginx go rust
docker compose exec -T postgres sh -c 'dropdb -U "$POSTGRES_USER" kedp && createdb -U "$POSTGRES_USER" kedp'
docker compose exec -T postgres sh -c 'pg_restore -U "$POSTGRES_USER" -d kedp --exit-on-error' < "$F"
docker compose up -d
```

## 7. 다음 배포부터

```sh
# dell-debian
cd /srv/projects/aidev-kedp001 && git switch main && git pull
deploy/build.sh                                 # 새 tag
deploy/send.sh <Host> <새 tag>
scp deploy/compose.yaml deploy/backup.sh <Host>:/srv/kedp/    # 바뀌었을 때만

# macpro-debian
cd /srv/kedp
sed -i 's/^KEDP_IMAGE_TAG=.*/KEDP_IMAGE_TAG=<새 tag>/' .env
docker compose up -d                            # image가 바뀐 container만 다시 만든다
docker compose ps && docker compose logs --tail 20 go
```

- 관리자 session은 서버 메모리에 있으므로 go를 다시 띄우면 다시 로그인한다.
- 이전 tag image는 되돌릴 때 쓰므로 바로 지우지 않는다. 몇 번 배포한 뒤 `docker image rm kedp-go:<옛 tag> …`로 지운다.

### DB migration이 생겼을 때

새 migration(`b-sql-db/migrations/NNNN_*.up.sql`)은 image에 들어 있지 않다. 파일을 보내서 적용한다.

```sh
# dell-debian
scp b-sql-db/migrations/NNNN_name.up.sql <Host>:/srv/kedp-backup/

# macpro-debian: 백업 → 수집 멈춤 → 적용 → 새 image로 띄우기
cd /srv/kedp
./backup.sh
docker compose stop go
docker compose exec -T postgres sh -c 'psql -X -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d kedp' \
  < /srv/kedp-backup/NNNN_name.up.sql
# .env의 KEDP_IMAGE_TAG를 새 tag로 바꾼 뒤
docker compose up -d
```

- migration 파일은 `BEGIN`/`COMMIT`으로 감싸 있어 실패하면 아무것도 바뀌지 않는다(`b-sql-db/README.md`).
- 적용한 뒤 파일은 지운다.

## 8. 되돌리기

이전 tag로 다시 띄운다.

```sh
cd /srv/kedp
docker images 'kedp-*'                                       # 남아 있는 tag
sed -i 's/^KEDP_IMAGE_TAG=.*/KEDP_IMAGE_TAG=<이전 tag>/' .env
docker compose up -d
```

- 그 배포에서 migration을 적용했다면 image만 되돌리면 DB와 맞지 않을 수 있다. migration의 `down.sql`을 같은 방법으로 적용하거나, 적용 전에 만든 백업으로 되살린다(6장).

## 9. dell-debian에서 시험 (kedp-check)

실제 서비스와 겹치지 않게 compose project `kedp-check`, 빈 시험 volume, 다른 포트로 띄워 본다. 지금 돌고 있는 `kedp-postgres`와 개발용 서버는 건드리지 않는다.

```sh
cd /srv/projects/aidev-kedp001
deploy/build.sh check go && deploy/build.sh check rust && deploy/build.sh check nginx

# 시험 폴더: compose.yaml, 빈 admin, 시험용 .env(키 없음, 시험 비밀번호, 허용 IP는 문서용 주소)
T=$(mktemp -d) && cp deploy/compose.yaml deploy/backup.sh "$T"/ && : > "$T/admin" && chmod 644 "$T/admin"
(umask 077; cat > "$T/.env" <<EOF
kedp_go=$(openssl rand -hex 16)
kedp_rust=$(openssl rand -hex 16)
KEDP_PG_SUPERUSER=checkadmin
KEDP_PG_SUPERUSER_PASSWORD=$(openssl rand -hex 16)
KEDP_ADMIN_ALLOW_IPS=192.0.2.1
KEDP_NET_SUBNET=<쓰지 않는 사설 /24>
KEDP_HTTP_PORT=18046
KEDP_IMAGE_TAG=check
KEDP_BACKUP_DIR=$T/backup
EOF
)
cd "$T" && export COMPOSE_PROJECT_NAME=kedp-check

docker compose up -d postgres && sleep 10
PSQL='psql -X -q -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d kedp'
docker compose exec -T postgres sh -c "$PSQL" < /srv/projects/aidev-kedp001/b-sql-db/migrations/0001_schema.up.sql
docker compose exec -T postgres sh -c "$PSQL" < /srv/projects/aidev-kedp001/b-sql-db/migrations/0002_roles.up.sql
. ./.env   # 시험 값만 들어 있다
docker compose exec -T postgres sh -c "$PSQL" <<EOF
ALTER ROLE kedp_go PASSWORD '$kedp_go';
ALTER ROLE kedp_rust PASSWORD '$kedp_rust';
EOF
docker compose up -d && sleep 5 && docker compose ps

curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:18046/admin/dashboard     # 200 (index.html)
curl -s http://127.0.0.1:18046/api/indicators                                        # {"indicators":[]}
curl -s -w ' %{http_code}\n' http://127.0.0.1:18046/api/admin/session               # forbidden 403
./backup.sh

# 지우기(container, network, 시험 volume)
docker compose down -v && cd / && rm -rf "$T"
docker image rm kedp-nginx:check kedp-go:check kedp-rust:check
```
