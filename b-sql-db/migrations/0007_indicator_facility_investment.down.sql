-- 0007_indicator_facility_investment.up.sql을 되돌린다. 설비투자지수를 지운다.
-- 0005, 0006 down과 같은 방식이다: 수집된 관측값이 있어도 되돌릴 수 있게 이력, 관측값을 먼저 지운다.
-- 화면 설정과 관련 지표는 함께 지워지고, 수집 기록의 indicator_id는 비워진다(0001 외래 키). 원본 응답은 남는다.

BEGIN;

DELETE FROM observation_history WHERE indicator_id = 'kosis-facility-investment-sa';
DELETE FROM observation WHERE indicator_id = 'kosis-facility-investment-sa';
DELETE FROM indicator WHERE id = 'kosis-facility-investment-sa';

COMMIT;
