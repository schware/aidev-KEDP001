-- 지표 구분(Issue #52). 구분 9개와 지표의 구분(indicator.category_code).
-- 0003_visits.up.sql 뒤에 적용한다(kedp_go 계정이 있어야 한다).
--
-- 지금 있는 지표 7개는 여기서 구분을 채운다. 그 밖의 지표가 있으면 짐작해서 채우지 않고 멈춘다.
-- 한 transaction이라 멈추면 아무것도 바뀌지 않는다. 적용 전 확인 SQL은 b-sql-db/README.md에 있다.

BEGIN;

-- 구분 목록. 바꿀 때는 migration으로 바꾼다(kedp_go는 읽기만 한다).
CREATE TABLE category (
    code text PRIMARY KEY,
    name text NOT NULL,
    -- 화면에서 구분을 놓는 순서. 작은 것이 앞이다
    sort_order integer NOT NULL
);

INSERT INTO category (code, name, sort_order) VALUES
    ('growth', '성장·경기', 1),
    ('price', '물가', 2),
    ('labor', '고용·소득', 3),
    ('money', '금리·통화', 4),
    ('credit', '부채·금융안정', 5),
    ('external', '대외경제', 6),
    ('housing', '주택', 7),
    ('population', '인구', 8),
    ('market', '금융시장', 9);

ALTER TABLE indicator
    ADD COLUMN category_code text CONSTRAINT indicator_category_code_fkey REFERENCES category (code);

UPDATE indicator i SET category_code = v.code
FROM (VALUES
    ('ecos-base-rate', 'money'),
    ('kosis-employment-rate', 'labor'),
    ('kosis-total-population', 'population'),
    ('kosis-births', 'population'),
    ('data-go-kr-kodex-200', 'market'),
    ('data-go-kr-kodex-leverage', 'market'),
    ('data-go-kr-kodex-inverse', 'market')
) AS v (id, code)
WHERE i.id = v.id;

-- 구분을 채우지 못한 지표가 있으면 여기서 멈춘다(SET NOT NULL이 실패한다). 어느 지표인지 먼저 알린다.
DO $$
DECLARE
    ids text := (SELECT string_agg(id, ', ' ORDER BY id) FROM indicator WHERE category_code IS NULL);
BEGIN
    IF ids IS NOT NULL THEN
        RAISE NOTICE 'indicator without category: %', ids;
    END IF;
END
$$;

ALTER TABLE indicator ALTER COLUMN category_code SET NOT NULL;

-- Go: 구분 목록을 읽기만 한다. 지표의 구분(indicator.category_code)은 indicator 권한(0002)으로 쓴다.
GRANT SELECT ON category TO kedp_go;

COMMIT;
