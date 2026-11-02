-- KEDP001-D024 추가 지표(Issue #55). 코드를 샘플로 확인한 지표 29개를 넣는다.
-- 근거: specification/10.indicators/README.md "D024 추가 지표", 샘플 samples/ecos/, samples/kosis/
-- 0004_category.up.sql 뒤에 적용한다(구분 category가 있어야 한다).
--
-- 지표, 화면 설정, 구분을 한 transaction으로 넣는다. 같은 id가 이미 있으면 멈추고 아무것도 넣지 않는다.
-- 노출: 켠다. 순서: D024 목록 순서로 지금 지표 뒤(지금 가장 큰 sort_order + 1부터).
-- 차트 기간: D 3 years, M 10 years, Q 20 years, A 30 years.
-- 단위(unit)는 응답에 단위가 없는 지표에만 넣는다. 나머지는 수집 때 응답 단위(source_unit)를 쓴다.

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
    (1, 'ecos-gdp-nominal', '명목 GDP(원계열)', 'growth', 'ecos', NULL, '200Y105', '1400', '{}', 'Q', NULL, '20 years'),
    (2, 'ecos-gdp-real', '실질 GDP(원계열)', 'growth', 'ecos', NULL, '200Y106', '1400', '{}', 'Q', NULL, '20 years'),
    (3, 'ecos-gdp-real-sa', '실질 GDP(계절조정)', 'growth', 'ecos', NULL, '200Y104', '1400', '{}', 'Q', NULL, '20 years'),
    (4, 'ecos-gdp-growth-qoq', '실질 GDP 성장률(계절조정, 전기 대비)', 'growth', 'ecos', NULL, '200Y102', '10111', '{}', 'Q', NULL, '20 years'),
    (5, 'ecos-gdp-growth-yoy', '실질 GDP 성장률(원계열, 전년 동기 대비)', 'growth', 'ecos', NULL, '200Y102', '10211', '{}', 'Q', NULL, '20 years'),
    (6, 'kosis-industrial-production-sa', '전산업생산지수(계절조정)', 'growth', 'kosis', '101', 'DT_1JH20202', 'T1', '{1}', 'M', NULL, '10 years'),
    (7, 'kosis-manufacturing-production-sa', '제조업생산지수(계절조정)', 'growth', 'kosis', '101', 'DT_1F02001', 'T20', '{00,C}', 'M', NULL, '10 years'),
    (8, 'kosis-service-production-sa', '서비스업생산지수(계절조정)', 'growth', 'kosis', '101', 'DT_1KC2020', 'T3', '{T}', 'M', NULL, '10 years'),
    (9, 'kosis-retail-sales-sa', '소매판매액지수(계절조정)', 'growth', 'kosis', '101', 'DT_1K41012', 'T3', '{G0}', 'M', NULL, '10 years'),
    (10, 'kosis-cpi-yoy', '소비자물가지수 전년 동월 대비', 'price', 'kosis', '101', 'DT_1J22042', 'T03', '{0}', 'M', '%', '10 years'),
    (11, 'kosis-living-cpi-yoy', '생활물가지수 전년 동월 대비', 'price', 'kosis', '101', 'DT_1J22042', 'T03', '{1}', 'M', '%', '10 years'),
    (12, 'kosis-fresh-food-cpi-yoy', '신선식품지수 전년 동월 대비', 'price', 'kosis', '101', 'DT_1J22042', 'T03', '{2}', 'M', '%', '10 years'),
    (13, 'kosis-core-cpi-yoy', '근원물가(식료품·에너지 제외) 전년 동월 대비', 'price', 'kosis', '101', 'DT_1J22042', 'T03', '{4}', 'M', '%', '10 years'),
    (14, 'kosis-employed', '취업자 수', 'labor', 'kosis', '101', 'DT_1DA7001S', 'T30', '{0}', 'M', NULL, '10 years'),
    (15, 'kosis-unemployment-rate', '실업률', 'labor', 'kosis', '101', 'DT_1DA7001S', 'T80', '{0}', 'M', NULL, '10 years'),
    (16, 'kosis-participation-rate', '경제활동참가율', 'labor', 'kosis', '101', 'DT_1DA7001S', 'T60', '{0}', 'M', NULL, '10 years'),
    (17, 'kosis-youth-employment-rate', '청년 고용률(15~29세)', 'labor', 'kosis', '101', 'DT_1DA7002S', 'T90', '{75}', 'M', NULL, '10 years'),
    (18, 'ecos-ktb-3y', '국고채 3년 금리', 'money', 'ecos', NULL, '721Y001', '5020000', '{}', 'M', NULL, '10 years'),
    (19, 'ecos-ktb-10y', '국고채 10년 금리', 'money', 'ecos', NULL, '721Y001', '5050000', '{}', 'M', NULL, '10 years'),
    (20, 'ecos-corp-bond-3y-aa', '회사채 3년 AA- 금리', 'money', 'ecos', NULL, '721Y001', '7020000', '{}', 'M', NULL, '10 years'),
    (21, 'ecos-loan-rate-new', '예금은행 대출금리(신규취급액)', 'money', 'ecos', NULL, '121Y006', 'BECBLA01', '{}', 'M', NULL, '10 years'),
    (22, 'ecos-deposit-rate-new', '예금은행 수신금리(신규취급액)', 'money', 'ecos', NULL, '121Y002', 'BEABAA2', '{}', 'M', NULL, '10 years'),
    (23, 'ecos-m2-avg', 'M2(평잔, 원계열)', 'money', 'ecos', NULL, '161Y006', 'BBHA00', '{}', 'M', NULL, '10 years'),
    (24, 'ecos-household-credit', '가계신용(잔액)', 'credit', 'ecos', NULL, '151Y001', '1000000', '{}', 'Q', NULL, '20 years'),
    (25, 'ecos-housing-loan', '주택관련대출(예금취급기관, 잔액)', 'credit', 'ecos', NULL, '151Y005', '11100A0', '{}', 'M', NULL, '10 years'),
    (26, 'ecos-usd-krw', '원·달러 환율(월평균)', 'external', 'ecos', NULL, '731Y004', '0000001', '{0000100}', 'M', NULL, '10 years'),
    (27, 'ecos-current-account', '경상수지(원계열)', 'external', 'ecos', NULL, '301Y013', '000000', '{}', 'M', NULL, '10 years'),
    (28, 'kosis-working-age-population', '생산가능인구(15~64세)', 'population', 'kosis', '101', 'DT_1DA7002S', 'T10', '{63}', 'M', NULL, '10 years'),
    (29, 'kosis-elderly-ratio', '65세 이상 인구 비율(장래인구추계)', 'population', 'kosis', '101', 'DT_1BPA002', 'T10', '{1,33}', 'A', '%', '30 years');

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
