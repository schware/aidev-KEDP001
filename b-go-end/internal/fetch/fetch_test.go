package fetch

import (
	"bytes"
	"context"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

const key = "abc+/=KEY"

func TestMask(t *testing.T) {
	cases := []string{
		"https://example.com/api/" + key + "/json",
		"https://example.com/api/abc+%2F=KEY/json",
		"https://example.com/api?serviceKey=abc%2B%2F%3DKEY",
	}
	for _, s := range cases {
		if got := Mask(s, key); strings.Contains(got, "KEY") || !strings.Contains(got, "***") {
			t.Errorf("Mask(%q) = %q", s, got)
		}
	}
	// 이미 인코딩된 키(공공데이터포털 Encoding 키)는 푼 형태도 가린다.
	encoded := "abc%2B%2F%3DKEY"
	for _, s := range []string{"serviceKey=" + encoded, "serviceKey=" + key, "/api/abc+%2F=KEY/"} {
		if got := Mask(s, encoded); strings.Contains(got, "KEY") {
			t.Errorf("Mask(%q, encoded) = %q", s, got)
		}
	}
	if got := Mask("no key here", ""); got != "no key here" {
		t.Errorf("empty secret changed text: %q", got)
	}
}

func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	return &buf
}

func TestGetMasksLogAndErrors(t *testing.T) {
	logs := captureLog(t)

	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
	}))
	defer slow.Close()
	c := New(20*time.Millisecond, key)
	_, err := c.Get(context.Background(), slow.URL+"/api/"+key+"/json")
	if err == nil {
		t.Fatal("want timeout error")
	}
	if strings.Contains(err.Error(), key) || !strings.Contains(err.Error(), "***") {
		t.Errorf("timeout error not masked: %v", err)
	}

	notFound := httptest.NewServer(http.NotFoundHandler())
	defer notFound.Close()
	_, err = New(time.Second, key).Get(context.Background(), notFound.URL+"/api/"+key)
	var statusErr *StatusError
	if !errors.As(err, &statusErr) || statusErr.Code != http.StatusNotFound || strings.Contains(err.Error(), key) {
		t.Errorf("status error: %v", err)
	}
	if !strings.Contains(string(statusErr.Body), "not found") {
		t.Errorf("status error body = %q", statusErr.Body)
	}

	if strings.Contains(logs.String(), key) || !strings.Contains(logs.String(), "GET ") {
		t.Errorf("log not masked: %s", logs.String())
	}
}

func TestRecorder(t *testing.T) {
	captureLog(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/bad") {
			http.Error(w, `{"err":"bad"}`, http.StatusForbidden)
			return
		}
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()
	c := New(time.Second, key)

	var got []Exchange
	ctx := WithRecorder(context.Background(), func(e Exchange) { got = append(got, e) })
	if _, err := c.Get(ctx, srv.URL+"/api/"+key+"/ok"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Get(ctx, srv.URL+"/api/"+key+"/bad"); err == nil {
		t.Fatal("want status error")
	}
	if len(got) != 2 {
		t.Fatalf("recorded %d, want 2", len(got))
	}
	if got[0].Status != http.StatusOK || string(got[0].Body) != `{"ok":true}` || got[0].ReceivedAt.IsZero() {
		t.Errorf("ok = %+v", got[0])
	}
	if got[1].Status != http.StatusForbidden || !strings.Contains(string(got[1].Body), `"err"`) {
		t.Errorf("bad = %+v", got[1])
	}
	for _, e := range got {
		if strings.Contains(e.URL, key) || !strings.Contains(e.URL, "/api/***/") {
			t.Errorf("url not masked: %s", e.URL)
		}
	}

	// recorder가 없는 ctx는 그대로 동작한다.
	if _, err := c.Get(context.Background(), srv.URL+"/ok"); err != nil {
		t.Fatal(err)
	}
}

func TestSave(t *testing.T) {
	dir := t.TempDir()
	c := New(time.Second, key)

	if _, err := c.Save(dir, "leak.json", []byte(`{"url":"/api/`+key+`"}`)); err == nil {
		t.Error("want error when body contains key")
	}
	if _, err := os.Stat(dir + "/leak.json"); !os.IsNotExist(err) {
		t.Error("leak.json must not be written")
	}

	if _, err := c.Save(dir, "leak2.json", []byte(`{"url":"/api/abc%2B%2F%3DKEY"}`)); err == nil {
		t.Error("want error when body contains encoded key")
	}

	path, err := c.Save(dir+"/ecos", "ok.json", []byte(`{"ok":true}`))
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(path); string(b) != `{"ok":true}` {
		t.Errorf("saved = %s", b)
	}
}

func TestHasKey(t *testing.T) {
	c := New(time.Second, key)
	if !c.HasKey([]byte(`{"url":"/api/abc%2B%2F%3DKEY"}`)) || c.HasKey([]byte(`{"ok":true}`)) {
		t.Error("HasKey")
	}
}

// SetHost를 하면 경로와 질의는 그대로 두고 시험용 서버로 보낸다. 기록에는 바꾼 주소를 가려서 남긴다.
func TestSetHost(t *testing.T) {
	captureLog(t)
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.EscapedPath(), r.URL.RawQuery
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	c := New(time.Second, key)
	if err := c.SetHost(srv.URL); err != nil {
		t.Fatal(err)
	}

	var got Exchange
	ctx := WithRecorder(context.Background(), func(e Exchange) { got = e })
	if _, err := c.Get(ctx, "https://ecos.bok.or.kr/api/x?apiKey=k&a=1"); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/api/x" || gotQuery != "apiKey=k&a=1" {
		t.Errorf("path = %s, query = %s", gotPath, gotQuery)
	}
	if !strings.HasPrefix(got.URL, srv.URL+"/api/x?") {
		t.Errorf("recorded url = %s", got.URL)
	}

	for _, bad := range []string{"", "127.0.0.1:1", "http://"} {
		if err := c.SetHost(bad); err == nil {
			t.Errorf("SetHost(%q): want error", bad)
		}
	}
}
