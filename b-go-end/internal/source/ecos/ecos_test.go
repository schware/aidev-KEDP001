package ecos

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/schware/aidev-KEDP001/b-go-end/internal/config"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/fetch"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/source"
)

// 받아 둔 샘플만 읽는다. 실제 API는 부르지 않는다(KEDP001-D021).
const sampleDir = "../../../../samples/ecos"

func readSample(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(sampleDir, name))
	if err != nil {
		t.Fatal(err)
	}
	return b
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
	if !NoData(err) {
		t.Errorf("INFO-200: err = %v, want no data", err)
	}

	_, err = Parse(readSample(t, "error-INFO-100.json"))
	apiErr, ok := err.(*APIError)
	if !ok || apiErr.Code != "INFO-100" || NoData(err) {
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

// 샘플 파일에 키 자리 문자열이 그대로 남지 않았는지 확인한다.
func TestSamplesHaveNoKeyPath(t *testing.T) {
	entries, err := os.ReadDir(sampleDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		b := readSample(t, e.Name())
		if strings.Contains(string(b), "/StatisticSearch/") && !strings.Contains(string(b), "/StatisticSearch/***/") {
			t.Errorf("%s: request path without masked key", e.Name())
		}
	}
}
