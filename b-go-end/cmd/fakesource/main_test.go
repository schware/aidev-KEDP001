package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/schware/aidev-KEDP001/b-go-end/internal/config"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/fetch"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/source"
	datagokr "github.com/schware/aidev-KEDP001/b-go-end/internal/source/data_go_kr"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/source/ecos"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/source/kosis"
)

var samples = filepath.Join("..", "..", "..", "samples")

// fakeSources는 수집기와 같은 제공처 코드로 요청 주소를 만들어 fakesource(시험 서버)에 보낸다.
func fakeSources(t *testing.T) map[string]source.Source {
	t.Helper()
	srv := httptest.NewServer(handler(samples))
	t.Cleanup(srv.Close)
	client := fetch.New(5*time.Second, "TESTKEY")
	if err := client.SetHost(srv.URL); err != nil {
		t.Fatal(err)
	}
	return map[string]source.Source{
		"ecos":       ecos.New("TESTKEY", client),
		"kosis":      kosis.New("TESTKEY", client),
		"data-go-kr": datagokr.New("TESTKEY", client),
	}
}

// fetchSample은 지표 ind의 요청을 보내고, 받은 본문이 samples/의 file과 같은지 본다.
func fetchSample(t *testing.T, srcs map[string]source.Source, ind config.Indicator, start, end, file string) {
	t.Helper()
	body, err := srcs[ind.Source].Fetch(context.Background(), source.Request{Indicator: ind, Start: start, End: end})
	if err != nil {
		t.Errorf("%s %s %v: %v", ind.StatCode, ind.ItemCode, ind.Classes, err)
		return
	}
	want, err := os.ReadFile(filepath.Join(samples, file))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != string(want) {
		t.Errorf("%s %s %v: body is not %s", ind.StatCode, ind.ItemCode, ind.Classes, file)
	}
}

// 지금 지표(config/indicators.yaml)의 요청은 그 지표의 샘플을 받는다.
func TestCurrentIndicators(t *testing.T) {
	srcs := fakeSources(t)
	entries, err := config.LoadIndicators("../../config/indicators.yaml")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]struct{ start, end, file string }{
		"ecos-base-rate":            {"201610", "202610", "ecos/722Y001-0101000-M-202401-202609.json"},
		"kosis-employment-rate":     {"201610", "202610", "kosis/DT_1DA7001S-T90-0-M-202401-202609.json"},
		"kosis-total-population":    {"2006", "2026", "kosis/DT_1IN1502-T100-00-Y-2015-2025.json"},
		"kosis-births":              {"2006", "2026", "kosis/INH_1B8000F_01-T1-11-Y-2015-2025.json"},
		"data-go-kr-kodex-200":      {"20251030", "20261030", "data-go-kr/069500-D-20260901-20261022.json"},
		"data-go-kr-kodex-leverage": {"20251030", "20261030", "data-go-kr/122630-D-20260901-20261022.json"},
		"data-go-kr-kodex-inverse":  {"20251030", "20261030", "data-go-kr/114800-D-20260901-20261022.json"},
	}
	if len(entries) != len(want) {
		t.Fatalf("indicators.yaml has %d indicators, test knows %d", len(entries), len(want))
	}
	for _, e := range entries {
		w, ok := want[e.ID]
		if !ok {
			t.Errorf("%s: no expected sample", e.ID)
			continue
		}
		fetchSample(t, srcs, e.Indicator, w.start, w.end, w.file)
	}
}

