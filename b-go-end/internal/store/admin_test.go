package store

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/schware/aidev-KEDP001/b-go-end/internal/config"
)

// 관리자 API의 DB 일을 시험 DB로 확인한다. KEDP_ADMIN_TEST_DB_NAME(예: kedp_admin_check)이 있을 때만 돈다.
// 시험 DB에는 migration 0001~0004와 testdata/admin.sql을 넣어 둔다. 시험이 값을 바꾸므로 돌릴 때마다 DB를 새로 만든다.
// 시험은 이 파일의 순서대로 돈다.

func openAdminDB(t *testing.T) *Store { return openDB(t, "KEDP_ADMIN_TEST_DB_NAME") }

var bg = context.Background()

func TestAdminIndicators(t *testing.T) {
	s := openAdminDB(t)
	got, err := s.AdminIndicators(bg)
	if err != nil {
		t.Fatal(err)
	}
	want := []AdminIndicator{
		{ID: "a-rate", Name: "금리", Category: "money", Source: "ecos", StatCode: "722Y001", Cycle: "M", ItemCode: ptr("0101000"),
			Classes: []string{}, Unit: ptr("%"), SourceUnit: ptr("연%"), HasDisplay: true, Visible: true, Order: ptr(1),
			ChartPeriod: ptr("10 years"), Decimals: ptr(1), Related: []string{"a-pop", "a-hidden"}},
		{ID: "a-pop", Name: "인구", Category: "population", Source: "kosis", StatCode: "DT_1", Cycle: "A", ItemCode: ptr("T1"), OrgID: ptr("101"),
			Classes: []string{"00"}, SourceUnit: ptr("명"), HasDisplay: true, Visible: true, Order: ptr(2),
			ChartPeriod: ptr("20 years"), Notice: ptr("안내 문구"), Related: []string{}},
		{ID: "a-hidden", Name: "숨김", Category: "market", Source: "data-go-kr", StatCode: "069500", Cycle: "D", ValueField: ptr("clpr"),
			Classes: []string{}, HasDisplay: true, Order: ptr(3), ChartPeriod: ptr("18 months"), Related: []string{}},
		// 화면 설정이 없는 지표도 나온다(맨 뒤).
		{ID: "a-bare", Name: "설정 없음", Category: "growth", Source: "ecos", StatCode: "X", Cycle: "Q", ItemCode: ptr("Y"),
			Classes: []string{}, Related: []string{}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got  %+v\nwant %+v", got, want)
	}
	if _, err := s.AdminIndicator(bg, "none"); !errors.Is(err, ErrNotFound) {
		t.Errorf("none: %v", err)
	}
}

func TestAddIndicator(t *testing.T) {
	s := openAdminDB(t)
	n := NewIndicator{ID: "a-kospi", Name: "코스피", Category: "market", Source: "kosis", StatCode: "DT_K", ItemCode: "T2", OrgID: "101",
		Classes: []string{"0", "1"}, Cycle: "M", Unit: "pt", Visible: true, ChartPeriod: "5 years", Decimals: ptr(2),
		Notice: "", Related: []string{"a-rate", "a-pop"}}
	if err := s.AddIndicator(bg, n); err != nil {
		t.Fatal(err)
	}
	got, err := s.AdminIndicator(bg, "a-kospi")
	if err != nil {
		t.Fatal(err)
	}
	want := AdminIndicator{ID: "a-kospi", Name: "코스피", Category: "market", Source: "kosis", StatCode: "DT_K", Cycle: "M",
		ItemCode: ptr("T2"), OrgID: ptr("101"), Classes: []string{"0", "1"}, Unit: ptr("pt"),
		HasDisplay: true, Visible: true, Order: ptr(4), ChartPeriod: ptr("5 years"), Decimals: ptr(2),
		Related: []string{"a-rate", "a-pop"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got  %+v\nwant %+v", got, want)
	}

	// 같은 id
	if err := s.AddIndicator(bg, n); !errors.Is(err, ErrExists) {
		t.Errorf("duplicate: %v", err)
	}
	// 없는 관련 지표면 아무것도 넣지 않는다.
	n2 := NewIndicator{ID: "a-new", Name: "새", Category: "market", Source: "data-go-kr", StatCode: "1", ValueField: "clpr", Cycle: "D",
		ChartPeriod: "30 days", Related: []string{"none"}}
	var fe *FieldError
	if err := s.AddIndicator(bg, n2); !errors.As(err, &fe) || fe.Field != "related" {
		t.Errorf("unknown related: %v", err)
	}
	if _, err := s.AdminIndicator(bg, "a-new"); !errors.Is(err, ErrNotFound) {
		t.Errorf("a-new was inserted: %v", err)
	}
	// 없는 구분도 아무것도 넣지 않는다.
	n2.Category, n2.Related = "none", nil
	if err := s.AddIndicator(bg, n2); !errors.As(err, &fe) || fe.Field != "category" || fe.Reason != `category "none" does not exist` {
		t.Errorf("unknown category: %v", err)
	}
	if _, err := s.AdminIndicator(bg, "a-new"); !errors.Is(err, ErrNotFound) {
		t.Errorf("a-new was inserted: %v", err)
	}
}

func TestUpdateIndicator(t *testing.T) {
	s := openAdminDB(t)
	e := Edit{Name: "기준금리", Category: "price", Cycle: "Q", Unit: "", ChartPeriod: "3 years", Decimals: nil, Notice: "주의",
		Related: []string{"a-hidden"}}
	if err := s.UpdateIndicator(bg, "a-rate", e); err != nil {
		t.Fatal(err)
	}
	got, err := s.AdminIndicator(bg, "a-rate")
	if err != nil {
		t.Fatal(err)
	}
	// id, 제공처, 코드, 노출, 순서는 그대로다. 빈 단위는 NULL이다.
	want := AdminIndicator{ID: "a-rate", Name: "기준금리", Category: "price", Source: "ecos", StatCode: "722Y001", Cycle: "Q",
		ItemCode: ptr("0101000"), Classes: []string{}, SourceUnit: ptr("연%"), HasDisplay: true, Visible: true,
		Order: ptr(1), ChartPeriod: ptr("3 years"), Notice: ptr("주의"), Related: []string{"a-hidden"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got  %+v\nwant %+v", got, want)
	}
	// number_format의 다른 칸은 남는다.
	var nf string
	if err := s.pool.QueryRow(bg, `SELECT number_format::text FROM indicator_display WHERE indicator_id = 'a-rate'`).Scan(&nf); err != nil {
		t.Fatal(err)
	}
	if nf != `{"keep": true}` {
		t.Errorf("number_format = %s", nf)
	}

	if err := s.UpdateIndicator(bg, "none", e); !errors.Is(err, ErrNotFound) {
		t.Errorf("none: %v", err)
	}
	var fe *FieldError
	if err := s.UpdateIndicator(bg, "a-rate", Edit{Name: "x", Category: "money", Cycle: "M", ChartPeriod: "1 years",
		Related: []string{"none"}}); !errors.As(err, &fe) {
		t.Errorf("unknown related: %v", err)
	}
	if err := s.UpdateIndicator(bg, "a-rate", Edit{Name: "x", Category: "none", Cycle: "M", ChartPeriod: "1 years",
		Related: []string{}}); !errors.As(err, &fe) || fe.Field != "category" {
		t.Errorf("unknown category: %v", err)
	}
	if got, _ := s.AdminIndicator(bg, "a-rate"); got.Name != "기준금리" || got.Category != "price" || len(got.Related) != 1 {
		t.Errorf("failed update changed the indicator: %+v", got)
	}
}

// 주기를 바꾸면 이전 주기 값은 남고, 수집은 새 주기 값이 없으므로 차트 기간만큼 처음부터 받는다(KEDP001-D022 11, 12).
func TestCycleChangeCollectsFromStart(t *testing.T) {
	s := openAdminDB(t)
	var n int
	if err := s.pool.QueryRow(bg, `SELECT count(*) FROM observation WHERE indicator_id = 'a-rate' AND cycle = 'M'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("old cycle observations = %d, want 2", n)
	}
	targets, err := s.Targets(bg, day("2026-10-30"))
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range targets {
		if x.Indicator.ID != "a-rate" {
			continue
		}
		if x.Indicator.Cycle != "Q" || x.LastPeriod != nil || x.FirstStart == nil || !x.FirstStart.Equal(day("2023-10-30")) {
			t.Errorf("target = %+v, LastPeriod %v, FirstStart %v", x.Indicator, x.LastPeriod, x.FirstStart)
		}
		return
	}
	t.Error("a-rate is not a target")
}

func TestUpdateIndicatorCreatesDisplay(t *testing.T) {
	s := openAdminDB(t)
	if err := s.UpdateIndicator(bg, "a-bare", Edit{Name: "설정 없음", Category: "growth", Cycle: "Q", ChartPeriod: "2 years",
		Decimals: ptr(0), Related: []string{}}); err != nil {
		t.Fatal(err)
	}
	got, err := s.AdminIndicator(bg, "a-bare")
	if err != nil {
		t.Fatal(err)
	}
	// 노출 안 함, 맨 뒤 순서(a-kospi 4 다음)
	if !got.HasDisplay || got.Visible || got.Order == nil || *got.Order != 5 ||
		*got.ChartPeriod != "2 years" || got.Decimals == nil || *got.Decimals != 0 {
		t.Errorf("got %+v", got)
	}
}

func TestSaveOrder(t *testing.T) {
	s := openAdminDB(t)
	list, err := s.AdminIndicators(bg)
	if err != nil {
		t.Fatal(err)
	}
	// 지금 목록을 거꾸로, 노출을 뒤집어 저장한다.
	var items []OrderItem
	for i := range list {
		a := list[len(list)-1-i]
		items = append(items, OrderItem{ID: a.ID, Visible: !a.Visible, Order: (i + 1) * 10})
	}
	if err := s.SaveOrder(bg, items); err != nil {
		t.Fatal(err)
	}
	got, err := s.AdminIndicators(bg)
	if err != nil {
		t.Fatal(err)
	}
	for i, a := range got {
		if a.ID != items[i].ID || a.Visible != items[i].Visible || *a.Order != items[i].Order {
			t.Errorf("%d: got %s %t %d, want %+v", i, a.ID, a.Visible, *a.Order, items[i])
		}
	}

	// 빠진 지표, 없는 지표가 있으면 아무것도 바꾸지 않는다.
	var fe *FieldError
	changed := []OrderItem{{ID: items[0].ID, Visible: true, Order: 999}}
	if err := s.SaveOrder(bg, changed); !errors.As(err, &fe) {
		t.Errorf("missing: %v", err)
	}
	if err := s.SaveOrder(bg, append(items, OrderItem{ID: "none"})); !errors.As(err, &fe) {
		t.Errorf("unknown: %v", err)
	}
	if a, _ := s.AdminIndicator(bg, items[0].ID); *a.Order == 999 {
		t.Error("failed save changed the order")
	}
}

func TestSetCollectTime(t *testing.T) {
	s := openAdminDB(t)
	old, err := s.SetCollectTime(bg, "07:30")
	if err != nil || old != "06:00" {
		t.Fatalf("old = %q, %v", old, err)
	}
	if v, err := s.CollectTime(bg); err != nil || v != "07:30" {
		t.Errorf("collect_time = %q, %v", v, err)
	}
}

func TestSourceRuns(t *testing.T) {
	s := openAdminDB(t)
	got, err := s.SourceRuns(bg)
	if err != nil {
		t.Fatal(err)
	}
	want := []SourceRun{
		{Code: "data-go-kr", Name: "공공데이터포털"},
		{Code: "ecos", Name: "ECOS", Status: "error", FinishedAt: ptr(time.Date(2026, 10, 24, 21, 0, 1, 0, time.UTC))},
		{Code: "kosis", Name: "KOSIS", Status: "no_data", FinishedAt: ptr(time.Date(2026, 10, 24, 21, 0, 5, 0, time.UTC))},
	}
	for i := range got {
		if got[i].FinishedAt != nil {
			got[i].FinishedAt = ptr(got[i].FinishedAt.UTC())
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got  %+v\nwant %+v", got, want)
	}
}

func TestFormatPeriod(t *testing.T) {
	for _, c := range []struct {
		in   pgtype.Interval
		want string
	}{
		{pgtype.Interval{Months: 120}, "10 years"},
		{pgtype.Interval{Months: 12}, "1 years"},
		{pgtype.Interval{Months: 18}, "18 months"},
		{pgtype.Interval{Days: 30}, "30 days"},
		{pgtype.Interval{Months: 1, Days: 2}, "1 mons 2 days 0s"},
	} {
		if got := FormatPeriod(c.in); got != c.want {
			t.Errorf("%+v: got %s, want %s", c.in, got, c.want)
		}
	}
}

func TestCategories(t *testing.T) {
	s := openAdminDB(t)
	got, err := s.Categories(bg)
	if err != nil {
		t.Fatal(err)
	}
	// 0004_category.up.sql의 9개, sort_order 순서다.
	want := []Category{
		{"growth", "성장·경기", 1}, {"price", "물가", 2}, {"labor", "고용·소득", 3},
		{"money", "금리·통화", 4}, {"credit", "부채·금융안정", 5}, {"external", "대외경제", 6},
		{"housing", "주택", 7}, {"population", "인구", 8}, {"market", "금융시장", 9},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got  %+v\nwant %+v", got, want)
	}
}

func TestSeedCategory(t *testing.T) {
	s := openAdminDB(t)
	entry := func(id, category string) config.Entry {
		return config.Entry{
			Indicator: config.Indicator{ID: id, Name: "시드", Source: "ecos", StatCode: "S", ItemCode: "I", Cycle: "M", Category: category},
			Display:   config.Display{ChartPeriod: "1 year"},
		}
	}
	if _, err := s.Seed(bg, []config.Entry{entry("a-seed", "labor")}); err != nil {
		t.Fatal(err)
	}
	if got, err := s.AdminIndicator(bg, "a-seed"); err != nil || got.Category != "labor" {
		t.Errorf("a-seed = %+v, %v", got, err)
	}
	// 없는 구분이면 하나도 넣지 않는다(한 transaction).
	if _, err := s.Seed(bg, []config.Entry{entry("a-seed2", "labor"), entry("a-seed3", "none")}); err == nil {
		t.Error("unknown category: no error")
	}
	if _, err := s.AdminIndicator(bg, "a-seed2"); !errors.Is(err, ErrNotFound) {
		t.Errorf("a-seed2 was inserted: %v", err)
	}
}
