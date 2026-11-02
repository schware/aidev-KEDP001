# b-sql-db

KEDP PostgreSQL 스키마. 설계와 결정은 `specification/30.database/README.md`(KEDP001-D022)를 본다.

```text
b-sql-db/
├── migrations/
│   ├── 0001_schema.up.sql     테이블 9개, 이력 트리거, 기본 행(source 3행, collect_time)
│   ├── 0001_schema.down.sql
│   ├── 0002_roles.up.sql      계정 kedp_go, kedp_rust와 권한
│   ├── 0002_roles.down.sql
│   ├── 0003_visits.up.sql     방문자 수 visit_daily, 시작값 visit_total_base, kedp_go 권한
│   ├── 0003_visits.down.sql
│   ├── 0004_category.up.sql   지표 구분 category(9행), indicator.category_code, 지금 지표 7개의 구분, kedp_go 권한
│   └── 0004_category.down.sql
└── checks/
    └── smoke.sql              권한, 기간 검사, 이력 트리거, 원본 주소 검사 확인 (시험 DB 전용)
```

- 적용은 번호 순서(0001 → 0002 → 0003 → 0004), 되돌리기는 반대 순서(0004 → 0003 → 0002 → 0001)로 한다.
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
$K < b-sql-db/migrations/0003_visits.up.sql
$K < b-sql-db/migrations/0004_category.up.sql
```

이미 적용한 번호는 건너뛰고 그다음 번호부터 적용한다(예: 0003까지 적용한 DB에는 0004만). 배포 서버(macpro-debian)는 `deploy/README.md` 7장 "DB migration이 생겼을 때"를 따른다.

### 0004 적용 전 확인

`0004`는 아래 7개 지표만 구분을 채운다. 그 밖의 지표가 있으면 `0004`는 멈추고 아무것도 바꾸지 않는다. 적용 전에 확인한다. 결과가 0행이어야 한다.

```sh
$K -c "SELECT id, name FROM indicator WHERE id NOT IN ('ecos-base-rate', 'kosis-employment-rate', 'kosis-total-population', 'kosis-births', 'data-go-kr-kodex-200', 'data-go-kr-kodex-leverage', 'data-go-kr-kodex-inverse') ORDER BY id"
```

행이 나오면 적용하지 않는다. 그 지표의 구분은 소유자가 정한다.

적용 뒤 확인:

```sh
$K -c "SELECT i.id, i.category_code FROM indicator i ORDER BY i.id" -c "SELECT count(*) FROM category"   # 구분 9개
```

## 확인

```sh
$K -c '\dt' -c '\du kedp_*' -c '\dp'
```

- `\dt`: 테이블 11개 (`app_setting`, `category`, `collection_run`, `indicator`, `indicator_display`, `indicator_relation`, `observation`, `observation_history`, `raw_response`, `source`, `visit_daily`)
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
$P < b-sql-db/migrations/0004_category.down.sql
$P < b-sql-db/migrations/0003_visits.down.sql
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
$P < b-sql-db/migrations/0003_visits.up.sql
$P < b-sql-db/checks/smoke.sql   # NOTICE "ok: …" 5줄. 끝에 ROLLBACK하므로 데이터가 남지 않는다
$P < b-sql-db/migrations/0004_category.up.sql   # smoke.sql은 category_code를 넣지 않으므로 0004 전에 돌린다
# 위 "되돌리기"를 한 뒤 다시 적용하고 smoke.sql을 한 번 더 돌린다
docker exec kedp-postgres dropdb -U sunmoon kedp_migration_check
```

- `smoke.sql`은 `kedp`에서 돌리지 않는다. 데이터는 남지 않지만 `collection_run` 등의 번호가 넘어간다.
- 계정(`kedp_go`, `kedp_rust`)은 DB 서버 전체에 하나씩이다. `kedp`에 `0002`가 적용돼 있으면 시험 DB의 `0002_roles.down.sql`은 시험 DB의 권한만 거두고 계정은 남긴다(NOTICE `… is kept`). 그래서 `\du kedp_*`에 계정이 그대로 보이는 것이 정상이다.
