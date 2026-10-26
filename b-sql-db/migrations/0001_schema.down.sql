-- 0001_schema.up.sql을 되돌린다. 모든 테이블과 데이터가 지워진다.

BEGIN;

DROP TABLE IF EXISTS raw_response;
DROP TABLE IF EXISTS observation_history;
-- 트리거는 테이블과 함께 지워진다. 함수는 트리거가 없어진 뒤에 지운다.
DROP TABLE IF EXISTS observation;
DROP FUNCTION IF EXISTS observation_keep_history();
DROP TABLE IF EXISTS collection_run;
DROP TABLE IF EXISTS app_setting;
DROP TABLE IF EXISTS indicator_relation;
DROP TABLE IF EXISTS indicator_display;
DROP TABLE IF EXISTS indicator;
DROP TABLE IF EXISTS source;

COMMIT;
