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
