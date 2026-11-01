package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// 방문자 수(Issue #48, b-sql-db/migrations/0003_visits.up.sql). DB에는 날짜별 숫자와 전체 시작값만 둔다.
// 날짜(day)는 Asia/Seoul의 날짜다. 연·월·일만 쓴다.

// visitBaseKey는 전체 방문자 수의 시작값이 있는 app_setting key다. 전체 = 시작값 + visit_daily 합
const visitBaseKey = "visit_total_base"

// VisitSummary는 오늘 방문자 수와 전체 방문자 수다.
type VisitSummary struct {
	Today, Total int64
}

// VisitDay는 하루의 방문자 수다.
type VisitDay struct {
	Day   time.Time
	Count int64
}

// VisitStats는 관리자 화면의 방문자 수다. Days는 오래된 날부터이고, 기록이 없는 날은 0이다.
type VisitStats struct {
	Today, Total, Base int64
	Days               []VisitDay
}

// VisitReset은 초기화로 지운 일별 기록(날 수, 방문자 수 합)과 이전 시작값이다.
type VisitReset struct {
	Days, Count int64
	OldBase     string
}

func dateText(day time.Time) string { return day.Format(time.DateOnly) }

// AddVisit은 day의 방문자 수를 1 늘린다. 한 문장이라 동시에 불러도 빠지지 않는다.
func (s *Store) AddVisit(ctx context.Context, day time.Time) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO visit_daily (day, count) VALUES ($1::text::date, 1)
		ON CONFLICT (day) DO UPDATE SET count = visit_daily.count + 1`, dateText(day))
	if err != nil {
		return wrap("update visit_daily", err)
	}
	return nil
}

// summarySQL은 오늘($1) 방문자 수와 전체 방문자 수다. 시작값이 없으면 0으로 본다.
const summarySQL = `
	SELECT coalesce((SELECT count FROM visit_daily WHERE day = $1::text::date), 0)::bigint,
	       coalesce((SELECT value::bigint FROM app_setting WHERE key = '` + visitBaseKey + `'), 0),
	       coalesce((SELECT sum(count) FROM visit_daily), 0)::bigint`

// VisitSummary는 today의 방문자 수와 전체 방문자 수를 읽는다.
func (s *Store) VisitSummary(ctx context.Context, today time.Time) (VisitSummary, error) {
	var v VisitSummary
	var base, sum int64
	if err := s.pool.QueryRow(ctx, summarySQL, dateText(today)).Scan(&v.Today, &base, &sum); err != nil {
		return VisitSummary{}, wrap("read visits", err)
	}
	v.Total = base + sum
	return v, nil
}

// VisitStats는 today의 방문자 수, 전체, 시작값과 today까지 days일의 일별 방문자 수를 읽는다.
// 한 읽기 transaction이라 숫자가 서로 맞는다.
func (s *Store) VisitStats(ctx context.Context, today time.Time, days int) (VisitStats, error) {
	var v VisitStats
	err := pgx.BeginTxFunc(ctx, s.pool, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly},
		func(tx pgx.Tx) error {
			var sum int64
			if err := tx.QueryRow(ctx, summarySQL, dateText(today)).Scan(&v.Today, &v.Base, &sum); err != nil {
				return err
			}
			v.Total = v.Base + sum
			rows, err := tx.Query(ctx, `
				SELECT d.day, coalesce(v.count, 0)::bigint
				FROM (SELECT $1::text::date - g AS day FROM generate_series($2::int - 1, 0, -1) g) d
				LEFT JOIN visit_daily v ON v.day = d.day
				ORDER BY d.day`, dateText(today), days)
			if err != nil {
				return err
			}
			v.Days, err = pgx.CollectRows(rows, pgx.RowToStructByPos[VisitDay])
			return err
		})
	if err != nil {
		return VisitStats{}, wrap("read visits", err)
	}
	return v, nil
}

// SetVisitBase는 전체 방문자 수의 시작값을 base로 바꾸고 이전 값을 돌려준다. 범위는 admin이 검사한다.
func (s *Store) SetVisitBase(ctx context.Context, base int64) (string, error) {
	var old string
	err := s.pool.QueryRow(ctx, `
		WITH old AS (SELECT value FROM app_setting WHERE key = '`+visitBaseKey+`')
		INSERT INTO app_setting (key, value) VALUES ('`+visitBaseKey+`', $1::bigint::text)
		ON CONFLICT (key) DO UPDATE SET value = excluded.value, updated_at = now()
		RETURNING coalesce((SELECT value FROM old), '')`, base).Scan(&old)
	if err != nil {
		return "", wrap("update "+visitBaseKey, err)
	}
	return old, nil
}

// ResetVisits는 일별 기록을 모두 지우고 시작값을 0으로 한다. 한 transaction이다.
func (s *Store) ResetVisits(ctx context.Context) (VisitReset, error) {
	var r VisitReset
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `
			WITH d AS (DELETE FROM visit_daily RETURNING count)
			SELECT count(*), coalesce(sum(count), 0)::bigint FROM d`).Scan(&r.Days, &r.Count); err != nil {
			return fmt.Errorf("delete visit_daily: %w", err)
		}
		if err := tx.QueryRow(ctx, `
			WITH old AS (SELECT value FROM app_setting WHERE key = '`+visitBaseKey+`')
			INSERT INTO app_setting (key, value) VALUES ('`+visitBaseKey+`', '0')
			ON CONFLICT (key) DO UPDATE SET value = excluded.value, updated_at = now()
			RETURNING coalesce((SELECT value FROM old), '')`).Scan(&r.OldBase); err != nil {
			return fmt.Errorf("update %s: %w", visitBaseKey, err)
		}
		return nil
	})
	if err != nil {
		return VisitReset{}, wrap("reset visits", err)
	}
	return r, nil
}
