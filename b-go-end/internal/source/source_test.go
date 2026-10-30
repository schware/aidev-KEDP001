package source

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/schware/aidev-KEDP001/b-go-end/internal/config"
)

func TestSampleName(t *testing.T) {
	ecos := Request{Indicator: config.Indicator{StatCode: "722Y001", ItemCode: "0101000", Cycle: "M"}, Start: "202401", End: "202609"}
	if got := ecos.SampleName(); got != "722Y001-0101000-M-202401-202609.json" {
		t.Errorf("ecos = %s", got)
	}
	kosis := Request{Indicator: config.Indicator{StatCode: "DT_1IN1502", ItemCode: "T100", Classes: []string{"00"}, Cycle: "Y"}, Start: "2015", End: "2025"}
	if got := kosis.SampleName(); got != "DT_1IN1502-T100-00-Y-2015-2025.json" {
		t.Errorf("kosis = %s", got)
	}
	// 항목 코드가 없으면 그 칸이 빠진다.
	dataGoKr := Request{Indicator: config.Indicator{StatCode: "069500", Cycle: "D"}, Start: "20260901", End: "20261022"}
	if got := dataGoKr.SampleName(); got != "069500-D-20260901-20261022.json" {
		t.Errorf("data-go-kr = %s", got)
	}
}

func TestNoData(t *testing.T) {
	wrapped := fmt.Errorf("fetch: %w", &APIError{Source: "kosis", Code: "30", NoData: true})
	if !NoData(wrapped) {
		t.Error("wrapped no data")
	}
	if NoData(&APIError{Code: "11"}) || NoData(errors.New("x")) || NoData(nil) {
		t.Error("want false")
	}
}

func TestRequireKey(t *testing.T) {
	if err := RequireKey("kosis", "KOSIS_API_KEY", ""); err == nil {
		t.Error("want error")
	}
	if err := RequireKey("kosis", "KOSIS_API_KEY", "k"); err != nil {
		t.Error(err)
	}
}

func TestBaseGetWithoutKey(t *testing.T) {
	b := NewBase("kosis", "KOSIS_API_KEY", "", nil)
	_, err := b.Get(t.Context(), func(string) string { t.Error("buildURL called without key"); return "" })
	if err == nil || !strings.Contains(err.Error(), "KOSIS_API_KEY") || b.Name() != "kosis" {
		t.Errorf("err = %v", err)
	}
}

func TestPages(t *testing.T) {
	cases := []struct{ total, size, want int }{{0, 1000, 1}, {1, 1000, 1}, {1000, 1000, 1}, {1001, 1000, 2}, {2500, 1000, 3}, {5, 0, 1}}
	for _, c := range cases {
		if got := Pages(c.total, c.size); got != c.want {
			t.Errorf("Pages(%d, %d) = %d, want %d", c.total, c.size, got, c.want)
		}
	}
	if (Request{}).PageNumber() != 1 || (Request{Page: 3}).PageNumber() != 3 {
		t.Error("PageNumber")
	}
}

func TestQuarter(t *testing.T) {
	for m, want := range map[time.Month]int{time.January: 1, time.March: 1, time.April: 2, time.September: 3, time.October: 4, time.December: 4} {
		if got := Quarter(time.Date(2026, m, 15, 0, 0, 0, 0, time.UTC)); got != want {
			t.Errorf("Quarter(%s) = %d, want %d", m, got, want)
		}
	}
}
