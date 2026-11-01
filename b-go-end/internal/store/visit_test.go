package store

import (
	"errors"
	"reflect"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

// 방문자 수의 DB 일을 시험 DB로 확인한다. KEDP_VISIT_TEST_DB_NAME(예: kedp_visit_check)이 있을 때만 돈다.
// 시험 DB에는 migration 0001, 0002, 0003만 넣는다. 시험이 값을 바꾸므로 돌릴 때마다 DB를 새로 만든다.
// kedp_go로 접속하므로 0003의 권한도 함께 확인한다. 시험은 이 파일의 순서대로 돈다.

func openVisitDB(t *testing.T) *Store { return openDB(t, "KEDP_VISIT_TEST_DB_NAME") }

func TestVisitEmpty(t *testing.T) {
	s := openVisitDB(t)
	got, err := s.VisitSummary(bg, day("2026-10-30"))
	if err != nil || got != (VisitSummary{}) {
		t.Errorf("got %+v, %v", got, err)
	}
}

func TestAddVisit(t *testing.T) {
	s := openVisitDB(t)
	// 2026-10-30에 50명이 동시에 온다(한 문장이라 빠지지 않는다).
	var wg sync.WaitGroup
	errs := make(chan error, 50)
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- s.AddVisit(bg, day("2026-10-30"))
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, d := range []string{"2026-10-01", "2026-09-30", "2026-10-31", "2026-10-31"} {
		if err := s.AddVisit(bg, day(d)); err != nil {
			t.Fatal(err)
		}
	}
	got, err := s.VisitSummary(bg, day("2026-10-30"))
	if err != nil || got != (VisitSummary{Today: 50, Total: 54}) {
		t.Errorf("got %+v, %v", got, err)
	}
	// 오늘 기록이 없는 날
	got, err = s.VisitSummary(bg, day("2026-11-05"))
	if err != nil || got != (VisitSummary{Today: 0, Total: 54}) {
		t.Errorf("got %+v, %v", got, err)
	}
}

func TestSetVisitBase(t *testing.T) {
	s := openVisitDB(t)
	old, err := s.SetVisitBase(bg, 1000)
	if err != nil || old != "0" {
		t.Fatalf("old = %q, %v", old, err)
	}
	got, err := s.VisitSummary(bg, day("2026-10-31"))
	if err != nil || got != (VisitSummary{Today: 2, Total: 1054}) {
		t.Errorf("got %+v, %v", got, err)
	}
}

func TestVisitStats(t *testing.T) {
	s := openVisitDB(t)
	got, err := s.VisitStats(bg, day("2026-10-31"), 30)
	if err != nil {
		t.Fatal(err)
	}
	if got.Today != 2 || got.Total != 1054 || got.Base != 1000 || len(got.Days) != 30 {
		t.Fatalf("got %+v", got)
	}
	// 오래된 날부터 30일(10-02 ~ 10-31). 10-01, 09-30은 범위 밖이다. 빈 날은 0이다.
	for i, d := range got.Days {
		want := VisitDay{Day: day("2026-10-02").AddDate(0, 0, i)}
		switch want.Day.Format("2006-01-02") {
		case "2026-10-30":
			want.Count = 50
		case "2026-10-31":
			want.Count = 2
		}
		if !reflect.DeepEqual(d, want) {
			t.Errorf("%d: got %v %d, want %v %d", i, d.Day, d.Count, want.Day, want.Count)
		}
	}
	// 월이 바뀌는 범위
	got, err = s.VisitStats(bg, day("2026-10-01"), 3)
	if err != nil {
		t.Fatal(err)
	}
	want := []VisitDay{{day("2026-09-29"), 0}, {day("2026-09-30"), 1}, {day("2026-10-01"), 1}}
	if !reflect.DeepEqual(got.Days, want) {
		t.Errorf("got %+v", got.Days)
	}
}

func TestResetVisits(t *testing.T) {
	s := openVisitDB(t)
	got, err := s.ResetVisits(bg)
	if err != nil || got != (VisitReset{Days: 4, Count: 54, OldBase: "1000"}) {
		t.Fatalf("got %+v, %v", got, err)
	}
	stats, err := s.VisitStats(bg, day("2026-10-31"), 30)
	if err != nil || stats.Today != 0 || stats.Total != 0 || stats.Base != 0 {
		t.Errorf("after reset: %+v, %v", stats, err)
	}
	for _, d := range stats.Days {
		if d.Count != 0 {
			t.Errorf("after reset: %v = %d", d.Day, d.Count)
		}
	}
	// 초기화한 뒤에도 다시 센다.
	if err := s.AddVisit(bg, day("2026-10-31")); err != nil {
		t.Fatal(err)
	}
	if v, err := s.VisitSummary(bg, day("2026-10-31")); err != nil || v != (VisitSummary{1, 1}) {
		t.Errorf("got %+v, %v", v, err)
	}
}

// DB 검사: 숫자는 0 이상이다.
func TestVisitCountCheck(t *testing.T) {
	s := openVisitDB(t)
	_, err := s.pool.Exec(bg, `UPDATE visit_daily SET count = -1`)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23514" {
		t.Errorf("negative count: %v", err)
	}
}
