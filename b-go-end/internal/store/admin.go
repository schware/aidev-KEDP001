package store

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// 관리자 API(internal/admin)가 읽고 쓰는 값이다(KEDP001-D018, specification/40.api 5장).
// 입력 값의 형식은 admin이 먼저 검사한다. 여기서는 DB에 있는지(id, 관련 지표)만 본다.

// ErrExists는 같은 id의 지표가 이미 있다는 오류다.
var ErrExists = errors.New("already exists")

// FieldError는 요청의 한 칸이 DB 내용과 맞지 않는다는 오류다(예: 없는 관련 지표).
type FieldError struct{ Field, Reason string }

func (e *FieldError) Error() string { return e.Field + ": " + e.Reason }

// AdminIndicator는 관리자 지표 목록의 지표 하나다. 화면 설정이 없으면 Visible은 false, Order와 ChartPeriod는 nil이다.
type AdminIndicator struct {
	ID, Name, Source, StatCode, Cycle string
	OrgID, ItemCode, ValueField       *string
	Classes                           []string
	Unit, SourceUnit                  *string
	HasDisplay, Visible               bool
	Order                             *int
	// ChartPeriod는 차트 기간이다(N years, N months, N days). 그렇게 쓸 수 없는 기간은 PostgreSQL interval 글이다.
	ChartPeriod *string
	Decimals    *int
	Notice      *string
	// Related는 관련 지표 id다(관리자가 정한 순서).
	Related []string
}

// AdminIndicators는 지표를 모두 읽는다(숨긴 지표, 화면 설정이 없는 지표도). 순서는 화면 순서, 그다음 id다.
func (s *Store) AdminIndicators(ctx context.Context) ([]AdminIndicator, error) {
	return s.adminIndicators(ctx, s.pool, "")
}

// AdminIndicator는 지표 하나를 읽는다. 없으면 ErrNotFound다.
func (s *Store) AdminIndicator(ctx context.Context, id string) (AdminIndicator, error) {
	list, err := s.adminIndicators(ctx, s.pool, id)
	if err != nil {
		return AdminIndicator{}, err
	}
	if len(list) == 0 {
		return AdminIndicator{}, ErrNotFound
	}
	return list[0], nil
}

type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// adminIndicators는 id가 비어 있으면 모든 지표, 아니면 그 지표만 읽는다.
func (s *Store) adminIndicators(ctx context.Context, q querier, id string) ([]AdminIndicator, error) {
	rows, err := q.Query(ctx, `
		SELECT i.id, i.name, i.source_code, i.stat_code, i.cycle, i.org_id, i.item_code, i.value_field,
		       i.classes, i.unit, i.source_unit,
		       d.indicator_id IS NOT NULL, coalesce(d.is_visible, false), d.sort_order, d.chart_period,
		       `+decimalsExpr+`, d.notice,
		       coalesce((SELECT array_agg(r.related_indicator_id ORDER BY r.sort_order, r.related_indicator_id)
		                 FROM indicator_relation r WHERE r.indicator_id = i.id), '{}')
		FROM indicator i
		LEFT JOIN indicator_display d ON d.indicator_id = i.id
		WHERE $1 = '' OR i.id = $1
		ORDER BY d.sort_order NULLS LAST, i.id`, id)
	if err != nil {
		return nil, wrap("read indicators", err)
	}
	list, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (AdminIndicator, error) {
		var a AdminIndicator
		var period pgtype.Interval
		err := row.Scan(&a.ID, &a.Name, &a.Source, &a.StatCode, &a.Cycle, &a.OrgID, &a.ItemCode, &a.ValueField,
			&a.Classes, &a.Unit, &a.SourceUnit,
			&a.HasDisplay, &a.Visible, &a.Order, &period, &a.Decimals, &a.Notice, &a.Related)
		if period.Valid {
			p := FormatPeriod(period)
			a.ChartPeriod = &p
		}
		return a, err
	})
	if err != nil {
		return nil, wrap("read indicators", err)
	}
	return list, nil
}

// FormatPeriod는 차트 기간을 관리자 API 형식(N years, N months, N days)으로 쓴다.
// 년·월과 일이 섞였거나 시간이 있으면 그렇게 쓸 수 없으므로 PostgreSQL interval 글에 가깝게 쓴다.
func FormatPeriod(p pgtype.Interval) string {
	switch {
	case p.Microseconds == 0 && p.Days == 0 && p.Months%12 == 0:
		return fmt.Sprintf("%d years", p.Months/12)
	case p.Microseconds == 0 && p.Days == 0:
		return fmt.Sprintf("%d months", p.Months)
	case p.Microseconds == 0 && p.Months == 0:
		return fmt.Sprintf("%d days", p.Days)
	}
	return fmt.Sprintf("%d mons %d days %s", p.Months, p.Days, time.Duration(p.Microseconds)*time.Microsecond)
}

