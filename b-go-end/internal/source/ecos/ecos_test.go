package ecos

import (
	"strings"
	"testing"
	"time"

	"github.com/schware/aidev-KEDP001/b-go-end/internal/config"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/fetch"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/source"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/source/sourcetest"
)

var sampleDir = sourcetest.SampleDir("ecos")

func readSample(t *testing.T, name string) []byte {
	t.Helper()
	return sourcetest.ReadSample(t, sampleDir, name)
}

func TestParseBaseRate(t *testing.T) {
	rows, err := Parse(readSample(t, "722Y001-0101000-M-202401-202609.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 33 {
		t.Fatalf("rows = %d, want 33", len(rows))
	}
	first := rows[0]
	want := Row{
		StatCode:  "722Y001",
		StatName:  "1.3.1. 한국은행 기준금리 및 여수신금리",
		ItemCode1: "0101000",
		ItemName1: "한국은행 기준금리",
		UnitName:  "연%",
		Time:      "202401",
		DataValue: "3.5",
	}
	if first != want {
		t.Errorf("first row = %+v, want %+v", first, want)
	}
	if last := rows[len(rows)-1]; last.Time != "202609" || last.DataValue != "3" {
		t.Errorf("last row = %+v", last)
	}
	for _, r := range rows {
		if r.ItemCode1 != "0101000" || r.UnitName != "연%" || r.Time == "" || r.DataValue == "" {
			t.Errorf("unexpected row %+v", r)
		}
	}
}

func TestParseErrors(t *testing.T) {
	_, err := Parse(readSample(t, "error-INFO-200.json"))
	if !source.NoData(err) {
		t.Errorf("INFO-200: err = %v, want no data", err)
	}

	_, err = Parse(readSample(t, "error-INFO-100.json"))
	apiErr, ok := err.(*source.APIError)
	if !ok || apiErr.Source != "ecos" || apiErr.Code != "INFO-100" || source.NoData(err) {
		t.Errorf("INFO-100: err = %v", err)
	}

	if _, err := Parse([]byte(`{}`)); err == nil {
		t.Error("empty object: want error")
	}
}

func TestSearchURL(t *testing.T) {
	req := source.Request{
		Indicator: config.Indicator{StatCode: "722Y001", ItemCode: "0101000", Cycle: "M"},
		Start:     "202401",
		End:       "202609",
	}
	got := searchURL("TESTKEY", req)
	want := "https://ecos.bok.or.kr/api/StatisticSearch/TESTKEY/json/kr/1/1000/722Y001/M/202401/202609/0101000"
	if got != want {
		t.Errorf("searchURL = %s", got)
	}
	if masked := fetch.Mask(got, "TESTKEY"); strings.Contains(masked, "TESTKEY") {
		t.Errorf("key not masked: %s", masked)
	}
}

func TestSearchURLWithClasses(t *testing.T) {
	req := source.Request{
		Indicator: config.Indicator{StatCode: "901Y009", ItemCode: "0", Classes: []string{"A", "B"}, Cycle: "Q"},
		Start:     "2024Q1",
		End:       "2024Q4",
	}
	want := "https://ecos.bok.or.kr/api/StatisticSearch/TESTKEY/json/kr/1/1000/901Y009/Q/2024Q1/2024Q4/0/A/B"
	if got := searchURL("TESTKEY", req); got != want {
		t.Errorf("searchURL = %s", got)
	}
}

func TestItemListURL(t *testing.T) {
	want := "https://ecos.bok.or.kr/api/StatisticItemList/TESTKEY/json/kr/1/1000/722Y001"
	if got := itemListURL("TESTKEY", "722Y001"); got != want {
		t.Errorf("itemListURL = %s", got)
	}
	_, err := New("", fetch.New(0)).FetchItems(t.Context(), "722Y001")
	if err == nil || !strings.Contains(err.Error(), "ECOS_API_KEY") {
		t.Errorf("FetchItems without key: err = %v", err)
	}
}

func TestSearchURLPage(t *testing.T) {
	req := source.Request{
		Indicator: config.Indicator{StatCode: "722Y001", ItemCode: "0101000", Cycle: "D"},
		Start:     "20200101",
		End:       "20261030",
		Page:      3,
	}
	want := "https://ecos.bok.or.kr/api/StatisticSearch/TESTKEY/json/kr/2001/3000/722Y001/D/20200101/20261030/0101000"
	if got := searchURL("TESTKEY", req); got != want {
		t.Errorf("searchURL = %s", got)
	}
}

func TestLastPage(t *testing.T) {
	s := New("", nil)
	cases := map[string]int{
		string(readSample(t, "722Y001-0101000-M-202401-202609.json")): 1,
		string(readSample(t, "error-INFO-200.json")):                  1,
		`{"StatisticSearch":{"list_total_count":1000,"row":[]}}`:      1,
		`{"StatisticSearch":{"list_total_count":2001,"row":[]}}`:      3,
		`not json`: 1,
	}
	for body, want := range cases {
		if got := s.LastPage([]byte(body)); got != want {
			t.Errorf("LastPage(%.40s) = %d, want %d", body, got, want)
		}
	}
	// 1000행이 넘는 응답의 첫 페이지도 읽는다.
	if rows, err := Parse([]byte(`{"StatisticSearch":{"list_total_count":2,"row":[{}]}}`)); err != nil || len(rows) != 1 {
		t.Errorf("first page: %d rows, %v", len(rows), err)
	}
}

func TestPeriod(t *testing.T) {
	s := New("", nil)
	day := time.Date(2026, 8, 7, 0, 0, 0, 0, time.UTC)
	for cycle, want := range map[string]string{"D": "20260807", "M": "202608", "Q": "2026Q3", "A": "2026"} {
		if got, err := s.Period(cycle, day); err != nil || got != want {
			t.Errorf("Period(%s) = %q, %v, want %q", cycle, got, err, want)
		}
	}
	if _, err := s.Period("Y", day); err == nil {
		t.Error("Y: want error")
	}
}

func TestFetchWithoutKey(t *testing.T) {
	_, err := New("", fetch.New(0)).Fetch(t.Context(), source.Request{})
	if err == nil || !strings.Contains(err.Error(), "ECOS_API_KEY") {
		t.Errorf("err = %v", err)
	}
}

// 샘플 파일에 키 자리 문자열이 그대로 남지 않았는지 확인한다.
func TestSamplesHaveNoKeyPath(t *testing.T) {
	sourcetest.CheckMasked(t, sampleDir, "/StatisticSearch/")
}
