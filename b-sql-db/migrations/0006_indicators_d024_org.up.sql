-- KEDP001-D024 추가 지표 2차(Issue #55). 통계청이 아닌 기관의 KOSIS 표로 확인한 지표 5개를 넣는다.
-- 기관: 고용노동부(118), 관세청(134), 국토교통부(116). 확인은 cmd/sample -org로 했다(#58).
-- 근거: specification/10.indicators/README.md "D024 추가 지표", 샘플 samples/kosis/
-- 0005_indicators_d024.up.sql 뒤에 적용한다.
--
-- 0005와 같은 방식이다: 지표, 화면 설정, 구분을 한 transaction으로 넣는다. 같은 id가 이미 있으면 멈추고 아무것도 넣지 않는다.
-- 노출: 켠다. 순서: D024 목록 순서로 지금 지표 뒤(지금 가장 큰 sort_order + 1부터). 차트 기간: M 10 years.

BEGIN;

CREATE TEMP TABLE d024_indicator (
    n integer PRIMARY KEY,
    id text NOT NULL,
    name text NOT NULL,
    category_code text NOT NULL,
    source_code text NOT NULL,
    org_id text,
    stat_code text NOT NULL,
    item_code text,
    classes text[] NOT NULL,
    cycle text NOT NULL,
    unit text,
    chart_period interval NOT NULL
) ON COMMIT DROP;

INSERT INTO d024_indicator (n, id, name, category_code, source_code, org_id, stat_code, item_code, classes, cycle, unit, chart_period) VALUES
    (1, 'kosis-wage-nominal', '명목임금(월, 전체임금총액)', 'labor', 'kosis', '118', 'DT_118N_MON051', '13103110311MD_12', '{190326INDUSTRY_10S0,size01}', 'M', NULL, '10 years'),
    (2, 'kosis-exports', '수출액', 'external', 'kosis', '134', 'DT_134001_001', 'T002', '{DATA}', 'M', NULL, '10 years'),
    (3, 'kosis-imports', '수입액', 'external', 'kosis', '134', 'DT_134001_001', 'T004', '{DATA}', 'M', NULL, '10 years'),
    (4, 'kosis-trade-balance', '무역수지', 'external', 'kosis', '134', 'DT_134001_001', 'T005', '{DATA}', 'M', NULL, '10 years'),
    (5, 'kosis-unsold-housing', '미분양 주택', 'housing', 'kosis', '116', 'DT_MLTM_2080', '13103792722T1', '{13102792722A.0001,13102792722B.0001,13102792722C.0001}', 'M', NULL, '10 years');

INSERT INTO indicator (id, name, category_code, source_code, org_id, stat_code, item_code, classes, cycle, unit)
SELECT id, name, category_code, source_code, org_id, stat_code, item_code, classes, cycle, unit
FROM d024_indicator
ORDER BY n;

INSERT INTO indicator_display (indicator_id, is_visible, sort_order, chart_period)
SELECT t.id, true, base.max_order + t.n, t.chart_period
FROM d024_indicator t
CROSS JOIN (SELECT coalesce(max(sort_order), 0) AS max_order FROM indicator_display) base
ORDER BY t.n;

COMMIT;
