// Package store는 Go가 쓰는 DB 테이블을 다룬다. kedp_go 계정으로 접속한다(KEDP001-D022 7, 14).
//
//   - 쓰는 테이블: indicator, indicator_display(처음 값), collection_run
//   - 읽는 테이블: app_setting(collect_time), observation(마지막 기간)
//   - 조회 API(read.go): indicator, indicator_display, indicator_relation, source, observation, collection_run을 읽기만 한다
//
// 오류와 기록에는 접속 정보(주소, 포트, DB 이름, 비밀번호)를 넣지 않는다.
package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/schware/aidev-KEDP001/b-go-end/internal/config"
)

// collection_run.status, trigger 값(b-sql-db/migrations/0001_schema.up.sql)
const (
	StatusSuccess = "success"
	StatusNoData  = "no_data"
	StatusError   = "error"

	TriggerSchedule = "schedule"
	TriggerManual   = "manual"
)

// Store는 DB 연결 pool이다.
type Store struct{ pool *pgxpool.Pool }

// Open은 DB에 접속하고 접속되는지 확인한다.
func Open(ctx context.Context, db config.DB) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(dsn(db))
	if err != nil {
		// 파싱 오류에는 접속 문자열이 들어갈 수 있어 감싸지 않는다.
		return nil, errors.New("DB settings are invalid (KEDP_DB_HOST, KEDP_DB_PORT, KEDP_DB_NAME, kedp_go)")
	}
	cfg.MaxConns = 2
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, wrap("connect DB", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, wrap("connect DB", err)
	}
	return &Store{pool: pool}, nil
}

// Close는 연결을 닫는다.
func (s *Store) Close() { s.pool.Close() }

// dsn은 keyword=value 형식의 접속 문자열이다. 값은 따옴표로 감싸고 '와 \는 \로 막는다.
func dsn(db config.DB) string {
	q := func(v string) string {
		return "'" + strings.NewReplacer(`\`, `\\`, `'`, `\'`).Replace(v) + "'"
	}
	return fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s connect_timeout=5",
		q(db.Host), db.Port, q(db.Name), q(config.DBUser), q(db.Password))
}

// wrap은 DB 오류에서 접속 정보를 뺀다. 접속 오류(pgconn.ConnectError)의 글에는 주소, 포트, DB 이름이 들어 있다.
// DB가 돌려준 오류(예: 권한 없음, 비밀번호 틀림)는 그 메시지만 쓴다. 이 메시지에는 비밀번호가 없다.
func wrap(what string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return fmt.Errorf("%s: %s (SQLSTATE %s)", what, pgErr.Message, pgErr.Code)
	}
	var connErr *pgconn.ConnectError
	if errors.As(err, &connErr) {
		return fmt.Errorf("%s: cannot connect (check KEDP_DB_HOST, KEDP_DB_PORT, KEDP_DB_NAME)", what)
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("%s: %w", what, err)
	}
	return fmt.Errorf("%s: %s", what, err.Error())
}

// Target은 수집할 지표 하나와 수집 범위를 정하는 값이다.
type Target struct {
	Indicator config.Indicator
	// LastPeriod는 이 지표·주기의 마지막 관측값의 기간 시작일이다. 관측값이 없으면 nil이다.
	LastPeriod *time.Time
	// FirstStart는 오늘에서 차트 기간(indicator_display.chart_period)을 뺀 날이다. 화면 설정이 없으면 nil이다.
	FirstStart *time.Time
}

