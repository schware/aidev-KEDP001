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
// 시험 DB에는 migration 0001~0004와 testdata/read.sql을 넣어 둔다(b-go-end/README.md).
func openTestDB(t *testing.T) *Store { return openDB(t, "KEDP_TEST_DB_NAME") }

// openDB는 변수 env가 가리키는 시험 DB에 kedp_go로 접속한다. 변수가 없으면 시험을 건너뛴다.
// 접속 정보(KEDP_DB_HOST, KEDP_DB_PORT, kedp_go)는 ../../../.env와 환경 변수에서 읽는다.
func openDB(t *testing.T, env string) *Store {
	t.Helper()
	name := os.Getenv(env)
	if name == "" {
		t.Skip(env + " is not set")
	}
	if name == "kedp" {
		t.Fatal(env + " must not be kedp")
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
	// 노출된 것만, 구분 순서(money 4, population 8)가 먼저이고 그다음 sort_order다.
	// t-hidden(노출 안 함), t-bare(화면 설정 없음)는 없다.
	want := []Card{
		{ID: "t-rate", Name: "금리", Cycle: "M", CategoryCode: "money", CategoryName: "금리·통화",
			Unit: ptr("%"), Decimals: ptr(2), Latest: ptr(pt("2026-09-01", 2.5)),
			Previous: ptr(pt("2026-08-01", 2.75)), Change: ptr(-0.25)},
		{ID: "t-pop", Name: "인구", Cycle: "A", CategoryCode: "population", CategoryName: "인구",
			Unit: ptr("명"), Latest: ptr(pt("2025-01-01", 51684564)),
			Previous: ptr(pt("2024-01-01", 51700000)), Change: ptr(-15436.0)},
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
		CategoryCode: "money", CategoryName: "금리·통화",
		Unit: ptr("%"), Decimals: ptr(2), ChartPeriod: ptr("1 year"),
		Latest: ptr(pt("2026-09-01", 2.5)),
		// 바로 앞 기간 값이다. 이전 주기(Q) 값은 보지 않는다.
		Previous: ptr(pt("2026-08-01", 2.75)), Change: ptr(-0.25),
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
		CategoryCode: "money", CategoryName: "금리·통화",
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

// 직전 값(Issue #70): 바로 앞 기간이 빈 경우, D 주기, 관측값이 하나뿐인 경우
func TestDetailPrevious(t *testing.T) {
	s := openTestDB(t)
	tests := []struct {
		id       string
		latest   Point
		previous *Point
		change   *float64
	}{
		// 2026-08 값이 없다. 더 앞(2026-07) 값으로 대신하지 않는다.
		{"t-gap", pt("2026-09-01", 2), nil, nil},
		// 월요일의 직전 값은 금요일이다. 차이는 numeric으로 빼서 0.3이다.
		{"t-day", pt("2026-10-26", 100.4), ptr(pt("2026-10-23", 100.1)), ptr(0.3)},
		{"t-one", pt("2026-04-01", 1.2), nil, nil},
	}
	for _, tt := range tests {
		got, err := s.Detail(context.Background(), tt.id, day("2026-10-30"), 10)
		if err != nil {
			t.Fatal(err)
		}
		if got.Latest == nil || *got.Latest != tt.latest ||
			!reflect.DeepEqual(got.Previous, tt.previous) || !reflect.DeepEqual(got.Change, tt.change) {
			t.Errorf("%s: latest %v, previous %v, change %v", tt.id, got.Latest, got.Previous, deref(got.Change))
		}
	}
}

func deref(p *float64) any {
	if p == nil {
		return nil
	}
	return *p
}
