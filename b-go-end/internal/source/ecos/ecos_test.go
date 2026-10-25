package ecos

import (
	"strings"
	"testing"

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
	if _, err := Parse([]byte(`{"StatisticSearch":{"list_total_count":2,"row":[{}]}}`)); err == nil {
		t.Error("partial page: want error")
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

func TestCheck(t *testing.T) {
	n, err := New("", nil).Check(readSample(t, "722Y001-0101000-M-202401-202609.json"))
	if err != nil || n != 33 {
		t.Errorf("Check = %d, %v", n, err)
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
