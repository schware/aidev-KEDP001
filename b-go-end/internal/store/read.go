package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// 조회 API(사용자 화면)가 읽는 값이다. 읽기만 한다.
// 단위는 관리자 단위(unit)가 있으면 그것, 없으면 응답 단위(source_unit), 둘 다 없으면 nil이다(KEDP001-D022 9).
// 관측값은 지표의 지금 주기(indicator.cycle) 것만 읽는다(KEDP001-D022 11).
// 구분(category)은 code와 이름을 함께 읽는다(Issue #52).
// 직전 값(previous)과 차이(change)는 읽을 때 계산하고 저장하지 않는다. 증감률은 계산하지 않는다(KEDP001-D024, Issue #70).

// ErrNotFound는 지표가 없다는 오류다.
var ErrNotFound = errors.New("not found")

// Point는 관측값 하나다. Period는 기간 시작일이다.
type Point struct {
	Period time.Time
	Value  float64
}

// Card는 메인 화면 카드 하나다. 관측값이 없으면 Latest가 nil이다.
// Previous, Change는 previousJoin을 본다.
type Card struct {
	ID, Name, Cycle string
	// CategoryCode, CategoryName은 지표의 구분이다.
	CategoryCode, CategoryName string
	Unit                       *string
	// Decimals는 소수 자릿수(indicator_display.number_format의 decimals, 0~4)다. 없으면 nil이다.
	Decimals *int
	Latest   *Point
	Previous *Point
	Change   *float64
}

const unitExpr = `coalesce(nullif(i.unit, ''), nullif(i.source_unit, ''))`

// decimalsExpr는 화면 설정 d의 소수 자릿수다. 숫자가 아니면 NULL이다.
const decimalsExpr = `CASE WHEN jsonb_typeof(d.number_format->'decimals') = 'number'
		       THEN (d.number_format->>'decimals')::numeric::int END`

// latestJoin은 지표 i의 마지막 관측값 o(value numeric, period_start)와 그 직전 값 p다.
// p는 M, Q, A면 o의 바로 앞 기간(1개월, 3개월, 1년 전) 관측값이고, 그 기간 값이 없으면 없다(NULL).
// D면 o 바로 앞의 관측값이다(주말·휴일은 관측값이 없으므로 건너뛴다). 관측값이 하나뿐이면 없다.
const latestJoin = `
	LEFT JOIN LATERAL (
		SELECT o.value, o.period_start
		FROM observation o
		WHERE o.indicator_id = i.id AND o.cycle = i.cycle
		ORDER BY o.period_start DESC
		LIMIT 1
	) o ON true
	LEFT JOIN LATERAL (
		SELECT p.value, p.period_start
		FROM observation p
		WHERE p.indicator_id = i.id AND p.cycle = i.cycle
		  AND CASE i.cycle
		        WHEN 'D' THEN p.period_start < o.period_start
		        ELSE p.period_start = (o.period_start - CASE i.cycle
		                                 WHEN 'M' THEN interval '1 month'
		                                 WHEN 'Q' THEN interval '3 months'
		                                 ELSE interval '1 year' END)::date
		      END
		ORDER BY p.period_start DESC
		LIMIT 1
	) p ON true`

// latestCols는 latestJoin의 최신값, 직전 값, 차이다. 차이는 numeric으로 빼서 반올림 오차가 없게 한 뒤 float8로 바꾼다.
const latestCols = `o.value::float8, o.period_start, p.value::float8, p.period_start, (o.value - p.value)::float8`

