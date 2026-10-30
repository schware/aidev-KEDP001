package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHandler(t *testing.T) {
	samples := filepath.Join("..", "..", "..", "samples")
	srv := httptest.NewServer(handler(samples))
	defer srv.Close()

	tests := []struct {
		path   string
		status int
		file   string
	}{
		{"/api/StatisticSearch/KEY/json/kr/1/1000/722Y001/M/201610/202610/0101000", 200, "ecos/722Y001-0101000-M-202401-202609.json"},
		{"/openapi/Param/statisticsParameterData.do?apiKey=KEY&tblId=DT_1IN1502&prdSe=Y", 200, "kosis/DT_1IN1502-T100-00-Y-2015-2025.json"},
		{"/1160100/GetSecuritiesProductInfoService_V2/getETFPriceInfo_V2?serviceKey=KEY&likeSrtnCd=069500", 200, "data-go-kr/069500-D-20260901-20261022.json"},
		{"/api/StatisticSearch/KEY/json/kr/1/1000/722Y001/Q/2016Q4/2026Q4/0101000", 404, ""},
		{"/openapi/Param/statisticsParameterData.do?tblId=../x", 404, ""},
		{"/other", 404, ""},
	}
	for _, tt := range tests {
		resp, err := http.Get(srv.URL + tt.path)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != tt.status {
			t.Errorf("%s: status %d, want %d", tt.path, resp.StatusCode, tt.status)
			continue
		}
		if tt.file == "" {
			continue
		}
		want, err := os.ReadFile(filepath.Join(samples, tt.file))
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != string(want) {
			t.Errorf("%s: body is not %s", tt.path, tt.file)
		}
	}
}
