-- DB 테스트(tests/db.rs)용 지표와 수집 기록. scripts/db-test.sh가 시험 DB(kedp_rust_check)에 superuser로 넣는다.
-- kedp_rust는 indicator, collection_run에 쓰지 못하므로 여기서 미리 넣는다.
-- 테스트끼리 서로 영향을 주지 않도록 테스트마다 지표와 run_id를 따로 쓴다.

BEGIN;

INSERT INTO indicator (id, name, source_code, org_id, stat_code, item_code, classes, value_field, cycle) VALUES
    ('ecos-base-rate', '한국은행 기준금리', 'ecos', NULL, '722Y001', '0101000', '{}', NULL, 'M'),
    ('ecos-empty', '빈 값 시험', 'ecos', NULL, '722Y001', '0101000', '{}', NULL, 'M'),
    ('ecos-invalid', '잘못된 값 시험', 'ecos', NULL, '722Y001', '0101000', '{}', NULL, 'M'),
    ('kosis-employment-rate', '고용률', 'kosis', '101', 'DT_1DA7001S', 'T90', '{0}', NULL, 'M'),
    ('kosis-births', '출생아 수', 'kosis', '101', 'INH_1B8000F_01', 'T1', '{11}', NULL, 'A'),
    ('data-go-kr-kodex-200', 'KODEX 200', 'data-go-kr', NULL, '069500', NULL, '{}', 'clpr', 'D');

-- run_id를 테스트에서 바로 쓰려고 id를 정해서 넣는다.
INSERT INTO collection_run (id, source_code, indicator_id, status, trigger) OVERRIDING SYSTEM VALUE VALUES
    (1, 'ecos', 'ecos-base-rate', 'running', 'manual'),
    (2, 'ecos', 'ecos-base-rate', 'running', 'manual'),
    (3, 'ecos', 'ecos-base-rate', 'running', 'manual'),
    (4, 'kosis', 'kosis-employment-rate', 'running', 'manual'),
    (5, 'kosis', 'kosis-births', 'running', 'manual'),
    (6, 'data-go-kr', 'data-go-kr-kodex-200', 'running', 'manual'),
    (7, 'ecos', 'ecos-empty', 'running', 'manual'),
    (8, 'ecos', 'ecos-invalid', 'running', 'manual'),
    (9, 'ecos', 'ecos-base-rate', 'running', 'manual'),
    (10, 'kosis', 'kosis-employment-rate', 'running', 'manual'),
    (11, 'data-go-kr', 'data-go-kr-kodex-200', 'running', 'manual'),
    (12, 'ecos', 'ecos-base-rate', 'running', 'manual'),
    (13, 'ecos', 'ecos-base-rate', 'running', 'manual');

COMMIT;