// Cards는 노출(is_visible)된 지표를 구분 순서(category.sort_order), 그다음 화면 순서(sort_order, id)대로 읽는다.
// 화면은 받은 순서대로 구분을 묶는다.
func (s *Store) Cards(ctx context.Context) ([]Card, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT i.id, i.name, i.cycle, c.code, c.name, `+unitExpr+`, `+decimalsExpr+`, `+latestCols+`
		FROM indicator i
		JOIN category c ON c.code = i.category_code
		JOIN indicator_display d ON d.indicator_id = i.id`+latestJoin+`
		WHERE d.is_visible
		ORDER BY c.sort_order, d.sort_order, i.id`)
	if err != nil {
		return nil, wrap("read cards", err)
	}
	list, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Card, error) {
		var c Card
		var l latest
		err := row.Scan(&c.ID, &c.Name, &c.Cycle, &c.CategoryCode, &c.CategoryName, &c.Unit, &c.Decimals,
			&l.value, &l.period, &l.prevValue, &l.prevPeriod, &l.change)
		c.Latest, c.Previous, c.Change = l.points()
		return c, err
	})
	if err != nil {
		return nil, wrap("read cards", err)
	}
	return list, nil
}

// latest는 latestCols를 받는 곳이다.
type latest struct {
	value, prevValue, change *float64
	period, prevPeriod       *time.Time
}

// points는 최신값, 직전 값, 차이다. 직전 값이 없으면 SQL에서 차이도 NULL이라 nil이다.
func (l latest) points() (cur, prev *Point, change *float64) {
	return point(l.value, l.period), point(l.prevValue, l.prevPeriod), l.change
}

func point(value *float64, period *time.Time) *Point {
	if value == nil || period == nil {
		return nil
	}
	return &Point{Period: *period, Value: *value}
}

// Related는 관련 지표 하나다.
type Related struct{ ID, Name string }

// Detail은 지표 상세 화면의 값이다.
type Detail struct {
	ID, Name, Cycle, StatCode  string
	SourceCode, SourceName     string
	CategoryCode, CategoryName string
	Unit, Notice               *string
	// Decimals는 소수 자릿수다(Card와 같다).
	Decimals *int
	// ChartPeriod는 차트 기간(PostgreSQL interval 글, 예: 10 years)이다. 화면 설정이 없으면 nil이다.
	ChartPeriod *string
	Latest      *Point
	// Previous, Change는 Card와 같다.
	Previous *Point
	Change   *float64
	// LastCollectedAt은 status가 success인 마지막 수집의 끝난 시각이다. 없으면 nil이다.
	LastCollectedAt *time.Time
	// Chart는 차트 기간 안의 관측값이다(오래된 것부터). 화면 설정이 없으면 비어 있다.
	Chart []Point
	// Recent는 최근 관측값이다(최근 것부터).
	Recent  []Point
	Related []Related
}

// Detail은 지표 하나의 상세 값을 읽는다. 지표가 없으면 ErrNotFound다.
// today는 Asia/Seoul의 오늘이다. 차트는 기간 시작일이 (today - 차트 기간) 이후인 관측값이다.
// recent는 최근 데이터 표의 행 수다.
func (s *Store) Detail(ctx context.Context, id string, today time.Time, recent int) (Detail, error) {
	var d Detail
	var l latest
	err := s.pool.QueryRow(ctx, `
		SELECT i.id, i.name, i.cycle, i.stat_code, src.code, src.name, c.code, c.name, `+unitExpr+`, `+decimalsExpr+`,
		       d.notice, d.chart_period::text, `+latestCols+`,
		       (SELECT max(r.finished_at) FROM collection_run r
		        WHERE r.indicator_id = i.id AND r.status = 'success')
		FROM indicator i
		JOIN source src ON src.code = i.source_code
		JOIN category c ON c.code = i.category_code
		LEFT JOIN indicator_display d ON d.indicator_id = i.id`+latestJoin+`
		WHERE i.id = $1`, id).Scan(
		&d.ID, &d.Name, &d.Cycle, &d.StatCode, &d.SourceCode, &d.SourceName, &d.CategoryCode, &d.CategoryName, &d.Unit, &d.Decimals,
		&d.Notice, &d.ChartPeriod, &l.value, &l.period, &l.prevValue, &l.prevPeriod, &l.change, &d.LastCollectedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Detail{}, ErrNotFound
	}
	if err != nil {
		return Detail{}, wrap("read indicator", err)
	}
	d.Latest, d.Previous, d.Change = l.points()

	if d.Chart, err = s.points(ctx, `
		SELECT o.period_start, o.value::float8
		FROM observation o
		JOIN indicator i ON i.id = o.indicator_id AND i.cycle = o.cycle
		JOIN indicator_display d ON d.indicator_id = i.id
		WHERE i.id = $1 AND o.period_start >= ($2::text::date - d.chart_period)::date
		ORDER BY o.period_start`, id, today.Format(time.DateOnly)); err != nil {
		return Detail{}, err
	}
	if d.Recent, err = s.points(ctx, `
		SELECT o.period_start, o.value::float8
		FROM observation o
		JOIN indicator i ON i.id = o.indicator_id AND i.cycle = o.cycle
		WHERE i.id = $1
		ORDER BY o.period_start DESC
		LIMIT $2`, id, recent); err != nil {
		return Detail{}, err
	}

	rows, err := s.pool.Query(ctx, `
		SELECT i.id, i.name
		FROM indicator_relation r
		JOIN indicator i ON i.id = r.related_indicator_id
		WHERE r.indicator_id = $1
		ORDER BY r.sort_order, i.id`, id)
	if err != nil {
		return Detail{}, wrap("read related", err)
	}
	if d.Related, err = pgx.CollectRows(rows, pgx.RowToStructByPos[Related]); err != nil {
		return Detail{}, wrap("read related", err)
	}
	return d, nil
}

func (s *Store) points(ctx context.Context, sql string, args ...any) ([]Point, error) {
	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, wrap("read observation", err)
	}
	list, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Point])
	if err != nil {
		return nil, wrap("read observation", err)
	}
	return list, nil
}
