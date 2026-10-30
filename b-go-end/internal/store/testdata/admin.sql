-- admin_test.go의 시험 데이터. 0001, 0002 migration을 적용한 시험 DB(kedp_admin_check)에 superuser로 넣는다.
-- 시험이 값을 바꾸므로 시험마다 DB를 새로 만든다(b-go-end/README.md). kedp DB에는 넣지 않는다.

\set ON_ERROR_STOP on
BEGIN;

INSERT INTO indicator (id, name, source_code, stat_code, item_code, org_id, classes, value_field, cycle, unit, source_unit) VALUES
    ('a-rate', '금리', 'ecos', '722Y001', '0101000', NULL, '{}', NULL, 'M', '%', '연%'),
    ('a-pop', '인구', 'kosis', 'DT_1', 'T1', '101', '{00}', NULL, 'A', NULL, '명'),
    ('a-hidden', '숨김', 'data-go-kr', '069500', NULL, NULL, '{}', 'clpr', 'D', NULL, NULL),
    -- 화면 설정이 없다
    ('a-bare', '설정 없음', 'ecos', 'X', 'Y', NULL, '{}', NULL, 'Q', NULL, NULL);

-- a-rate의 number_format에는 decimals 말고 다른 칸도 있다(고쳐도 남아야 한다).
INSERT INTO indicator_display (indicator_id, is_visible, sort_order, chart_period, number_format, notice) VALUES
    ('a-rate', true, 1, '10 years', '{"decimals": 1, "keep": true}', NULL),
    ('a-pop', true, 2, '20 years', '{}', '안내 문구'),
    ('a-hidden', false, 3, '18 months', '{}', NULL);

INSERT INTO indicator_relation (indicator_id, related_indicator_id, sort_order) VALUES
    ('a-rate', 'a-hidden', 2),
    ('a-rate', 'a-pop', 1);

-- ecos: 성공 뒤에 오류로 끝났다. kosis: 데이터 없음 뒤에 진행 중이다(끝나지 않은 기록은 보지 않는다). data-go-kr: 기록 없음.
INSERT INTO collection_run (source_code, indicator_id, started_at, finished_at, status, row_count, trigger) VALUES
    ('ecos', 'a-rate', '2026-10-24 06:00:00+09', '2026-10-24 06:00:02+09', 'success', 3, 'schedule'),
    ('ecos', NULL, '2026-10-25 06:00:00+09', '2026-10-25 06:00:01+09', 'error', 0, 'schedule'),
    ('kosis', 'a-pop', '2026-10-25 06:00:00+09', '2026-10-25 06:00:05+09', 'no_data', 0, 'schedule');
INSERT INTO collection_run (source_code, indicator_id, started_at, status, trigger) VALUES
    ('kosis', 'a-pop', '2026-10-26 06:00:00+09', 'running', 'manual');

INSERT INTO observation (indicator_id, cycle, period_start, period_raw, value, value_raw, run_id)
SELECT 'a-rate', 'M', p::date, to_char(p::date, 'YYYYMM'), 2.5, '2.5', (SELECT min(id) FROM collection_run)
FROM (VALUES ('2026-08-01'), ('2026-09-01')) AS o (p);

COMMIT;
