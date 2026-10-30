-- read_test.go의 시험 데이터. 0001, 0002 migration을 적용한 시험 DB에 superuser로 넣는다.
-- kedp DB에는 넣지 않는다.

\set ON_ERROR_STOP on
BEGIN;

INSERT INTO indicator (id, name, source_code, stat_code, item_code, org_id, value_field, cycle, unit, source_unit) VALUES
    -- 관리자 단위가 응답 단위보다 먼저다
    ('t-rate', '금리', 'ecos', '722Y001', '0101000', NULL, NULL, 'M', '%', '연%'),
    -- 관리자 단위가 없으면 응답 단위다
    ('t-pop', '인구', 'kosis', 'DT_1', 'T1', '101', NULL, 'A', NULL, '명'),
    -- 노출하지 않는다
    ('t-hidden', '숨김', 'data-go-kr', '069500', NULL, NULL, 'clpr', 'D', NULL, NULL),
    -- 화면 설정이 없다. 빈 단위는 없는 것과 같다
    ('t-bare', '설정 없음', 'ecos', 'X', 'Y', NULL, NULL, 'Q', '', '');

INSERT INTO indicator_display (indicator_id, is_visible, sort_order, chart_period, notice) VALUES
    ('t-rate', true, 2, '1 year', NULL),
    ('t-pop', true, 1, '20 years', '안내 문구'),
    ('t-hidden', false, 0, '1 year', NULL);

INSERT INTO indicator_relation (indicator_id, related_indicator_id, sort_order) VALUES
    ('t-rate', 't-pop', 2),
    ('t-rate', 't-hidden', 1);

-- t-rate: 성공(10-25 06:00:03 KST) 뒤에 오류(10-26), 진행 중(10-27)이 있다. 마지막 수집은 성공한 10-25다.
INSERT INTO collection_run (source_code, indicator_id, started_at, finished_at, status, row_count, trigger) VALUES
    ('ecos', 't-rate', '2026-10-24 06:00:00+09', '2026-10-24 06:00:02+09', 'success', 3, 'schedule'),
    ('ecos', 't-rate', '2026-10-25 06:00:00+09', '2026-10-25 06:00:03+09', 'success', 3, 'schedule'),
    ('ecos', 't-rate', '2026-10-26 06:00:00+09', '2026-10-26 06:00:01+09', 'error', 0, 'schedule'),
    ('kosis', 't-pop', '2026-10-25 06:00:00+09', '2026-10-25 06:00:05+09', 'no_data', 0, 'schedule');
INSERT INTO collection_run (source_code, indicator_id, started_at, status, trigger) VALUES
    ('ecos', 't-rate', '2026-10-27 06:00:00+09', 'running', 'manual');

INSERT INTO observation (indicator_id, cycle, period_start, period_raw, value, value_raw, run_id)
SELECT o.indicator_id, o.cycle, o.period_start::date, o.period_raw, o.value::numeric, o.value, (SELECT min(id) FROM collection_run)
FROM (VALUES
    ('t-rate', 'M', '2025-09-01', '202509', '3.5'),
    -- 차트 시작(2026-10-30 - 1 year = 2025-10-30) 전이다
    ('t-rate', 'M', '2025-10-01', '202510', '3.25'),
    ('t-rate', 'M', '2025-11-01', '202511', '3'),
    ('t-rate', 'M', '2026-08-01', '202608', '2.75'),
    ('t-rate', 'M', '2026-09-01', '202609', '2.5'),
    -- 이전 주기 값. 기간이 더 늦어도 읽지 않는다
    ('t-rate', 'Q', '2026-10-01', '2026Q4', '99'),
    ('t-pop', 'A', '2024-01-01', '2024', '51700000'),
    ('t-pop', 'A', '2025-01-01', '2025', '51684564')
) AS o (indicator_id, cycle, period_start, period_raw, value);

COMMIT;
