package main

import (
	"testing"

	"github.com/schware/aidev-KEDP001/b-go-end/internal/config"
)

func entry(id, period string) config.Entry {
	return config.Entry{Indicator: config.Indicator{ID: id}, Display: config.Display{ChartPeriod: period}}
}

func TestValidate(t *testing.T) {
	list, err := config.LoadIndicators("../../config/indicators.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if err := validate(list); err != nil {
		t.Errorf("indicators.yaml: %v", err)
	}

	for name, bad := range map[string][]config.Entry{
		"empty":           nil,
		"no id":           {entry("", "1 year")},
		"duplicated id":   {entry("a", "1 year"), entry("a", "1 year")},
		"no chart period": {entry("a", "")},
	} {
		if err := validate(bad); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}
