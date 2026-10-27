-- DB 테스트가 끝난 뒤 superuser로 실행한다(scripts/db-test.sh).
-- kedp_rust는 observation_history를 읽을 수 없어서, 트리거가 남긴 이력은 여기서 확인한다.
-- tests/db.rs의 ecos_insert_unchanged_updated: 202401 값이 3.50 → 3.25로 바뀐 것 1건만 남아야 한다.

DO $$
DECLARE
    n integer;
BEGIN
    SELECT count(*) INTO n FROM observation_history;
    IF n <> 1 THEN
        RAISE EXCEPTION 'observation_history: want 1 row, got %', n;
    END IF;
    PERFORM 1 FROM observation_history
    WHERE indicator_id = 'ecos-base-rate' AND cycle = 'M' AND period_start = '2024-01-01'
      AND old_value = 3.5 AND new_value = 3.25 AND run_id = 3;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'observation_history: the row is not 202401 3.5 -> 3.25 (run 3)';
    END IF;
    RAISE NOTICE 'ok: observation_history has 1 row (202401 3.5 -> 3.25, run 3)';
END
$$;
