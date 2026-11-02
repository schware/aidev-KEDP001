-- KEDP001-D024 추가 지표 3차(Issue #55). 설비투자지수(계절조정) 1개를 넣는다.
-- 분류 B(부문별)가 둘째 자리인 표라 분류를 {ALL,C}로 둔다. 응답 행에 C1이 없으며, Rust는 ALL 자리를 대조하지 않는다(#60).
-- 근거: specification/10.indicators/README.md "D024 추가 지표", 샘플 samples/kosis/DT_1F70011-T5-ALL-C-M-202401-202609.json
-- 0006_indicators_d024_org.up.sql 뒤에 적용한다. Rust image는 #60이 들어간 것이어야 수집된다.
--
-- 0005, 0006과 같은 방식이다: 지표, 화면 설정, 구분을 한 transaction으로 넣는다. 같은 id가 이미 있으면 멈춘다.
-- 노출: 켠다. 순서: 지금 지표 뒤(지금 가장 큰 sort_order + 1). 차트 기간: M 10 years.

BEGIN;

INSERT INTO indicator (id, name, category_code, source_code, org_id, stat_code, item_code, classes, cycle, unit)
VALUES ('kosis-facility-investment-sa', '설비투자지수(계절조정)', 'growth', 'kosis', '101', 'DT_1F70011', 'T5', '{ALL,C}', 'M', NULL);

INSERT INTO indicator_display (indicator_id, is_visible, sort_order, chart_period)
SELECT 'kosis-facility-investment-sa', true, coalesce(max(sort_order), 0) + 1, '10 years'
FROM indicator_display;

COMMIT;
