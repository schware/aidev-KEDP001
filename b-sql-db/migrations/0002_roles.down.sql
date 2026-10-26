-- 0002_roles.up.sql을 되돌린다. 이 DB에서 준 권한을 거두고 계정을 지운다.
-- 계정은 DB 서버 전체에 하나씩이다. 다른 DB에 아직 권한이 있으면(예: 시험 DB를 되돌리는데 kedp에 적용돼 있음)
-- 이 DB의 권한만 거두고 계정은 남긴다.

BEGIN;

DO $$
DECLARE
    r text;
BEGIN
    FOREACH r IN ARRAY ARRAY['kedp_go', 'kedp_rust'] LOOP
        IF EXISTS (SELECT FROM pg_roles WHERE rolname = r) THEN
            EXECUTE format('REVOKE ALL ON ALL TABLES IN SCHEMA public FROM %I', r);
            IF EXISTS (
                SELECT FROM pg_shdepend d
                JOIN pg_roles ro ON ro.oid = d.refobjid
                WHERE ro.rolname = r
                  AND d.refclassid = 'pg_authid'::regclass
                  AND d.dbid <> (SELECT oid FROM pg_database WHERE datname = current_database())
            ) THEN
                RAISE NOTICE '% is kept: it still has privileges in another database', r;
            ELSE
                EXECUTE format('DROP ROLE %I', r);
            END IF;
        END IF;
    END LOOP;
END
$$;

COMMIT;
