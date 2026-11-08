-- 0008_rust_source_unit.up.sql을 되돌린다. kedp_rust의 source_unit UPDATE 권한만 거둔다.
-- 이미 채워진 source_unit 값은 그대로 둔다. #76이 들어간 Rust image는 이 권한이 없으면 수집이 실패하므로, 되돌리기 전에 Rust를 이전 image로 바꾼다.

BEGIN;

REVOKE UPDATE (source_unit) ON indicator FROM kedp_rust;

COMMIT;
