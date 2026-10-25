package fetch

import (
	"bytes"
	"context"
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
	if err == nil || strings.Contains(err.Error(), key) {
		t.Errorf("status error: %v", err)
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

	path, err := c.Save(dir+"/ecos", "ok.json", []byte(`{"ok":true}`))
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(path); string(b) != `{"ok":true}` {
		t.Errorf("saved = %s", b)
	}
}
