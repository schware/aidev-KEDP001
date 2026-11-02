-- 0006_indicators_d024_org.up.sql을 되돌린다. 0006이 넣은 지표 5개를 지운다.
-- 0005 down과 같은 방식이다: 수집된 관측값이 있어도 되돌릴 수 있게 이력, 관측값을 먼저 지운다.
-- 화면 설정과 관련 지표는 함께 지워지고, 수집 기록의 indicator_id는 비워진다(0001 외래 키). 원본 응답은 남는다.

BEGIN;

CREATE TEMP TABLE d024_id (id text PRIMARY KEY) ON COMMIT DROP;
INSERT INTO d024_id (id) VALUES
    ('kosis-wage-nominal'),
    ('kosis-exports'),
    ('kosis-imports'),
    ('kosis-trade-balance'),
    ('kosis-unsold-housing');

DELETE FROM observation_history WHERE indicator_id IN (SELECT id FROM d024_id);
DELETE FROM observation WHERE indicator_id IN (SELECT id FROM d024_id);
DELETE FROM indicator WHERE id IN (SELECT id FROM d024_id);

COMMIT;
