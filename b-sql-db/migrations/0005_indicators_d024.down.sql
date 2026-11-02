-- 0005_indicators_d024.up.sql을 되돌린다. 0005가 넣은 지표 29개를 지운다.
-- 수집된 관측값이 있어도 되돌릴 수 있게 이력, 관측값을 먼저 지운다.
-- 화면 설정과 관련 지표(다른 지표에서 이 지표를 고른 것 포함)는 함께 지워지고, 수집 기록의 indicator_id는 비워진다(0001 외래 키).
-- 원본 응답(raw_response)은 수집 기록에 묶여 있어 남는다(90일 보관 규칙을 따른다).

BEGIN;

CREATE TEMP TABLE d024_id (id text PRIMARY KEY) ON COMMIT DROP;
INSERT INTO d024_id (id) VALUES
    ('ecos-gdp-nominal'),
    ('ecos-gdp-real'),
    ('ecos-gdp-real-sa'),
    ('ecos-gdp-growth-qoq'),
    ('ecos-gdp-growth-yoy'),
    ('kosis-industrial-production-sa'),
    ('kosis-manufacturing-production-sa'),
    ('kosis-service-production-sa'),
    ('kosis-retail-sales-sa'),
    ('kosis-cpi-yoy'),
    ('kosis-living-cpi-yoy'),
    ('kosis-fresh-food-cpi-yoy'),
    ('kosis-core-cpi-yoy'),
    ('kosis-employed'),
    ('kosis-unemployment-rate'),
    ('kosis-participation-rate'),
    ('kosis-youth-employment-rate'),
    ('ecos-ktb-3y'),
    ('ecos-ktb-10y'),
    ('ecos-corp-bond-3y-aa'),
    ('ecos-loan-rate-new'),
    ('ecos-deposit-rate-new'),
    ('ecos-m2-avg'),
    ('ecos-household-credit'),
    ('ecos-housing-loan'),
    ('ecos-usd-krw'),
    ('ecos-current-account'),
    ('kosis-working-age-population'),
    ('kosis-elderly-ratio');

DELETE FROM observation_history WHERE indicator_id IN (SELECT id FROM d024_id);
DELETE FROM observation WHERE indicator_id IN (SELECT id FROM d024_id);
DELETE FROM indicator WHERE id IN (SELECT id FROM d024_id);

COMMIT;
