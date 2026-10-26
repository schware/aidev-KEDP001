-- KEDP DB 스키마 첫 버전 (KEDP001-D022, specification/30.database 4장)
-- 번호는 설계서 6장의 결정 번호다.

BEGIN;

-- 제공처 목록
CREATE TABLE source (
    code text PRIMARY KEY,
    name text NOT NULL
);

INSERT INTO source (code, name) VALUES
    ('ecos', 'ECOS'),
    ('kosis', 'KOSIS'),
    ('data-go-kr', '공공데이터포털');

-- 지표 정의. config/indicators.yaml을 대신한다(KEDP001-D021).
-- 제공처별 코드는 여기에만 둔다(결정 1).
CREATE TABLE indicator (
    -- 화면 주소 /indicator/{id}에도 쓴다
    id text PRIMARY KEY CHECK (id ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    name text NOT NULL,
    source_code text NOT NULL REFERENCES source (code),
    -- KOSIS 기관 코드(orgId)
    org_id text,
    -- ECOS 통계 코드, KOSIS tblId, 공공데이터포털 종목 코드
    stat_code text NOT NULL,
    -- ECOS 항목 코드, KOSIS itmId. 공공데이터포털은 비운다
    item_code text,
    -- KOSIS objL1, objL2… 순서. 분류가 2개 이상인 표는 확인하지 않았다(KEDP001-D022 5절 19)
    classes text[] NOT NULL DEFAULT '{}',
    -- 공공데이터포털에서 지표 값으로 쓰는 항목(결정 8: clpr)
    value_field text,
    -- D/M/Q/A. 연은 A다. KOSIS 요청의 Y는 요청할 때만 쓴다(결정 3)
    cycle text NOT NULL CHECK (cycle IN ('D', 'M', 'Q', 'A')),
    -- 관리자가 넣은 화면 단위. 있으면 source_unit보다 먼저 쓴다(결정 9)
    unit text,
    -- 응답에 있는 단위
    source_unit text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT indicator_ecos_check CHECK (source_code <> 'ecos' OR item_code IS NOT NULL),
    CONSTRAINT indicator_kosis_check CHECK (source_code <> 'kosis' OR (org_id IS NOT NULL AND item_code IS NOT NULL)),
    CONSTRAINT indicator_data_go_kr_check CHECK (source_code <> 'data-go-kr' OR (value_field IS NOT NULL AND item_code IS NULL))
);

-- 화면 설정 (Dashboard 관리)
CREATE TABLE indicator_display (
    indicator_id text PRIMARY KEY REFERENCES indicator (id) ON DELETE CASCADE,
    is_visible boolean NOT NULL DEFAULT false,
    sort_order integer NOT NULL DEFAULT 0,
    -- 상세 화면 차트 기간. 처음 수집할 과거 범위도 이 기간이다(결정 12)
    chart_period interval NOT NULL CHECK (chart_period > interval '0'),
    -- 숫자 표시 형식(자릿수, 소수점). 내용은 화면 작업에서 정한다
    number_format jsonb NOT NULL DEFAULT '{}',
    date_format text,
    -- 상세 화면 맨 위 안내 문구. 없으면 비운다
    notice text
);

-- 관련 지표
CREATE TABLE indicator_relation (
    indicator_id text NOT NULL REFERENCES indicator (id) ON DELETE CASCADE,
    related_indicator_id text NOT NULL REFERENCES indicator (id) ON DELETE CASCADE,
    sort_order integer NOT NULL DEFAULT 0,
    PRIMARY KEY (indicator_id, related_indicator_id),
    CHECK (indicator_id <> related_indicator_id)
);

-- 전체 설정
CREATE TABLE app_setting (
    key text PRIMARY KEY,
    value text NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- 수집 시각 기본값 06:00, Asia/Seoul(KEDP001-D019)
INSERT INTO app_setting (key, value) VALUES ('collect_time', '06:00');

-- 수집 실행 기록. 마지막 수집 일시는 여기서 구한다.
CREATE TABLE collection_run (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    source_code text NOT NULL REFERENCES source (code),
    -- 제공처 전체에 걸친 실행(예: 키 오류)은 비울 수 있다
    indicator_id text REFERENCES indicator (id) ON DELETE SET NULL,
    started_at timestamptz NOT NULL DEFAULT now(),
    finished_at timestamptz,
    -- running: 시작했고 아직 끝나지 않음
    status text NOT NULL CHECK (status IN ('running', 'success', 'no_data', 'error')),
    -- 제공처 오류 코드(예: INFO-100, 11, 30)
    error_code text,
    row_count integer CHECK (row_count >= 0),
    trigger text NOT NULL CHECK (trigger IN ('schedule', 'manual')),
    CHECK (finished_at IS NULL OR finished_at >= started_at),
    CHECK ((status = 'running') = (finished_at IS NULL))
);

CREATE INDEX collection_run_indicator_idx ON collection_run (indicator_id, started_at DESC);
CREATE INDEX collection_run_source_idx ON collection_run (source_code, started_at DESC);

-- 관측값 (화면이 읽는 값). 값이 빈 행(결측)은 넣지 않는다(결정 15).
CREATE TABLE observation (
    indicator_id text NOT NULL REFERENCES indicator (id),
    -- 이전 주기 값을 남기려고 주기를 함께 둔다(결정 11)
    cycle text NOT NULL CHECK (cycle IN ('D', 'M', 'Q', 'A')),
    -- 기간 시작일(설계서 5.1)
    period_start date NOT NULL,
    -- 응답 원문(예: 202401, 2025Q1)
    period_raw text NOT NULL,
    value numeric NOT NULL,
    -- 응답 원문 문자열(결정 10)
    value_raw text NOT NULL,
    -- KOSIS LST_CHN_DE. 다른 제공처는 비운다
    source_changed_at date,
    collected_at timestamptz NOT NULL DEFAULT now(),
    run_id bigint NOT NULL REFERENCES collection_run (id),
    PRIMARY KEY (indicator_id, cycle, period_start),
    -- 기간 시작일은 주기의 첫날이다(설계서 5.1)
    CONSTRAINT observation_period_start_check CHECK (
        CASE cycle
            WHEN 'M' THEN period_start::timestamp = date_trunc('month', period_start::timestamp)
            WHEN 'Q' THEN period_start::timestamp = date_trunc('quarter', period_start::timestamp)
            WHEN 'A' THEN period_start::timestamp = date_trunc('year', period_start::timestamp)
            ELSE true
        END
    )
);

-- 고쳐진 값의 이전 값(결정 4). observation의 트리거가 쓴다.
CREATE TABLE observation_history (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    indicator_id text NOT NULL,
    cycle text NOT NULL,
    period_start date NOT NULL,
    old_value numeric NOT NULL,
    new_value numeric NOT NULL,
    changed_at timestamptz NOT NULL DEFAULT now(),
    -- 새 값을 가져온 수집
    run_id bigint NOT NULL REFERENCES collection_run (id),
    FOREIGN KEY (indicator_id, cycle, period_start) REFERENCES observation (indicator_id, cycle, period_start)
);

CREATE INDEX observation_history_observation_idx ON observation_history (indicator_id, cycle, period_start);

-- 값이 실제로 달라졌을 때만 이전 값을 남긴다(결정 4).
-- 같은 값으로 다시 쓰면(수집 시각, run_id만 바뀜) 남기지 않는다.
CREATE FUNCTION observation_keep_history() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO observation_history (indicator_id, cycle, period_start, old_value, new_value, run_id)
    VALUES (OLD.indicator_id, OLD.cycle, OLD.period_start, OLD.value, NEW.value, NEW.run_id);
    RETURN NULL;
END;
$$;

CREATE TRIGGER observation_keep_history
    AFTER UPDATE OF value ON observation
    FOR EACH ROW
    WHEN (OLD.value IS DISTINCT FROM NEW.value)
    EXECUTE FUNCTION observation_keep_history();

-- 원본 응답(결정 5, 13). 오류·데이터 없음 응답도 넣는다. 90일 보관(결정 6)이며 삭제 작업은 MVP 이후에 만든다.
CREATE TABLE raw_response (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    run_id bigint NOT NULL REFERENCES collection_run (id),
    received_at timestamptz NOT NULL,
    -- 키 자리를 ***로 가린 요청 주소. 가리지 않은 주소가 들어오지 않도록 *** 가 있어야 한다
    request_masked text NOT NULL CHECK (position('***' IN request_masked) > 0),
    http_status integer NOT NULL CHECK (http_status BETWEEN 100 AND 599),
    -- JSON 응답은 body, JSON이 아닌 응답은 body_text에 넣는다. 둘 중 하나만 쓴다
    body jsonb,
    body_text text,
    CHECK (num_nonnulls(body, body_text) = 1)
);

CREATE INDEX raw_response_run_idx ON raw_response (run_id);
CREATE INDEX raw_response_received_at_idx ON raw_response (received_at);

COMMIT;
