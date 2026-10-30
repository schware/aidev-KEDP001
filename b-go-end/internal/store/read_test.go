package store

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/schware/aidev-KEDP001/b-go-end/internal/config"
)

// openTestDB는 KEDP_TEST_DB_NAME 시험 DB에 kedp_go로 접속한다. 변수가 없으면 시험을 건너뛴다.
// 접속 정보(KEDP_DB_HOST, KEDP_DB_PORT, kedp_go)는 ../../../.env와 환경 변수에서 읽는다.
// 시험 DB에는 migration 0001, 0002와 testdata/read.sql을 넣어 둔다(b-go-end/README.md).
func openTestDB(t *testing.T) *Store {
	t.Helper()
	name := os.Getenv("KEDP_TEST_DB_NAME")
	if name == "" {
		t.Skip("KEDP_TEST_DB_NAME is not set")
	}
	if name == "kedp" {
		t.Fatal("KEDP_TEST_DB_NAME must not be kedp")
	}
	t.Setenv("KEDP_DB_NAME", name)
	db, err := config.LoadDB("../../../.env")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	s, err := Open(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	return s
}

func ptr[T any](v T) *T { return &v }

func day(s string) time.Time {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		panic(err)
	}
	return t
}

func pt(period string, value float64) Point { return Point{Period: day(period), Value: value} }

func TestCards(t *testing.T) {
	s := openTestDB(t)
	got, err := s.Cards(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// 노출된 것만, sort_order 순서다. t-hidden(노출 안 함), t-bare(화면 설정 없음)는 없다.
	want := []Card{
		{ID: "t-pop", Name: "인구", Cycle: "A", Unit: ptr("명"), Latest: ptr(pt("2025-01-01", 51684564))},
		{ID: "t-rate", Name: "금리", Cycle: "M", Unit: ptr("%"), Latest: ptr(pt("2026-09-01", 2.5))},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got  %+v\nwant %+v", got, want)
	}
}

func TestDetail(t *testing.T) {
	s := openTestDB(t)
	got, err := s.Detail(context.Background(), "t-rate", day("2026-10-30"), 3)
	if err != nil {
		t.Fatal(err)
	}
	if got.LastCollectedAt == nil || !got.LastCollectedAt.Equal(time.Date(2026, 10, 24, 21, 0, 3, 0, time.UTC)) {
		t.Errorf("LastCollectedAt = %v, want 2026-10-25 06:00:03 KST", got.LastCollectedAt)
	}
	got.LastCollectedAt = nil
	want := Detail{
		ID: "t-rate", Name: "금리", Cycle: "M", StatCode: "722Y001", SourceCode: "ecos", SourceName: "ECOS",
		Unit: ptr("%"), ChartPeriod: ptr("1 year"),
		Latest: ptr(pt("2026-09-01", 2.5)),
		// 2025-10-30 이후, 오래된 것부터. 이전 주기(Q) 값은 없다.
		Chart:   []Point{pt("2025-11-01", 3), pt("2026-08-01", 2.75), pt("2026-09-01", 2.5)},
		Recent:  []Point{pt("2026-09-01", 2.5), pt("2026-08-01", 2.75), pt("2025-11-01", 3)},
		Related: []Related{{"t-hidden", "숨김"}, {"t-pop", "인구"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got  %+v\nwant %+v", got, want)
	}
}

func TestDetailNotice(t *testing.T) {
	s := openTestDB(t)
	got, err := s.Detail(context.Background(), "t-pop", day("2026-10-30"), 10)
	if err != nil {
		t.Fatal(err)
	}
	// no_data 수집은 마지막 수집이 아니다.
	if got.Notice == nil || *got.Notice != "안내 문구" || got.LastCollectedAt != nil || len(got.Recent) != 2 {
		t.Errorf("got %+v", got)
	}
}

func TestDetailWithoutDisplay(t *testing.T) {
	s := openTestDB(t)
	got, err := s.Detail(context.Background(), "t-bare", day("2026-10-30"), 10)
	if err != nil {
		t.Fatal(err)
	}
	want := Detail{
		ID: "t-bare", Name: "설정 없음", Cycle: "Q", StatCode: "X", SourceCode: "ecos", SourceName: "ECOS",
		Chart: []Point{}, Recent: []Point{}, Related: []Related{},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got  %+v\nwant %+v", got, want)
	}
}

func TestDetailNotFound(t *testing.T) {
	s := openTestDB(t)
	if _, err := s.Detail(context.Background(), "none", day("2026-10-30"), 10); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}
