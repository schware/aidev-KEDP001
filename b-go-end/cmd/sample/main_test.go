package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/schware/aidev-KEDP001/b-go-end/internal/fetch"
)

const key = "abc+/=KEY"

func TestNormalize(t *testing.T) {
	cases := []struct {
		req      Request
		api      string
		fileName string
	}{
		{Request{Source: "ecos", Stat: "722Y001", Item: "0101000", Cycle: "M", Start: "202401", End: "202609"},
			apiSearch, "722Y001-0101000-M-202401-202609.json"},
		{Request{Source: "ecos", API: "items", Stat: "722Y001"}, apiItems, "722Y001-items.json"},
		{Request{Source: "kosis", Stat: "DT_1DA7001S", Item: "T90", Classes: []string{"0", "1"}, Cycle: "Q", Start: "2024", End: "2025"},
			apiData, "DT_1DA7001S-T90-0-1-Q-2024-2025.json"},
		{Request{Source: "kosis", API: "meta", Stat: "DT_1DA7001S", MetaType: "ITM"}, apiMeta, "DT_1DA7001S-meta-ITM.json"},
		{Request{Source: "data-go-kr", Stat: "069500", Start: "20260901", End: "20261022"}, apiPrice, "069500-D-20260901-20261022.json"},
		{Request{Source: "ecos", API: "items", Stat: "722Y001", Name: "my-items"}, apiItems, "my-items.json"},
	}
	for _, c := range cases {
		r := c.req
		if err := r.normalize(); err != nil {
			t.Errorf("%+v: %v", c.req, err)
			continue
		}
		if r.API != c.api || r.Name != c.fileName {
			t.Errorf("%+v: api %s, name %s", c.req, r.API, r.Name)
		}
	}
	if r := (Request{Name: "x.json"}); r.infoName() != "x.request.json" {
		t.Errorf("infoName = %s", r.infoName())
	}
}

func TestNormalizeErrors(t *testing.T) {
	cases := map[string]Request{
		"source":  {Source: "bok", Stat: "x"},
		"api":     {Source: "ecos", API: "meta", Stat: "x"},
		"missing": {Source: "kosis", Stat: "DT_1DA7001S", Item: "T90"},
		"meta":    {Source: "kosis", API: "meta", Stat: "DT_1DA7001S"},
		"date":    {Source: "data-go-kr", Stat: "069500"},
		"folder":  {Source: "ecos", API: "items", Stat: "x", Name: "../x"},
		"hidden":  {Source: "ecos", API: "items", Stat: "x", Name: ".x"},
	}
	for name, r := range cases {
		if err := r.normalize(); err == nil {
			t.Errorf("%s: want error", name)
		}
	}
}

func writeFile(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "requests.yaml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadList(t *testing.T) {
	reqs, err := loadList(writeFile(t, `
requests:
  - source: ecos
    api: items
    stat: 722Y001
  - source: kosis
    stat: DT_1DA7001S
    item: T90
    classes: ["0"]
    cycle: M
    start: "202401"
    end: "202403"
`))
	if err != nil {
		t.Fatal(err)
	}
	if len(reqs) != 2 || reqs[1].Classes[0] != "0" || reqs[1].Start != "202401" {
		t.Errorf("reqs = %+v", reqs)
	}
	if err := prepare(reqs); err != nil {
		t.Error(err)
	}

	// 모르는 칸(키 등)은 받지 않는다.
	if _, err := loadList(writeFile(t, "requests:\n  - source: ecos\n    apiKey: x\n")); err == nil {
		t.Error("unknown field: want error")
	}
	if _, err := loadList(writeFile(t, "requests: []\n")); err == nil {
		t.Error("empty list: want error")
	}
}

func TestPrepareSameFile(t *testing.T) {
	reqs := []Request{
		{Source: "ecos", API: "items", Stat: "722Y001"},
		{Source: "ecos", API: "items", Stat: "722Y001"},
		{Source: "kosis", Stat: "x"},
	}
	err := prepare(reqs)
	if err == nil || !strings.Contains(err.Error(), "request 2: same file") || !strings.Contains(err.Error(), "request 3:") {
		t.Errorf("err = %v", err)
	}
}

func TestSave(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "kosis")
	client := fetch.New(time.Second, key)
	r := Request{Source: "kosis", API: apiData, Name: "x.json"}
	received := time.Date(2026, 10, 25, 16, 0, 0, 0, time.FixedZone("KST", 9*3600))
	e := fetch.Exchange{
		URL:        "https://kosis.kr/openapi/x?apiKey=***&tblId=x",
		Status:     200,
		ReceivedAt: received,
		Body:       []byte(`{"err":"30","errMsg":"데이터가 존재하지 않습니다."}`),
	}

	path, err := save(client, dir, r, e)
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(path); string(b) != string(e.Body) {
		t.Errorf("body = %s", b)
	}
	b, err := os.ReadFile(filepath.Join(dir, "x.request.json"))
	if err != nil {
		t.Fatal(err)
	}
	var info requestInfo
	if err := json.Unmarshal(b, &info); err != nil {
		t.Fatal(err)
	}
	want := requestInfo{Source: "kosis", API: apiData, URL: e.URL, Status: 200, ReceivedAt: "2026-10-25T16:00:00+09:00"}
	if info != want {
		t.Errorf("info = %+v", info)
	}

	// 이미 있으면 덮어쓰지 않는다.
	if _, err := save(client, dir, r, e); err == nil {
		t.Error("existing file: want error")
	}

	// 본문에 키가 있으면 아무것도 저장하지 않는다.
	leak := Request{Source: "kosis", API: apiData, Name: "leak.json"}
	if _, err := save(client, dir, leak, fetch.Exchange{Body: []byte(key)}); err == nil {
		t.Error("body with key: want error")
	}
	for _, name := range []string{"leak.json", "leak.request.json"} {
		if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Errorf("%s must not be written", name)
		}
	}
}
