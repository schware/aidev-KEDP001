package logx

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func capture(t *testing.T, lv slog.Level) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	t.Cleanup(SetOutput(&buf, lv))
	return &buf
}

func lines(buf *bytes.Buffer) []string {
	s := strings.TrimSpace(buf.String())
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

// 한 줄의 앞: time(Asia/Seoul, 밀리초) level comp event, 그다음 ctx 값, 그다음 args. 공백이 있는 값은 따옴표.
func TestLineFormat(t *testing.T) {
	buf := capture(t, slog.LevelInfo)
	ctx := With(context.Background(), "req", "abc123", "run", int64(812))
	Info(ctx, "collect.indicator", "indicator", "cpi", "detail", "pages 1 (inserted 0)")
	got := lines(buf)
	if len(got) != 1 {
		t.Fatalf("lines = %q", got)
	}
	re := regexp.MustCompile(`^time=\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d\.\d{3}\+09:00 level=INFO comp=go event=collect\.indicator req=abc123 run=812 indicator=cpi detail="pages 1 \(inserted 0\)"$`)
	if !re.MatchString(got[0]) {
		t.Errorf("line = %q", got[0])
	}
}

func TestLevelFilter(t *testing.T) {
	buf := capture(t, slog.LevelWarn)
	ctx := context.Background()
	Debug(ctx, "a")
	Info(ctx, "b")
	Warn(ctx, "c")
	Error(ctx, "d")
	got := lines(buf)
	if len(got) != 2 || !strings.Contains(got[0], "level=WARN") || !strings.Contains(got[0], "event=c") ||
		!strings.Contains(got[1], "level=ERROR") || !strings.Contains(got[1], "event=d") {
		t.Errorf("lines = %q", got)
	}

	buf = capture(t, slog.LevelDebug)
	Debug(ctx, "a")
	if got := lines(buf); len(got) != 1 || !strings.Contains(got[0], "level=DEBUG") {
		t.Errorf("debug: %q", got)
	}
}

func TestParseLevel(t *testing.T) {
	for in, want := range map[string]slog.Level{
		"": slog.LevelInfo, "DEBUG": slog.LevelDebug, "INFO": slog.LevelInfo, "WARN": slog.LevelWarn,
		"ERROR": slog.LevelError, " debug ": slog.LevelDebug, "warn": slog.LevelWarn,
	} {
		if got, ok := ParseLevel(in); !ok || got != want {
			t.Errorf("%q: got %v %v", in, got, ok)
		}
	}
	for _, in := range []string{"WARNING", "TRACE", "FATAL", "1"} {
		if got, ok := ParseLevel(in); ok || got != slog.LevelInfo {
			t.Errorf("%q: got %v %v, want INFO false", in, got, ok)
		}
	}
}

func TestStatusLevel(t *testing.T) {
	for _, c := range []struct {
		method, path string
		status       int
		want         slog.Level
	}{
		{"GET", "/api/indicators", 200, slog.LevelInfo},
		{"POST", "/api/visits", 204, slog.LevelInfo},
		{"GET", "/api/x", 304, slog.LevelInfo},
		{"GET", "/api/indicators/x", 404, slog.LevelInfo},
		{"GET", "/api/admin/session", 401, slog.LevelInfo},
		{"POST", "/api/admin/login", 401, slog.LevelWarn},
		{"GET", "/api/admin/indicators", 401, slog.LevelWarn},
		{"GET", "/api/admin/session", 403, slog.LevelWarn},
		{"PUT", "/api/admin/collect-time", 400, slog.LevelWarn},
		{"POST", "/api/indicators", 405, slog.LevelWarn},
		{"GET", "/api/indicators", 500, slog.LevelError},
		{"GET", "/api/indicators", 503, slog.LevelError},
	} {
		if got := StatusLevel(c.method, c.path, c.status); got != c.want {
			t.Errorf("%s %s %d: got %v, want %v", c.method, c.path, c.status, got, c.want)
		}
	}
}

func serve(h http.Handler, method, target, reqID string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, target, nil)
	r.RemoteAddr = "203.0.113.9:5555"
	r.Header.Set("User-Agent", "test-browser/1.0")
	if reqID != "" {
		r.Header.Set(RequestIDHeader, reqID)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

func TestHTTPRequestLine(t *testing.T) {
	buf := capture(t, slog.LevelInfo)
	var inner string
	h := HTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Info(r.Context(), "inside")
		w.WriteHeader(http.StatusTeapot)
	}))
	rec := serve(h, http.MethodGet, "/api/indicators?secret=1", "4f1cabc")
	if rec.Header().Get(RequestIDHeader) != "4f1cabc" {
		t.Errorf("response header = %q", rec.Header().Get(RequestIDHeader))
	}
	got := lines(buf)
	if len(got) != 2 {
		t.Fatalf("lines = %q", got)
	}
	inner = got[0]
	if !strings.Contains(inner, "event=inside req=4f1cabc") {
		t.Errorf("inner line = %q", inner)
	}
	re := regexp.MustCompile(`level=WARN comp=go event=http\.request req=4f1cabc method=GET path=/api/indicators status=418 ms=\d+$`)
	if !re.MatchString(got[1]) {
		t.Errorf("request line = %q", got[1])
	}
	// 접속 주소, User-Agent, query string은 남기지 않는다.
	for _, s := range []string{"203.0.113.9", "test-browser", "secret"} {
		if strings.Contains(buf.String(), s) {
			t.Errorf("log has %q: %s", s, buf.String())
		}
	}
}

