#!/bin/sh
# 시험 DB(kedp_rust_check)를 만들어 DB 테스트(tests/db.rs)를 돌리고, 끝나면 시험 DB를 지운다.
# kedp DB에는 쓰지 않는다. DB 컨테이너(kedp-postgres)가 켜져 있어야 한다(b-sql-db/README.md).
#
# 1. 시험 DB를 만들고 migration(0001, 0002)과 tests/fixtures.sql을 superuser로 넣는다.
# 2. kedp_rust 계정으로 tests/db.rs를 돌린다. 접속 정보는 .env에서 읽고, DB 이름만 시험 DB로 바꾼다.
# 3. 트리거가 남긴 이력을 superuser로 확인한다(tests/check_history.sql).
# 4. 성공하든 실패하든 시험 DB를 지운다.
set -eu

cd "$(dirname "$0")/.."
DB=kedp_rust_check

# 컨테이너 안의 psql을 superuser로 실행한다. 계정 이름은 컨테이너의 POSTGRES_USER에서 읽는다.
psql_su() {
    docker exec -i kedp-postgres sh -c 'exec psql -X -q -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" "$@"' psql "$@"
}

drop_db() {
    psql_su -d postgres -c "DROP DATABASE IF EXISTS $DB WITH (FORCE)"
}
trap drop_db EXIT

drop_db
psql_su -d postgres -c "CREATE DATABASE $DB"
psql_su -d "$DB" < ../b-sql-db/migrations/0001_schema.up.sql
psql_su -d "$DB" < ../b-sql-db/migrations/0002_roles.up.sql
psql_su -d "$DB" < tests/fixtures.sql

KEDP_DB_NAME=$DB CARGO_BUILD_JOBS=2 cargo test --test db -- --ignored

psql_su -d "$DB" < tests/check_history.sql
