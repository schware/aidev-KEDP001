-- 응답 단위 저장(Issue #76, KEDP001-D022 결정 9). Rust가 수집할 때 indicator.source_unit을 채운다.
-- kedp_rust에 indicator의 source_unit 칸 하나만 UPDATE 권한을 준다. 다른 칸(관리자 단위 unit 등)은 지금처럼 읽기만 한다.
-- 0007_indicator_facility_investment.up.sql 뒤에 적용한다. Rust image는 #76이 들어간 것이어야 이 칸을 쓴다.

BEGIN;

GRANT UPDATE (source_unit) ON indicator TO kedp_rust;

COMMIT;