// NewIndicator는 새 지표다. 빈 글자는 NULL로 넣는다(org_id, item_code, value_field, unit, notice).
type NewIndicator struct {
	ID, Name, Source, StatCode, Cycle string
	OrgID, ItemCode, ValueField, Unit string
	Classes                           []string
	Visible                           bool
	ChartPeriod                       string
	Decimals                          *int
	Notice                            string
	Related                           []string
}

// AddIndicator는 지표와 화면 설정, 관련 지표를 한 transaction으로 넣는다. 순서는 맨 뒤다.
// 같은 id가 있으면 ErrExists, 없는 관련 지표가 있으면 *FieldError다.
func (s *Store) AddIndicator(ctx context.Context, n NewIndicator) error {
	classes := n.Classes
	if classes == nil {
		classes = []string{}
	}
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO indicator (id, name, source_code, org_id, stat_code, item_code, classes, value_field, cycle, unit)
			VALUES ($1, $2, $3, NULLIF($4, ''), $5, NULLIF($6, ''), $7, NULLIF($8, ''), $9, NULLIF($10, ''))`,
			n.ID, n.Name, n.Source, n.OrgID, n.StatCode, n.ItemCode, classes, n.ValueField, n.Cycle, n.Unit)
		if isCode(err, "23505") {
			return ErrExists
		}
		if err != nil {
			return wrap("insert indicator", err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO indicator_display (indicator_id, is_visible, sort_order, chart_period, number_format, notice)
			VALUES ($1, $2, (SELECT coalesce(max(sort_order), 0) + 1 FROM indicator_display),
			        $3::text::interval, `+numberFormat(`'{}'::jsonb`, "$4")+`, NULLIF($5, ''))`,
			n.ID, n.Visible, n.ChartPeriod, n.Decimals, n.Notice); err != nil {
			return wrap("insert indicator_display", err)
		}
		return setRelated(ctx, tx, n.ID, n.Related)
	})
}

// numberFormat은 number_format(base)의 decimals를 param(정수 또는 NULL)로 바꾸는 식이다. 다른 칸은 그대로 둔다.
func numberFormat(base, param string) string {
	return fmt.Sprintf(`CASE WHEN %[2]s::int IS NULL THEN %[1]s - 'decimals'
		ELSE %[1]s || jsonb_build_object('decimals', %[2]s::int) END`, base, param)
}

// Edit는 지표에서 바꿀 수 있는 값이다. id, 제공처, 코드, 노출, 순서는 바꾸지 않는다.
type Edit struct {
	Name, Cycle, Unit string
	ChartPeriod       string
	Decimals          *int
	Notice            string
	Related           []string
}

