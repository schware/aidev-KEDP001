-- 0004_category.up.sql을 되돌린다. 지표의 구분과 구분 목록이 지워진다. 권한은 테이블과 함께 없어진다.

BEGIN;

ALTER TABLE indicator DROP COLUMN IF EXISTS category_code;
DROP TABLE IF EXISTS category;

COMMIT;
