-- 0001, 0002 migration을 적용한 시험용 DB에서 superuser로 실행한다.
--   psql -v ON_ERROR_STOP=1 -d <시험용 DB> -f b-sql-db/checks/smoke.sql
-- 계정 권한, 기간 시작일 검사, 이력 트리거, 원본 주소 검사를 확인한다.
-- 끝에 ROLLBACK하므로 데이터가 남지 않는다. 기대와 다르면 오류로 멈춘다.

\set ON_ERROR_STOP on
BEGIN;

-- Go: 지표 정의, 화면 설정, 수집 기록을 쓴다
SET LOCAL ROLE kedp_go;
INSERT INTO indicator (id, name, source_code, stat_code, item_code, cycle, source_unit)
VALUES ('ecos-base-rate', '한국은행 기준금리', 'ecos', '722Y001', '0101000', 'M', '연%');
INSERT INTO indicator_display (indicator_id, chart_period) VALUES ('ecos-base-rate', '10 years');
INSERT INTO collection_run (source_code, indicator_id, status, trigger)
VALUES ('ecos', 'ecos-base-rate', 'running', 'manual')
RETURNING id AS run_id \gset

DO $$ BEGIN
    INSERT INTO observation (indicator_id, cycle, period_start, period_raw, value, value_raw, run_id)
    VALUES ('ecos-base-rate', 'M', '2024-01-01', '202401', 1, '1', 1);
    RAISE EXCEPTION 'kedp_go가 observation에 썼다';
EXCEPTION WHEN insufficient_privilege THEN RAISE NOTICE 'ok: kedp_go는 observation에 쓰지 못한다';
END $$;

-- Rust: 관측값과 원본 응답을 쓴다
SET LOCAL ROLE kedp_rust;
INSERT INTO observation (indicator_id, cycle, period_start, period_raw, value, value_raw, run_id)
VALUES ('ecos-base-rate', 'M', '2024-01-01', '202401', 3.5, '3.5', :run_id);
-- 같은 값으로 다시 쓴다: 이력이 남지 않아야 한다
INSERT INTO observation (indicator_id, cycle, period_start, period_raw, value, value_raw, run_id)
VALUES ('ecos-base-rate', 'M', '2024-01-01', '202401', 3.50, '3.50', :run_id)
ON CONFLICT (indicator_id, cycle, period_start)
DO UPDATE SET value = EXCLUDED.value, value_raw = EXCLUDED.value_raw, collected_at = now(), run_id = EXCLUDED.run_id;
-- 다른 값으로 다시 쓴다: 이력이 1건 남아야 한다
INSERT INTO observation (indicator_id, cycle, period_start, period_raw, value, value_raw, run_id)
VALUES ('ecos-base-rate', 'M', '2024-01-01', '202401', 3.25, '3.25', :run_id)
ON CONFLICT (indicator_id, cycle, period_start)
DO UPDATE SET value = EXCLUDED.value, value_raw = EXCLUDED.value_raw, collected_at = now(), run_id = EXCLUDED.run_id;
INSERT INTO raw_response (run_id, received_at, request_masked, http_status, body)
VALUES (:run_id, now(), 'https://ecos.bok.or.kr/api/StatisticSearch/***/json/kr/1/1000/722Y001/M/202401/202401/0101000', 200, '{}');

DO $$ BEGIN
    INSERT INTO indicator (id, name, source_code, stat_code, item_code, cycle)
    VALUES ('x', 'x', 'ecos', 'x', 'x', 'M');
    RAISE EXCEPTION 'kedp_rust가 indicator에 썼다';
EXCEPTION WHEN insufficient_privilege THEN RAISE NOTICE 'ok: kedp_rust는 indicator에 쓰지 못한다';
END $$;

RESET ROLE;

DO $$
DECLARE
    n integer;
    run bigint := (SELECT max(id) FROM collection_run);
BEGIN
    SELECT count(*) INTO n FROM observation_history;
    IF n <> 1 THEN
        RAISE EXCEPTION 'observation_history % rows, want 1', n;
    END IF;
    IF NOT EXISTS (SELECT FROM observation_history WHERE old_value = 3.5 AND new_value = 3.25) THEN
        RAISE EXCEPTION 'observation_history has wrong values';
    END IF;
    RAISE NOTICE 'ok: 값이 달라졌을 때만 이력이 남는다';

    -- 분기 값의 기간 시작일은 분기 첫날이어야 한다
    INSERT INTO observation (indicator_id, cycle, period_start, period_raw, value, value_raw, run_id)
    VALUES ('ecos-base-rate', 'Q', '2025-04-01', '2025Q2', 2.5, '2.5', run);
    BEGIN
        INSERT INTO observation (indicator_id, cycle, period_start, period_raw, value, value_raw, run_id)
        VALUES ('ecos-base-rate', 'Q', '2025-05-01', '2025Q2', 2.5, '2.5', run);
        RAISE EXCEPTION '분기 중간 날짜가 들어갔다';
    EXCEPTION WHEN check_violation THEN RAISE NOTICE 'ok: 분기 기간 시작일 검사';
    END;

    -- 가리지 않은 요청 주소는 넣지 못한다
    BEGIN
        INSERT INTO raw_response (run_id, received_at, request_masked, http_status, body)
        VALUES (run, now(), 'https://ecos.bok.or.kr/api/StatisticSearch/NOTMASKED/json', 200, '{}');
        RAISE EXCEPTION '가리지 않은 주소가 들어갔다';
    EXCEPTION WHEN check_violation THEN RAISE NOTICE 'ok: 원본 주소 *** 검사';
    END;
END
$$;

ROLLBACK;
