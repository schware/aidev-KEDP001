# b-sql-db

KEDP PostgreSQL 스키마. 설계와 결정은 `specification/30.database/README.md`(KEDP001-D022)를 본다.

```text
b-sql-db/
├── migrations/
│   ├── 0001_schema.up.sql     테이블 9개, 이력 트리거, 기본 행(source 3행, collect_time)
│   ├── 0001_schema.down.sql
│   ├── 0002_roles.up.sql      계정 kedp_go, kedp_rust와 권한
│   └── 0002_roles.down.sql
└── checks/
    └── smoke.sql              권한, 기간 검사, 이력 트리거, 원본 주소 검사 확인 (시험 DB 전용)
```

- 적용은 번호 순서(0001 → 0002), 되돌리기는 반대 순서(0002 → 0001)로 한다.
- 파일마다 `BEGIN`/`COMMIT`으로 감싸 있어 중간에 실패하면 그 파일 전체가 적용되지 않는다.
- 접속 정보와 비밀번호는 저장소에 쓰지 않는다. 계정 비밀번호는 적용한 뒤 소유자가 정한다(아래 "비밀번호").
- PostgreSQL 11 이상이 필요하다(`EXECUTE FUNCTION`). 개발 서버는 PostgreSQL 17이다.

## 개발 서버의 DB

dell-debian의 Docker 컨테이너 `kedp-postgres`다. 포트는 서버 안(`127.0.0.1:5432`)에만 열려 있다.

| DB | 쓰임 |
| --- | --- |
| `kedp` | 실제 DB |
| `kedp_migration_check` | migration 시험용. 필요할 때 만들고 끝나면 지운다 |

호스트에는 `psql`이 없으므로 컨테이너 안의 `psql`을 쓴다. 명령은 저장소 최상위 폴더에서 실행한다.

```sh
cd /srv/projects/aidev-kedp001
docker start kedp-postgres          # 서버를 다시 켰을 때
docker exec -it kedp-postgres psql -U sunmoon -d kedp
```

## 적용

```sh
K="docker exec -i kedp-postgres psql -X -v ON_ERROR_STOP=1 -U sunmoon -d kedp"
$K < b-sql-db/migrations/0001_schema.up.sql
$K < b-sql-db/migrations/0002_roles.up.sql
```

## 확인

```sh
$K -c '\dt' -c '\du kedp_*' -c '\dp'
```

- `\dt`: 테이블 9개 (`app_setting`, `collection_run`, `indicator`, `indicator_display`, `indicator_relation`, `observation`, `observation_history`, `raw_response`, `source`)
- `\du kedp_*`: 계정 `kedp_go`, `kedp_rust`
- `\dp`: 테이블별 권한. 계정별 권한은 설계서 4.1과 같다

## 비밀번호

```sh
docker exec -it kedp-postgres psql -U sunmoon -d kedp
```

psql 안에서 입력한다. 비밀번호는 화면과 명령 기록에 남지 않는다.

```text
\password kedp_go
\password kedp_rust
\q
```

로그인 확인 (비밀번호를 물으면 입력한다):

```sh
docker exec -it kedp-postgres psql -h 127.0.0.1 -U kedp_go -d kedp -c 'select current_user'
docker exec -it kedp-postgres psql -h 127.0.0.1 -U kedp_rust -d kedp -c 'select current_user'
```

## 되돌리기

모든 테이블과 데이터가 지워진다. `kedp`에서는 하지 않는다. 시험 DB에서만 쓴다.

```sh
P="docker exec -i kedp-postgres psql -X -v ON_ERROR_STOP=1 -U sunmoon -d kedp_migration_check"
$P < b-sql-db/migrations/0002_roles.down.sql
$P < b-sql-db/migrations/0001_schema.down.sql
$P -c '\dt' -c '\du kedp_*'      # 테이블은 없어야 한다. 계정은 kedp에 적용돼 있으면 남는다(아래 참고)
```

## migration 시험

새 migration을 만들거나 고쳤을 때 시험 DB에서 적용 → 점검 → 되돌리기 → 다시 적용을 확인한다.

```sh
P="docker exec -i kedp-postgres psql -X -v ON_ERROR_STOP=1 -U sunmoon -d kedp_migration_check"
docker exec kedp-postgres createdb -U sunmoon kedp_migration_check
$P < b-sql-db/migrations/0001_schema.up.sql
$P < b-sql-db/migrations/0002_roles.up.sql
$P < b-sql-db/checks/smoke.sql   # NOTICE "ok: …" 5줄. 끝에 ROLLBACK하므로 데이터가 남지 않는다
# 위 "되돌리기"를 한 뒤 다시 적용하고 smoke.sql을 한 번 더 돌린다
docker exec kedp-postgres dropdb -U sunmoon kedp_migration_check
```

- `smoke.sql`은 `kedp`에서 돌리지 않는다. 데이터는 남지 않지만 `collection_run` 등의 번호가 넘어간다.
- 계정(`kedp_go`, `kedp_rust`)은 DB 서버 전체에 하나씩이다. `kedp`에 `0002`가 적용돼 있으면 시험 DB의 `0002_roles.down.sql`은 시험 DB의 권한만 거두고 계정은 남긴다(NOTICE `… is kept`). 그래서 `\du kedp_*`에 계정이 그대로 보이는 것이 정상이다.
