package datagokr

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/schware/aidev-KEDP001/b-go-end/internal/config"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/fetch"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/source"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/source/sourcetest"
)

var sampleDir = sourcetest.SampleDir("data-go-kr")

func readSample(t *testing.T, name string) []byte {
	t.Helper()
	return sourcetest.ReadSample(t, sampleDir, name)
}

func TestParseETF(t *testing.T) {
	cases := []struct {
		file, code, name, index string
		firstClpr, lastClpr     string
	}{
		{"069500-D-20260901-20261022.json", "069500", "KODEX 200", "코스피 200", "86525", "107885"},
		{"122630-D-20260901-20261022.json", "122630", "KODEX 레버리지", "코스피 200", "66770", "107725"},
		{"114800-D-20260901-20261022.json", "114800", "KODEX 인버스", "코스피 200 선물지수", "1243", "1021"},
	}
	for _, c := range cases {
		rows, err := Parse(readSample(t, c.file))
		if err != nil {
			t.Fatalf("%s: %v", c.file, err)
		}
		if len(rows) != 34 {
			t.Fatalf("%s: rows = %d, want 34", c.file, len(rows))
		}
		// 응답은 기준일자 내림차순(최근 날짜가 먼저)이다.
		first, last := rows[0], rows[len(rows)-1]
		if first.BasDt != "20261022" || first.Clpr != c.firstClpr || last.BasDt != "20260901" || last.Clpr != c.lastClpr {
			t.Errorf("%s: first = %+v, last = %+v", c.file, first, last)
		}
		for i, r := range rows {
			if r.SrtnCd != c.code || r.ItmsNm != c.name || r.BssIdxNm != c.index || r.Clpr == "" || r.Nav == "" {
				t.Errorf("%s: unexpected row %+v", c.file, r)
			}
			if i > 0 && r.BasDt >= rows[i-1].BasDt {
				t.Errorf("%s: basDt not descending at %d", c.file, i)
			}
		}
	}
}

func TestParseFirstRow(t *testing.T) {
	rows, err := Parse(readSample(t, "069500-D-20260901-20261022.json"))
	if err != nil {
		t.Fatal(err)
	}
	want := Row{
		BasDt: "20261022", SrtnCd: "069500", IsinCd: "KR7069500007", ItmsNm: "KODEX 200",
		Clpr: "86525", Vs: "-3200", FltRt: "-3.57", Nav: "86756.18",
		Mkp: "88805", Hipr: "90120", Lopr: "85790", Trqu: "31012596", TrPrc: "2722482915116",
		MrktTotAmt: "21341391250000", StLstgCnt: "246650000",
		BssIdxNm: "코스피 200", BssIdxClpr: "858.02", NPptTotAmt: "21689044508700",
	}
	if rows[0] != want {
		t.Errorf("first row = %+v, want %+v", rows[0], want)
	}
}

func TestParseErrors(t *testing.T) {
	_, err := Parse(readSample(t, "nodata-totalCount-0.json"))
	if !source.NoData(err) {
		t.Errorf("totalCount 0: err = %v, want no data", err)
	}

	_, err = Parse(readSample(t, "error-30.json"))
	apiErr, ok := err.(*source.APIError)
	if !ok || apiErr.Source != "data-go-kr" || apiErr.Code != "30" || source.NoData(err) {
		t.Errorf("30: err = %v", err)
	}

	if _, err := Parse([]byte(`{"response":{"header":{"resultCode":"10","resultMsg":"INVALID_REQUEST_PARAMETER_ERROR"}}}`)); err == nil {
		t.Error("resultCode 10: want error")
	}
	if _, err := Parse([]byte(`{}`)); err == nil {
		t.Error("empty object: want error")
	}
}

func TestLastPage(t *testing.T) {
	s := New("", nil)
	cases := map[string]int{
		string(readSample(t, "114800-D-20260901-20261022.json")):                                     1,
		string(readSample(t, "nodata-totalCount-0.json")):                                            1,
		string(readSample(t, "error-30.json")):                                                       1,
		`{"response":{"header":{"resultCode":"00"},"body":{"totalCount":1001,"items":{"item":[]}}}}`: 2,
	}
	for body, want := range cases {
		if got := s.LastPage([]byte(body)); got != want {
			t.Errorf("LastPage(%.40s) = %d, want %d", body, got, want)
		}
	}
	// 1000행이 넘는 응답의 첫 페이지도 읽는다.
	if rows, err := Parse([]byte(`{"response":{"header":{"resultCode":"00"},"body":{"totalCount":2,"items":{"item":[{}]}}}}`)); err != nil || len(rows) != 1 {
		t.Errorf("first page: %d rows, %v", len(rows), err)
	}
}