// 같은 표의 샘플이 여러 개면 항목·분류 코드가 맞는 파일을 받는다(Issue #59).
func TestSameTable(t *testing.T) {
	srcs := fakeSources(t)
	kos := func(stat, item string, classes ...string) config.Indicator {
		return config.Indicator{Source: "kosis", OrgID: "101", StatCode: stat, ItemCode: item, Classes: classes, Cycle: "M"}
	}
	eco := func(stat, item string, classes ...string) config.Indicator {
		return config.Indicator{Source: "ecos", StatCode: stat, ItemCode: item, Classes: classes, Cycle: "M"}
	}
	for _, c := range []struct {
		ind  config.Indicator
		file string
	}{
		{kos("DT_1DA7001S", "T90", "0"), "kosis/DT_1DA7001S-T90-0-M-202401-202609.json"},
		{kos("DT_1DA7001S", "T30", "0"), "kosis/DT_1DA7001S-T30-0-M-202401-202609.json"},
		{kos("DT_1DA7001S", "T80", "0"), "kosis/DT_1DA7001S-T80-0-M-202401-202609.json"},
		{kos("DT_1J22042", "T03", "0"), "kosis/DT_1J22042-T03-0-M-202401-202609.json"},
		{kos("DT_1J22042", "T03", "4"), "kosis/DT_1J22042-T03-4-M-202401-202609.json"},
		{kos("DT_1DA7002S", "T90", "75"), "kosis/DT_1DA7002S-T90-75-M-202401-202609.json"},
		{kos("DT_1DA7002S", "T10", "63"), "kosis/DT_1DA7002S-T10-63-M-202401-202609.json"},
		// 분류가 둘
		{kos("DT_1F02001", "T20", "00", "C"), "kosis/DT_1F02001-T20-00-C-M-202401-202609.json"},
		{eco("721Y001", "5020000"), "ecos/721Y001-5020000-M-202401-202610.json"},
		{eco("721Y001", "5050000"), "ecos/721Y001-5050000-M-202401-202610.json"},
		{eco("721Y001", "7020000"), "ecos/721Y001-7020000-M-202401-202610.json"},
		// ECOS ITEM_CODE2
		{eco("731Y004", "0000001", "0000100"), "ecos/731Y004-0000001-0000100-M-202401-202610.json"},
		{config.Indicator{Source: "ecos", StatCode: "200Y102", ItemCode: "10111", Cycle: "Q"}, "ecos/200Y102-10111-Q-2024Q1-2026Q3.json"},
		{config.Indicator{Source: "ecos", StatCode: "200Y102", ItemCode: "10211", Cycle: "Q"}, "ecos/200Y102-10211-Q-2024Q1-2026Q3.json"},
	} {
		fetchSample(t, srcs, c.ind, "201610", "202610", c.file)
	}
}

// 맞는 샘플이 없으면 404다. 통계 코드만 맞는 다른 파일로 대신 보내지 않는다.
func TestNotFound(t *testing.T) {
	srv := httptest.NewServer(handler(samples))
	defer srv.Close()
	for _, path := range []string{
		// 주기가 다르다
		"/api/StatisticSearch/KEY/json/kr/1/1000/722Y001/Q/2016Q4/2026Q4/0101000",
		// 항목이 없다
		"/api/StatisticSearch/KEY/json/kr/1/1000/722Y001/M/201610/202610/0101001",
		// 분류가 하나 더 있다
		"/api/StatisticSearch/KEY/json/kr/1/1000/722Y001/M/201610/202610/0101000/X",
		// 항목 코드가 빠졌다
		"/api/StatisticSearch/KEY/json/kr/1/1000/722Y001/M/201610/202610",
		// 표는 있고 항목이 없다(예전에는 이름순 첫 파일을 보냈다)
		"/openapi/Param/statisticsParameterData.do?apiKey=KEY&tblId=DT_1DA7001S&itmId=T50&objL1=0&prdSe=M",
		// 분류가 다르다
		"/openapi/Param/statisticsParameterData.do?apiKey=KEY&tblId=DT_1DA7001S&itmId=T90&objL1=2&prdSe=M",
		// itmId가 없다
		"/openapi/Param/statisticsParameterData.do?apiKey=KEY&tblId=DT_1IN1502&prdSe=Y",
		// 영문, 숫자, _가 아닌 글자
		"/openapi/Param/statisticsParameterData.do?tblId=../x&itmId=T1&prdSe=M",
		"/openapi/Param/statisticsParameterData.do?tblId=DT_MLTM_2080&itmId=13103792722T1&objL1=13102792722A.0001&objL2=13102792722B.0001&objL3=13102792722C.0001&prdSe=M",
		"/api/StatisticSearch/KEY/json/kr/1/1000/404Y014/M/201610/202610/*AA",
		"/other",
	} {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("%s: status %d, want 404", path, resp.StatusCode)
		}
	}
}
