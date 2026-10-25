package kosis

import (
	"strings"
	"testing"

	"github.com/schware/aidev-KEDP001/b-go-end/internal/config"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/fetch"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/source"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/source/sourcetest"
)

var sampleDir = sourcetest.SampleDir("kosis")

func readSample(t *testing.T, name string) []byte {
	t.Helper()
	return sourcetest.ReadSample(t, sampleDir, name)
}

func TestParseEmploymentRate(t *testing.T) {
	rows, err := Parse(readSample(t, "DT_1DA7001S-T90-0-M-202401-202609.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 33 {
		t.Fatalf("rows = %d, want 33", len(rows))
	}
	want := Row{
		OrgID:    "101",
		TblID:    "DT_1DA7001S",
		TblNm:    "성별 경제활동인구 총괄",
		ItmID:    "T90",
		ItmNm:    "고용률",
		C1:       "0",
		C1Nm:     "계",
		C1ObjNm:  "성별",
		UnitNm:   "%",
		PrdSe:    "M",
		PrdDe:    "202401",
		DT:       "61",
		LstChnDe: "2024-02-01",
	}
	if rows[0] != want {
		t.Errorf("first row = %+v, want %+v", rows[0], want)
	}
	if last := rows[len(rows)-1]; last.PrdDe != "202609" || last.DT != "63.5" {
		t.Errorf("last row = %+v", last)
	}
	for _, r := range rows {
		if r.ItmID != "T90" || r.C1 != "0" || r.UnitNm != "%" || r.PrdDe == "" || r.DT == "" {
			t.Errorf("unexpected row %+v", r)
		}
	}
}

func TestParseBirths(t *testing.T) {
	rows, err := Parse(readSample(t, "INH_1B8000F_01-T1-11-Y-2015-2025.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 11 {
		t.Fatalf("rows = %d, want 11", len(rows))
	}
	first, last := rows[0], rows[len(rows)-1]
	if first.PrdDe != "2015" || first.DT != "438420" || last.PrdDe != "2025" || last.DT != "254341" {
		t.Errorf("first = %+v, last = %+v", first, last)
	}
	for _, r := range rows {
		// 이 통계표는 UNIT_NM이 없고 단위가 분류 이름에 들어 있다.
		if r.ItmID != "T1" || r.C1 != "11" || r.C1Nm != "출생아수(명)" || r.UnitNm != "" || r.PrdSe != "A" {
			t.Errorf("unexpected row %+v", r)
		}
	}
}

func TestParseTotalPopulation(t *testing.T) {
	rows, err := Parse(readSample(t, "DT_1IN1502-T100-00-Y-2015-2025.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 11 {
		t.Fatalf("rows = %d, want 11", len(rows))
	}
	first, last := rows[0], rows[len(rows)-1]
	if first.PrdDe != "2015" || first.DT != "51069375" || last.PrdDe != "2025" || last.DT != "51817499" {
		t.Errorf("first = %+v, last = %+v", first, last)
	}
	for _, r := range rows {
		if r.ItmID != "T100" || r.C1 != "00" || r.C1Nm != "전국" || r.UnitNm != "명" || r.PrdSe != "A" {
			t.Errorf("unexpected row %+v", r)
		}
	}
}

func TestParseErrors(t *testing.T) {
	_, err := Parse(readSample(t, "error-30.json"))
	if !source.NoData(err) {
		t.Errorf("30: err = %v, want no data", err)
	}

	_, err = Parse(readSample(t, "error-11.json"))
	apiErr, ok := err.(*source.APIError)
	if !ok || apiErr.Source != "kosis" || apiErr.Code != "11" || source.NoData(err) {
		t.Errorf("11: err = %v", err)
	}

	if _, err := Parse([]byte(`{}`)); err == nil {
		t.Error("empty object: want error")
	}
	if _, err := Parse([]byte(`not json`)); err == nil {
		t.Error("not json: want error")
	}
}

func TestCheck(t *testing.T) {
	n, err := New("", nil).Check(readSample(t, "DT_1IN1502-T100-00-Y-2015-2025.json"))
	if err != nil || n != 11 {
		t.Errorf("Check = %d, %v", n, err)
	}
}

func TestFetchWithoutKey(t *testing.T) {
	_, err := New("", fetch.New(0)).Fetch(t.Context(), source.Request{})
	if err == nil || !strings.Contains(err.Error(), "KOSIS_API_KEY") {
		t.Errorf("err = %v", err)
	}
}

func TestDataURL(t *testing.T) {
	req := source.Request{
		Indicator: config.Indicator{StatCode: "DT_1DA7001S", ItemCode: "T90", Classes: []string{"0"}, Cycle: "M"},
		Start:     "202401",
		End:       "202609",
	}
	got := dataURL("TESTKEY", req)
	want := "https://kosis.kr/openapi/Param/statisticsParameterData.do?apiKey=TESTKEY&endPrdDe=202609&format=json&itmId=T90&jsonVD=Y&method=getList&objL1=0&orgId=101&prdSe=M&startPrdDe=202401&tblId=DT_1DA7001S"
	if got != want {
		t.Errorf("dataURL = %s", got)
	}
	if masked := fetch.Mask(got, "TESTKEY"); strings.Contains(masked, "TESTKEY") || !strings.Contains(masked, "apiKey=***") {
		t.Errorf("key not masked: %s", masked)
	}
	if got := req.SampleName(); got != "DT_1DA7001S-T90-0-M-202401-202609.json" {
		t.Errorf("SampleName = %s", got)
	}
}

// 샘플 파일에 키 자리 문자열이 그대로 남지 않았는지 확인한다.
func TestSamplesHaveNoKey(t *testing.T) {
	sourcetest.CheckMasked(t, sampleDir, "apiKey=")
}
