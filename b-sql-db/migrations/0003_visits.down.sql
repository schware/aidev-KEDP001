-- 0003_visits.up.sql을 되돌린다. 방문자 수와 시작값이 지워진다. 권한은 테이블과 함께 없어진다.

BEGIN;

DROP TABLE IF EXISTS visit_daily;
DELETE FROM app_setting WHERE key = 'visit_total_base';

COMMIT;