// Targets는 DB indicator의 지표를 모두 읽는다(화면 순서, 그다음 id 순서).
// today는 Asia/Seoul의 오늘이다. 연·월·일만 쓴다.
func (s *Store) Targets(ctx context.Context, today time.Time) ([]Target, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT i.id, i.name, i.source_code, coalesce(i.org_id, ''), i.stat_code, coalesce(i.item_code, ''),
		       i.classes, coalesce(i.value_field, ''), i.cycle,
		       (SELECT max(o.period_start) FROM observation o WHERE o.indicator_id = i.id AND o.cycle = i.cycle),
		       ($1::text::date - d.chart_period)::date
		FROM indicator i
		LEFT JOIN indicator_display d ON d.indicator_id = i.id
		ORDER BY d.sort_order NULLS LAST, i.id`,
		today.Format(time.DateOnly))
	if err != nil {
		return nil, wrap("read indicator", err)
	}
	list, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Target, error) {
		var t Target
		ind := &t.Indicator
		err := row.Scan(&ind.ID, &ind.Name, &ind.Source, &ind.OrgID, &ind.StatCode, &ind.ItemCode,
			&ind.Classes, &ind.ValueField, &ind.Cycle, &t.LastPeriod, &t.FirstStart)
		return t, err
	})
	if err != nil {
		return nil, wrap("read indicator", err)
	}
	return list, nil
}

// CollectTime은 app_setting의 수집 시각(collect_time, 예: 06:00, Asia/Seoul)을 읽는다(KEDP001-D019).
func (s *Store) CollectTime(ctx context.Context) (string, error) {
	var v string
	err := s.pool.QueryRow(ctx, `SELECT value FROM app_setting WHERE key = 'collect_time'`).Scan(&v)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", errors.New("read collect_time: app_setting has no collect_time")
	}
	if err != nil {
		return "", wrap("read collect_time", err)
	}
	return v, nil
}

// StartRun은 지표 하나의 수집 기록을 running으로 넣고 id를 돌려준다. Rust를 부르기 전에 만든다(b-proto/README.md 흐름 2).
func (s *Store) StartRun(ctx context.Context, sourceCode, indicatorID, trigger string) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `
		INSERT INTO collection_run (source_code, indicator_id, status, trigger)
		VALUES ($1, $2, 'running', $3)
		RETURNING id`, sourceCode, indicatorID, trigger).Scan(&id)
	if err != nil {
		return 0, wrap("insert collection_run", err)
	}
	return id, nil
}

// FinishRun은 running인 수집 기록을 결과로 고친다(b-proto/README.md 흐름 4). errorCode가 비어 있으면 NULL이다.
func (s *Store) FinishRun(ctx context.Context, id int64, status, errorCode string, rowCount int) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE collection_run
		SET finished_at = greatest(now(), started_at), status = $2, error_code = NULLIF($3, ''), row_count = $4
		WHERE id = $1 AND status = 'running'`, id, status, errorCode, rowCount)
	if err != nil {
		return wrap("update collection_run", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("update collection_run: run %d is not running", id)
	}
	return nil
}

// SeedResult는 Seed가 새로 넣은 행 수와 이미 있어 그대로 둔 행 수다.
type SeedResult struct {
	IndicatorAdded, IndicatorKept int
	DisplayAdded, DisplayKept     int
}

// Seed는 지표 정의와 화면 처음 값을 넣는다. 한 transaction이라 하나라도 실패하면 아무것도 넣지 않는다.
// 이미 있는 행(같은 id)은 고치지 않는다. 그래서 다시 실행해도 결과가 같고, 관리자 화면에서 바꾼 값을 덮어쓰지 않는다.
func (s *Store) Seed(ctx context.Context, entries []config.Entry) (SeedResult, error) {
	var r SeedResult
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		for _, e := range entries {
			ind, d := e.Indicator, e.Display
			classes := ind.Classes
			if classes == nil {
				classes = []string{} // NULL이 아니라 빈 배열이다(NOT NULL)
			}
			tag, err := tx.Exec(ctx, `
				INSERT INTO indicator (id, name, source_code, org_id, stat_code, item_code, classes, value_field, cycle)
				VALUES ($1, $2, $3, NULLIF($4, ''), $5, NULLIF($6, ''), $7, NULLIF($8, ''), $9)
				ON CONFLICT (id) DO NOTHING`,
				ind.ID, ind.Name, ind.Source, ind.OrgID, ind.StatCode, ind.ItemCode, classes, ind.ValueField, ind.Cycle)
			if err != nil {
				return wrap("insert indicator "+ind.ID, err)
			}
			count(tag, &r.IndicatorAdded, &r.IndicatorKept)

			tag, err = tx.Exec(ctx, `
				INSERT INTO indicator_display (indicator_id, is_visible, sort_order, chart_period, notice)
				VALUES ($1, $2, $3, $4::text::interval, NULLIF($5, ''))
				ON CONFLICT (indicator_id) DO NOTHING`,
				ind.ID, d.Visible, d.Order, d.ChartPeriod, d.Notice)
			if err != nil {
				return wrap("insert indicator_display "+ind.ID, err)
			}
			count(tag, &r.DisplayAdded, &r.DisplayKept)
		}
		return nil
	})
	if err != nil {
		return SeedResult{}, err
	}
	return r, nil
}

func count(tag pgconn.CommandTag, added, kept *int) {
	if tag.RowsAffected() == 1 {
		*added++
	} else {
		*kept++
	}
}