func TestHTTPRequestIDMadeWhenMissingOrInvalid(t *testing.T) {
	buf := capture(t, slog.LevelInfo)
	h := HTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	hex32 := regexp.MustCompile(`^[0-9a-f]{32}$`)
	for _, in := range []string{"", "bad id with spaces", "a\nlevel=ERROR", strings.Repeat("a", 65)} {
		buf.Reset()
		rec := serve(h, http.MethodGet, "/api/indicators", in)
		id := rec.Header().Get(RequestIDHeader)
		if !hex32.MatchString(id) {
			t.Errorf("%q: made id %q", in, id)
		}
		if !strings.Contains(buf.String(), "req="+id+" ") || strings.Count(buf.String(), "\n") != 1 {
			t.Errorf("%q: log = %q", in, buf.String())
		}
	}
	if a, b := serve(h, "GET", "/", "").Header().Get(RequestIDHeader), serve(h, "GET", "/", "").Header().Get(RequestIDHeader); a == b {
		t.Errorf("same id twice: %s", a)
	}
}

func TestHTTPStatusDefaultsTo200(t *testing.T) {
	buf := capture(t, slog.LevelInfo)
	HTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })).
		ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/api/x", nil))
	HTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).
		ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/api/y", nil))
	if got := lines(buf); len(got) != 2 || !strings.Contains(got[0], "status=200") || !strings.Contains(got[1], "status=200") {
		t.Errorf("lines = %q", got)
	}
}

// 500의 오류는 따로 찍지 않고 http.request 한 줄(ERROR)에 err로 붙는다.
func TestSetErrorOneLine(t *testing.T) {
	buf := capture(t, slog.LevelInfo)
	h := HTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		SetError(r, errors.New("select card: boom"))
		AddAttrs(r, "ip", "192.0.2.1")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	serve(h, http.MethodGet, "/api/indicators", "r1")
	got := lines(buf)
	re := regexp.MustCompile(`level=ERROR comp=go event=http\.request req=r1 method=GET path=/api/indicators status=500 ms=\d+ ip=192\.0\.2\.1 err="select card: boom"$`)
	if len(got) != 1 || !re.MatchString(got[0]) {
		t.Errorf("lines = %q", got)
	}
}

// middleware 밖에서는 SetError가 바로 http.error 한 줄을 남긴다.
func TestSetErrorWithoutMiddleware(t *testing.T) {
	buf := capture(t, slog.LevelInfo)
	r := httptest.NewRequest(http.MethodPost, "/api/visits", nil)
	SetError(r, errors.New("boom"))
	AddAttrs(r, "x", 1)
	got := lines(buf)
	if len(got) != 1 || !strings.Contains(got[0], `level=ERROR comp=go event=http.error method=POST path=/api/visits err=boom`) {
		t.Errorf("lines = %q", got)
	}
}