func TestPeriod(t *testing.T) {
	s := New("", nil)
	day := time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)
	if got, err := s.Period("D", day); err != nil || got != "20261030" {
		t.Errorf("Period(D) = %q, %v", got, err)
	}
	if _, err := s.Period("M", day); err == nil {
		t.Error("M: want error")
	}
}

func TestFetchWithoutKey(t *testing.T) {
	_, err := New("", fetch.New(0)).Fetch(t.Context(), source.Request{Start: "20260901", End: "20261022"})
	if err == nil || !strings.Contains(err.Error(), "DATA_GO_KR_API_KEY") {
		t.Errorf("err = %v", err)
	}
}

func TestFetchBadPeriod(t *testing.T) {
	_, err := New("k", fetch.New(0)).Fetch(t.Context(), source.Request{Start: "2026-09-01", End: "20261022"})
	if err == nil || !strings.Contains(err.Error(), "YYYYMMDD") {
		t.Errorf("err = %v", err)
	}
}

// 인증키 오류는 HTTP 403과 함께 오므로, 본문을 읽어 APIError로 돌려주는지 본다.
// 테스트 서버로 받아 둔 샘플을 돌려준다. 실제 API는 부르지 않는다.
func TestReadStatusError(t *testing.T) {
	body := readSample(t, "error-30.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write(body)
	}))
	defer srv.Close()

	_, err := fetch.New(time.Second).Get(t.Context(), srv.URL)
	err = readStatusError(err)
	var apiErr *source.APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "30" || !strings.Contains(err.Error(), "status 403") {
		t.Errorf("err = %v", err)
	}

	plain := errors.New("x")
	if readStatusError(plain) != plain {
		t.Error("non-status error changed")
	}
}

func TestPriceURL(t *testing.T) {
	req := source.Request{
		Indicator: config.Indicator{StatCode: "069500", Cycle: "D"},
		Start:     "20260901",
		End:       "20261022",
	}
	end, err := endParam(req)
	if err != nil || end != "20261023" {
		t.Fatalf("endParam = %s, %v (endBasDt는 그 날을 포함하지 않아 하루를 더한다)", end, err)
	}
	got := priceURL("TEST+KEY/=", req, end)
	want := "https://apis.data.go.kr/1160100/GetSecuritiesProductInfoService_V2/getETFPriceInfo_V2?beginBasDt=20260901&endBasDt=20261023&likeSrtnCd=069500&numOfRows=1000&pageNo=1&resultType=json&serviceKey=TEST%2BKEY%2F%3D"
	if got != want {
		t.Errorf("priceURL = %s", got)
	}
	if masked := fetch.Mask(got, "TEST+KEY/="); strings.Contains(masked, "KEY") || !strings.Contains(masked, "serviceKey=***") {
		t.Errorf("key not masked: %s", masked)
	}
	if got := req.SampleName(); got != "069500-D-20260901-20261022.json" {
		t.Errorf("SampleName = %s", got)
	}
	req.Page = 2
	if got := priceURL("k", req, end); !strings.Contains(got, "&pageNo=2&") {
		t.Errorf("page 2: priceURL = %s", got)
	}
}

// Encoding 키(이미 인코딩된 형태)를 받아도 주소에서 두 번 인코딩되지 않는다.
func TestDecodeKey(t *testing.T) {
	for in, want := range map[string]string{
		"TEST%2BKEY%2F%3D": "TEST+KEY/=",
		"TEST+KEY/=":       "TEST+KEY/=",
		"plain":            "plain",
	} {
		if got := decodeKey(in); got != want {
			t.Errorf("decodeKey(%q) = %q, want %q", in, got, want)
		}
	}
}

// 샘플 파일에 키 자리 문자열이 그대로 남지 않았는지 확인한다.
func TestSamplesHaveNoKey(t *testing.T) {
	sourcetest.CheckMasked(t, sampleDir, "serviceKey=")
}
