-- 방문자 수(Issue #48). 날짜별 숫자만 둔다. IP, User-Agent, hash는 DB에 두지 않는다.
-- 0002_roles.up.sql 뒤에 적용한다(kedp_go 계정이 있어야 한다).

BEGIN;

-- 날짜(Asia/Seoul)별 방문자 수. 같은 사람(브라우저)은 하루에 한 번만 센다(Go가 거른다).
CREATE TABLE visit_daily (
    day date PRIMARY KEY,
    count integer NOT NULL CHECK (count >= 0)
);

-- 전체 방문자 수의 시작값. 전체 = 시작값 + visit_daily 합
INSERT INTO app_setting (key, value) VALUES ('visit_total_base', '0')
ON CONFLICT (key) DO NOTHING;

-- Go: 하루 숫자를 늘리고(INSERT … ON CONFLICT DO UPDATE), 관리자 초기화로 지운다.
GRANT SELECT, INSERT, UPDATE, DELETE ON visit_daily TO kedp_go;

COMMIT;
