-- Go와 Rust의 DB 접속 계정과 권한(KEDP001-D022 결정 7, 14)
-- 비밀번호는 여기에 쓰지 않는다. 적용한 뒤 소유자가 psql의 \password로 정한다.
-- 계정(role)은 DB 서버 전체에 하나씩이므로, 이미 있으면 만들지 않고 권한만 준다.

BEGIN;

DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'kedp_go') THEN
        CREATE ROLE kedp_go LOGIN;
    END IF;
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'kedp_rust') THEN
        CREATE ROLE kedp_rust LOGIN;
    END IF;
END
$$;

-- Go: 지표 정의, 화면 설정, 수집 시각, 수집 기록을 쓴다. 조회 API를 위해 관측값을 읽는다.
GRANT SELECT ON source TO kedp_go;
GRANT SELECT, INSERT, UPDATE, DELETE ON indicator, indicator_display, indicator_relation, app_setting TO kedp_go;
GRANT SELECT, INSERT, UPDATE ON collection_run TO kedp_go;
GRANT SELECT ON observation TO kedp_go;

-- Rust: 관측값과 원본 응답을 쓴다. 지표 정의는 읽기만 한다.
-- 이력은 observation 트리거가 Rust 계정으로 쓰므로 INSERT가 필요하다.
GRANT SELECT ON source, indicator TO kedp_rust;
GRANT SELECT, INSERT, UPDATE ON observation TO kedp_rust;
GRANT INSERT ON observation_history TO kedp_rust;
GRANT SELECT, INSERT ON raw_response TO kedp_rust;

COMMIT;
