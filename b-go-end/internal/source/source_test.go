package source

import (
	"errors"
	"fmt"
	"testing"

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