// UpdateIndicator는 지표 id를 e로 고친다. 관련 지표는 e.Related로 바꾼다. 한 transaction이다.
// 화면 설정이 없던 지표는 노출 안 함, 맨 뒤 순서로 화면 설정을 만든다.
// 지표가 없으면 ErrNotFound, 없는 관련 지표가 있으면 *FieldError다.
// 주기를 바꿔도 이전 주기의 관측값은 남는다(KEDP001-D022 11).
func (s *Store) UpdateIndicator(ctx context.Context, id string, e Edit) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE indicator SET name = $2, cycle = $3, unit = NULLIF($4, ''), updated_at = now()
			WHERE id = $1`, id, e.Name, e.Cycle, e.Unit)
		if err != nil {
			return wrap("update indicator", err)
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO indicator_display (indicator_id, is_visible, sort_order, chart_period, number_format, notice)
			VALUES ($1, false, (SELECT coalesce(max(sort_order), 0) + 1 FROM indicator_display),
			        $2::text::interval, `+numberFormat(`'{}'::jsonb`, "$3")+`, NULLIF($4, ''))
			ON CONFLICT (indicator_id) DO UPDATE SET
				chart_period = excluded.chart_period,
				number_format = `+numberFormat("indicator_display.number_format", "$3")+`,
				notice = excluded.notice`,
			id, e.ChartPeriod, e.Decimals, e.Notice); err != nil {
			return wrap("update indicator_display", err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM indicator_relation WHERE indicator_id = $1`, id); err != nil {
			return wrap("update indicator_relation", err)
		}
		return setRelated(ctx, tx, id, e.Related)
	})
}

// setRelated는 관련 지표를 related 순서대로 넣는다. 없는 지표가 있으면 *FieldError다.
func setRelated(ctx context.Context, tx pgx.Tx, id string, related []string) error {
	for i, r := range related {
		_, err := tx.Exec(ctx, `
			INSERT INTO indicator_relation (indicator_id, related_indicator_id, sort_order) VALUES ($1, $2, $3)`,
			id, r, i+1)
		if isCode(err, "23503") {
			return &FieldError{"related", fmt.Sprintf("indicator %q does not exist", r)}
		}
		if err != nil {
			return wrap("insert indicator_relation", err)
		}
	}
	return nil
}

// OrderItem은 지표 하나의 노출과 순서다.
type OrderItem struct {
	ID      string
	Visible bool
	Order   int
}

// SaveOrder는 화면 설정이 있는 지표 전체의 노출과 순서를 한 transaction으로 저장한다.
// items에는 화면 설정이 있는 지표가 모두 한 번씩 있어야 한다. 아니면 *FieldError이고 아무것도 바꾸지 않는다.
func (s *Store) SaveOrder(ctx context.Context, items []OrderItem) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		// 저장하는 동안 다른 요청이 화면 설정을 고치지 않게 잠근다.
		rows, err := tx.Query(ctx, `SELECT indicator_id FROM indicator_display ORDER BY indicator_id FOR UPDATE`)
		if err != nil {
			return wrap("read indicator_display", err)
		}
		ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
		if err != nil {
			return wrap("read indicator_display", err)
		}
		seen := map[string]bool{}
		for _, it := range items {
			if !slices.Contains(ids, it.ID) {
				return &FieldError{"indicators", fmt.Sprintf("indicator %q does not exist or has no display settings", it.ID)}
			}
			seen[it.ID] = true
		}
		for _, id := range ids {
			if !seen[id] {
				return &FieldError{"indicators", fmt.Sprintf("indicator %q is missing", id)}
			}
		}
		for _, it := range items {
			if _, err := tx.Exec(ctx, `
				UPDATE indicator_display SET is_visible = $2, sort_order = $3 WHERE indicator_id = $1`,
				it.ID, it.Visible, it.Order); err != nil {
				return wrap("update indicator_display", err)
			}
		}
		return nil
	})
}

// SetCollectTime은 수집 시각(collect_time, HH:MM)을 바꾸고 이전 값을 돌려준다. 형식은 admin이 검사한다.
func (s *Store) SetCollectTime(ctx context.Context, v string) (string, error) {
	var old string
	err := s.pool.QueryRow(ctx, `
		WITH old AS (SELECT value FROM app_setting WHERE key = 'collect_time')
		INSERT INTO app_setting (key, value) VALUES ('collect_time', $1)
		ON CONFLICT (key) DO UPDATE SET value = excluded.value, updated_at = now()
		RETURNING coalesce((SELECT value FROM old), '')`, v).Scan(&old)
	if err != nil {
		return "", wrap("update collect_time", err)
	}
	return old, nil
}

// SourceRun은 제공처 하나와 그 제공처의 마지막으로 끝난 수집 기록이다. 기록이 없으면 Status가 비어 있고 FinishedAt이 nil이다.
type SourceRun struct {
	Code, Name string
	// Status는 success, no_data, error 중 하나다(running은 끝나지 않았으므로 보지 않는다).
	Status     string
	FinishedAt *time.Time
}

// SourceRuns는 제공처마다 마지막으로 끝난 수집 기록을 읽는다(제공처 code 순서).
func (s *Store) SourceRuns(ctx context.Context) ([]SourceRun, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT src.code, src.name, coalesce(r.status, ''), r.finished_at
		FROM source src
		LEFT JOIN LATERAL (
			SELECT r.status, r.finished_at
			FROM collection_run r
			WHERE r.source_code = src.code AND r.finished_at IS NOT NULL
			ORDER BY r.finished_at DESC, r.id DESC
			LIMIT 1
		) r ON true
		ORDER BY src.code`)
	if err != nil {
		return nil, wrap("read collection_run", err)
	}
	list, err := pgx.CollectRows(rows, pgx.RowToStructByPos[SourceRun])
	if err != nil {
		return nil, wrap("read collection_run", err)
	}
	return list, nil
}

func isCode(err error, code string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code
}
