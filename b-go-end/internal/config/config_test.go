package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	content := "# comment\nECOS_API_KEY=ecos-test\nKOSIS_API_KEY=\"kosis-test\"\nDATA_GO_KR_API_KEY=\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ECOS_API_KEY", "")
	t.Setenv("KOSIS_API_KEY", "from-env")
	t.Setenv("DATA_GO_KR_API_KEY", "")

	k, err := LoadKeys(path)
	if err != nil {
		t.Fatal(err)
	}
	if k.ECOS != "ecos-test" || k.KOSIS != "from-env" || k.DataGoKr != "" {
		t.Errorf("keys = ecos:%q kosis:%q data:%q", k.ECOS, k.KOSIS, k.DataGoKr)
	}
	if got := k.All(); len(got) != 2 {
		t.Errorf("All() len = %d, want 2", len(got))
	}
}

func TestLoadKeysMissingFile(t *testing.T) {
	if _, err := LoadKeys(filepath.Join(t.TempDir(), "none")); err != nil {
		t.Errorf("missing .env: %v", err)
	}
}

func TestLoadDB(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	content := "KEDP_DB_HOST=db.test\nKEDP_DB_PORT=5432\nKEDP_DB_NAME=from-file\nkedp_go=secret-value\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"KEDP_DB_HOST", "KEDP_DB_PORT", "kedp_go"} {
		t.Setenv(name, "")
	}
	t.Setenv("KEDP_DB_NAME", "from-env")

	db, err := LoadDB(path)
	if err != nil {
		t.Fatal(err)
	}
	want := DB{Host: "db.test", Port: 5432, Name: "from-env", Password: "secret-value"}
	if db != want {
		t.Errorf("db = %#v", db)
	}
	// 찍어도 접속 정보가 나오지 않는다.
	if shown := fmt.Sprintf("%v %+v %s", db, db, db); strings.Contains(shown, "secret-value") || strings.Contains(shown, "db.test") {
		t.Errorf("shown = %s", shown)
	}
}

func TestLoadDBMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("KEDP_DB_HOST=h\nKEDP_DB_PORT=x\nkedp_go=secret-value\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"KEDP_DB_HOST", "KEDP_DB_PORT", "KEDP_DB_NAME", "kedp_go"} {
		t.Setenv(name, "")
	}
	_, err := LoadDB(path)
	if err == nil || err.Error() != "KEDP_DB_NAME is not set" {
		t.Errorf("err = %v", err)
	}
	t.Setenv("KEDP_DB_NAME", "n")
	_, err = LoadDB(path)
	if err == nil || !strings.Contains(err.Error(), "KEDP_DB_PORT") || strings.Contains(err.Error(), "secret-value") {
		t.Errorf("err = %v", err)
	}
}

func TestLoadProcessorAddr(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "none")
	t.Setenv("KEDP_PROCESSOR_ADDR", "")
	if addr, err := LoadProcessorAddr(missing); err != nil || addr != DefaultProcessorAddr {
		t.Errorf("default = %q, %v", addr, err)
	}
	t.Setenv("KEDP_PROCESSOR_ADDR", "127.0.0.1:50061")
	if addr, err := LoadProcessorAddr(missing); err != nil || addr != "127.0.0.1:50061" {
		t.Errorf("env = %q, %v", addr, err)
	}
}

func TestLoadAPIAddr(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "none")
	t.Setenv("KEDP_API_ADDR", "")
	if addr, err := LoadAPIAddr(missing); err != nil || addr != DefaultAPIAddr {
		t.Errorf("default = %q, %v", addr, err)
	}
	t.Setenv("KEDP_API_ADDR", "127.0.0.1:18081")
	if addr, err := LoadAPIAddr(missing); err != nil || addr != "127.0.0.1:18081" {
		t.Errorf("env = %q, %v", addr, err)
	}
}

func TestLoadIndicators(t *testing.T) {
	list, err := LoadIndicators("../../config/indicators.yaml")
	if err != nil {
		t.Fatal(err)
	}
	// 지표 7개, 화면 순서는 1부터 차례로다(Issue #34).
	wantOrder := []string{
		"ecos-base-rate", "kosis-employment-rate", "kosis-total-population", "kosis-births",
		"data-go-kr-kodex-200", "data-go-kr-kodex-leverage", "data-go-kr-kodex-inverse",
	}
	if len(list) != len(wantOrder) {
		t.Fatalf("len = %d, want %d", len(list), len(wantOrder))
	}
	byID := map[string]Entry{}
	for i, e := range list {
		if e.ID != wantOrder[i] || e.Display.Order != i+1 || !e.Display.Visible {
			t.Errorf("%d: id = %s, display = %+v", i, e.ID, e.Display)
		}
		byID[e.ID] = e
	}

	want := Entry{
		Indicator: Indicator{ID: "ecos-base-rate", Name: "한국은행 기준금리", Source: "ecos", StatCode: "722Y001", ItemCode: "0101000", Cycle: "M"},
		Display:   Display{Visible: true, Order: 1, ChartPeriod: "10 years"},
	}
	if got := byID["ecos-base-rate"]; !reflect.DeepEqual(got, want) {
		t.Errorf("ecos-base-rate = %+v", got)
	}
	want = Entry{
		Indicator: Indicator{ID: "kosis-births", Name: "출생아 수", Source: "kosis", OrgID: "101", StatCode: "INH_1B8000F_01", ItemCode: "T1", Classes: []string{"11"}, Cycle: "A"},
		Display:   Display{Visible: true, Order: 4, ChartPeriod: "20 years"},
	}
	if got := byID["kosis-births"]; !reflect.DeepEqual(got, want) {
		t.Errorf("kosis-births = %+v", got)
	}
	want = Entry{
		Indicator: Indicator{ID: "data-go-kr-kodex-leverage", Name: "KODEX 레버리지", Source: "data-go-kr", StatCode: "122630", ValueField: "clpr", Cycle: "D"},
		Display:   Display{Visible: true, Order: 6, ChartPeriod: "1 year", Notice: "이 정보는 신뢰할 수 없습니다. 투자 판단에 사용하지 마십시오."},
	}
	if got := byID["data-go-kr-kodex-leverage"]; !reflect.DeepEqual(got, want) {
		t.Errorf("data-go-kr-kodex-leverage = %+v", got)
	}

	// 제공처별로 DB가 검사하는 칸(설계서 4장)과 화면 처음 값을 확인한다.
	periods := map[string]string{"M": "10 years", "A": "20 years", "D": "1 year"}
	for _, e := range list {
		switch e.Source {
		case "kosis":
			if e.OrgID != "101" || e.ItemCode == "" {
				t.Errorf("%s: kosis org_id/item_code", e.ID)
			}
		case "data-go-kr":
			if e.ValueField != "clpr" || e.ItemCode != "" {
				t.Errorf("%s: data-go-kr value_field/item_code", e.ID)
			}
		}
		if e.Display.ChartPeriod != periods[e.Cycle] {
			t.Errorf("%s: chart_period = %q for cycle %s", e.ID, e.Display.ChartPeriod, e.Cycle)
		}
		leveraged := e.ID == "data-go-kr-kodex-leverage" || e.ID == "data-go-kr-kodex-inverse"
		if (e.Display.Notice != "") != leveraged {
			t.Errorf("%s: notice = %q", e.ID, e.Display.Notice)
		}
	}
}
