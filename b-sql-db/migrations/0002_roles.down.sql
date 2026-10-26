-- 0002_roles.up.sql을 되돌린다. 이 DB에서 준 권한을 거두고 계정을 지운다.
-- 다른 DB에도 권한이 있는 계정이면 DROP ROLE이 실패한다. 그때는 그 DB에서 먼저 되돌린다.

BEGIN;

DO $$
DECLARE
    r text;
BEGIN
    FOREACH r IN ARRAY ARRAY['kedp_go', 'kedp_rust'] LOOP
        IF EXISTS (SELECT FROM pg_roles WHERE rolname = r) THEN
            EXECUTE format('REVOKE ALL ON ALL TABLES IN SCHEMA public FROM %I', r);
            EXECUTE format('DROP ROLE %I', r);
        END IF;
    END LOOP;
END
$$;

COMMIT;
