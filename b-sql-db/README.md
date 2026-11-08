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
│   ├── 0004_category.down.sql
│   ├── 0005_indicators_d024.up.sql   D024 추가 지표 29개(지표, 화면 설정, 구분)
│   ├── 0005_indicators_d024.down.sql 0005 지표를 지운다(관측값·이력 포함)
│   ├── 0006_indicators_d024_org.up.sql   D024 추가 지표 2차 5개(통계청이 아닌 기관의 KOSIS 표)
│   ├── 0006_indicators_d024_org.down.sql 0006 지표를 지운다(관측값·이력 포함)
│   ├── 0007_indicator_facility_investment.up.sql   설비투자지수(계절조정) 1개(분류 {ALL,C}, Rust #60 뒤)
│   ├── 0007_indicator_facility_investment.down.sql 0007 지표를 지운다(관측값·이력 포함)
│   ├── 0008_rust_source_unit.up.sql   kedp_rust에 indicator.source_unit 칸 UPDATE 권한(Rust #76)
│   └── 0008_rust_source_unit.down.sql 그 권한만 거둔다(채워진 값은 남는다)
└── checks/
    └── smoke.sql              권한, 기간 검사, 이력 트리거, 원본 주소 검사 확인 (시험 DB 전용)
```

- 적용은 번호 순서(0001 → … → 0008), 되돌리기는 반대 순서(0008 → … → 0001)로 한다.
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
$K < b-sql-db/migrations/0005_indicators_d024.up.sql
$K < b-sql-db/migrations/0006_indicators_d024_org.up.sql
$K < b-sql-db/migrations/0007_indicator_facility_investment.up.sql
$K < b-sql-db/migrations/0008_rust_source_unit.up.sql
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

### 0005 적용 전 확인

`0005`는 D024 추가 지표 29개를 넣는다(`specification/10.indicators/README.md` "D024 추가 지표"). 같은 id가 이미 있으면 멈추고 아무것도 넣지 않는다. 결과가 0행이어야 한다.

```sh
$K -c "SELECT id, name FROM indicator WHERE id IN ('ecos-gdp-nominal', 'ecos-gdp-real', 'ecos-gdp-real-sa', 'ecos-gdp-growth-qoq', 'ecos-gdp-growth-yoy', 'kosis-industrial-production-sa', 'kosis-manufacturing-production-sa', 'kosis-service-production-sa', 'kosis-retail-sales-sa', 'kosis-cpi-yoy', 'kosis-living-cpi-yoy', 'kosis-fresh-food-cpi-yoy', 'kosis-core-cpi-yoy', 'kosis-employed', 'kosis-unemployment-rate', 'kosis-participation-rate', 'kosis-youth-employment-rate', 'ecos-ktb-3y', 'ecos-ktb-10y', 'ecos-corp-bond-3y-aa', 'ecos-loan-rate-new', 'ecos-deposit-rate-new', 'ecos-m2-avg', 'ecos-household-credit', 'ecos-housing-loan', 'ecos-usd-krw', 'ecos-current-account', 'kosis-working-age-population', 'kosis-elderly-ratio') ORDER BY id"
```

적용 뒤 확인(지표 36개, 노출 36개, D024 지표는 지금 순서 뒤):

```sh
$K -c "SELECT count(*), count(*) FILTER (WHERE d.is_visible) FROM indicator i JOIN indicator_display d ON d.indicator_id = i.id" \
   -c "SELECT c.sort_order, c.name, count(*) FROM indicator i JOIN category c ON c.code = i.category_code GROUP BY 1, 2 ORDER BY 1"
```

- 넣은 지표는 다음 예약 수집(수집 시각) 때 차트 기간만큼 처음부터 받는다. 첫 수집 호출은 ECOS 15번, KOSIS 14번이다.
- 되돌리기(`0005_indicators_d024.down.sql`)는 그 사이 수집된 관측값과 이력도 지운다. 원본 응답과 수집 기록은 남고 수집 기록의 지표 연결만 비워진다.

### 0006 적용 전 확인

`0006`은 D024 추가 지표 2차 5개를 넣는다(고용노동부 명목임금, 관세청 수출액·수입액·무역수지, 국토교통부 미분양 주택). `0005` 뒤에 적용한다. 같은 id가 이미 있으면 멈춘다. 결과가 0행이어야 한다.

```sh
$K -c "SELECT id, name FROM indicator WHERE id IN ('kosis-wage-nominal', 'kosis-exports', 'kosis-imports', 'kosis-trade-balance', 'kosis-unsold-housing') ORDER BY id"
```

적용 뒤 지표는 41개(지금 7 + 0005 29 + 0006 5)다. 첫 수집 호출은 KOSIS 5번이다.

### 0007 적용 전 확인

`0007`은 설비투자지수(계절조정) 1개를 넣는다(`kosis-facility-investment-sa`, 분류 `{ALL,C}`). `0006` 뒤에 적용한다. Rust image가 #60(분류 `ALL` 자리)이 들어간 것이어야 수집된다. 아니면 수집 결과가 `INVALID_RESPONSE`다. 결과가 0행이어야 한다.

```sh
$K -c "SELECT id, name FROM indicator WHERE id = 'kosis-facility-investment-sa'"
```

적용 뒤 지표는 42개다. 첫 수집 호출은 KOSIS 1번이다.

### 0008 적용 순서

`0008`은 `kedp_rust`에 `indicator.source_unit` 칸 하나만 UPDATE 권한을 준다(Issue #76). 지표와 관측값은 바꾸지 않는다.

- Rust image를 #76이 들어간 것으로 바꾸기 **전에** 적용한다. 권한 없이 새 Rust가 단위를 쓰려 하면 그 지표의 수집이 실패한다.
- 되돌릴 때는 반대로 Rust를 이전 image로 먼저 바꾸고 `0008_rust_source_unit.down.sql`을 적용한다.

적용 뒤 확인(`kedp_rust | UPDATE | source_unit` 한 줄만 나온다):

```sh
$K -c "SELECT grantee, privilege_type, column_name FROM information_schema.column_privileges WHERE table_name = 'indicator' AND grantee = 'kedp_rust' AND privilege_type = 'UPDATE'"
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
$P < b-sql-db/migrations/0008_rust_source_unit.down.sql
$P < b-sql-db/migrations/0007_indicator_facility_investment.down.sql
$P < b-sql-db/migrations/0006_indicators_d024_org.down.sql
$P < b-sql-db/migrations/0005_indicators_d024.down.sql
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
$P < b-sql-db/migrations/0005_indicators_d024.up.sql
$P < b-sql-db/migrations/0006_indicators_d024_org.up.sql
$P < b-sql-db/migrations/0007_indicator_facility_investment.up.sql
$P < b-sql-db/migrations/0008_rust_source_unit.up.sql
# 위 "되돌리기"를 한 뒤 다시 적용하고 smoke.sql을 한 번 더 돌린다
docker exec kedp-postgres dropdb -U sunmoon kedp_migration_check
```

- `smoke.sql`은 `kedp`에서 돌리지 않는다. 데이터는 남지 않지만 `collection_run` 등의 번호가 넘어간다.
- 계정(`kedp_go`, `kedp_rust`)은 DB 서버 전체에 하나씩이다. `kedp`에 `0002`가 적용돼 있으면 시험 DB의 `0002_roles.down.sql`은 시험 DB의 권한만 거두고 계정은 남긴다(NOTICE `… is kept`). 그래서 `\du kedp_*`에 계정이 그대로 보이는 것이 정상이다.
