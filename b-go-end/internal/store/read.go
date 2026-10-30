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

// ErrNotFound는 지표가 없다는 오류다.
var ErrNotFound = errors.New("not found")

// Point는 관측값 하나다. Period는 기간 시작일이다.
type Point struct {
	Period time.Time
	Value  float64
}

// Card는 메인 화면 카드 하나다. 관측값이 없으면 Latest가 nil이다.
type Card struct {
	ID, Name, Cycle string
	Unit            *string
	Latest          *Point
}

const unitExpr = `coalesce(nullif(i.unit, ''), nullif(i.source_unit, ''))`

// latestJoin은 지표 i의 마지막 관측값 o(value, period_start)다.
const latestJoin = `
	LEFT JOIN LATERAL (
		SELECT o.value::float8 AS value, o.period_start
		FROM observation o
		WHERE o.indicator_id = i.id AND o.cycle = i.cycle
		ORDER BY o.period_start DESC
		LIMIT 1
	) o ON true`

// Cards는 노출(is_visible)된 지표를 순서(sort_order, 그다음 id)대로 읽는다.
func (s *Store) Cards(ctx context.Context) ([]Card, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT i.id, i.name, i.cycle, `+unitExpr+`, o.value, o.period_start
		FROM indicator i
		JOIN indicator_display d ON d.indicator_id = i.id`+latestJoin+`
		WHERE d.is_visible
		ORDER BY d.sort_order, i.id`)
	if err != nil {
		return nil, wrap("read cards", err)
	}
	list, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Card, error) {
		var c Card
		var value *float64
		var period *time.Time
		err := row.Scan(&c.ID, &c.Name, &c.Cycle, &c.Unit, &value, &period)
		c.Latest = point(value, period)
		return c, err
	})
	if err != nil {
		return nil, wrap("read cards", err)
	}
	return list, nil
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
	ID, Name, Cycle, StatCode string
	SourceCode, SourceName    string
	Unit, Notice              *string
	// ChartPeriod는 차트 기간(PostgreSQL interval 글, 예: 10 years)이다. 화면 설정이 없으면 nil이다.
	ChartPeriod *string
	Latest      *Point
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
	var value *float64
	var period *time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT i.id, i.name, i.cycle, i.stat_code, src.code, src.name, `+unitExpr+`,
		       d.notice, d.chart_period::text, o.value, o.period_start,
		       (SELECT max(r.finished_at) FROM collection_run r
		        WHERE r.indicator_id = i.id AND r.status = 'success')
		FROM indicator i
		JOIN source src ON src.code = i.source_code
		LEFT JOIN indicator_display d ON d.indicator_id = i.id`+latestJoin+`
		WHERE i.id = $1`, id).Scan(
		&d.ID, &d.Name, &d.Cycle, &d.StatCode, &d.SourceCode, &d.SourceName, &d.Unit,
		&d.Notice, &d.ChartPeriod, &value, &period, &d.LastCollectedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Detail{}, ErrNotFound
	}
	if err != nil {
		return Detail{}, wrap("read indicator", err)
	}
	d.Latest = point(value, period)

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
