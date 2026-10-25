package config

import (
	"os"
	"path/filepath"
	"reflect"
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

func TestLoadIndicators(t *testing.T) {
	list, err := LoadIndicators("../../config/indicators.yaml")
	if err != nil {
		t.Fatal(err)
	}
	ind, err := FindIndicator(list, "ecos-base-rate")
	if err != nil {
		t.Fatal(err)
	}
	want := Indicator{ID: "ecos-base-rate", Name: "한국은행 기준금리", Source: "ecos", StatCode: "722Y001", ItemCode: "0101000", Cycle: "M"}
	if !reflect.DeepEqual(ind, want) {
		t.Errorf("indicator = %+v", ind)
	}

	ind, err = FindIndicator(list, "kosis-employment-rate")
	if err != nil {
		t.Fatal(err)
	}
	want = Indicator{ID: "kosis-employment-rate", Name: "고용률(15세 이상)", Source: "kosis", StatCode: "DT_1DA7001S", ItemCode: "T90", Classes: []string{"0"}, Cycle: "M"}
	if !reflect.DeepEqual(ind, want) {
		t.Errorf("indicator = %+v", ind)
	}
	if _, err := FindIndicator(list, "none"); err == nil {
		t.Error("want not found error")
	}
}
