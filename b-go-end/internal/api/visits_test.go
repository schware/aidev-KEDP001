package api

import (
	"bytes"
	"context"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/schware/aidev-KEDP001/b-go-end/internal/store"
)

// fakeVisits는 날짜별 숫자만 세는 가짜 DB다.
type fakeVisits struct {
	mu    sync.Mutex
	days  map[string]int64
	base  int64
	err   error
	today time.Time // VisitSummary가 받은 값
}

func (f *fakeVisits) AddVisit(_ context.Context, day time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	if f.days == nil {
		f.days = map[string]int64{}
	}
	f.days[day.Format(time.DateOnly)]++
	return nil
}

func (f *fakeVisits) VisitSummary(_ context.Context, today time.Time) (store.VisitSummary, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.today = today
	s := store.VisitSummary{Today: f.days[today.Format(time.DateOnly)], Total: f.base}
	for _, n := range f.days {
		s.Total += n
	}
	return s, f.err
}

const (
	userA   = "198.51.100.7:50000"
	userB   = "198.51.100.8:50000"
	proxy   = "172.18.0.5:40000" // 믿을 proxy(nginx) 주소
	uaA     = "Mozilla/5.0 test-browser-A"
	uaB     = "Mozilla/5.0 test-browser-B"
	trusted = "172.18.0.0/16"
)

type visitHarness struct {
	t   *testing.T
	h   http.Handler
	v   *visits
	st  *fakeVisits
	now time.Time
}

func newVisitHarness(t *testing.T, limit int) *visitHarness {
	x := &visitHarness{t: t, st: &fakeVisits{base: 1000},
		// 2026-10-30 12:00 Asia/Seoul
		now: time.Date(2026, 10, 30, 3, 0, 0, 0, time.UTC)}
	x.v, x.h = newVisitHandler(VisitConfig{
		Store:          x.st,
		TrustedProxies: []netip.Prefix{netip.MustParsePrefix(trusted)},
		Now:            func() time.Time { return x.now },
		Loc:            seoul,
		Limit:          limit,
	})
	return x
}

func (x *visitHarness) visit(remote, ua string, header ...string) *httptest.ResponseRecorder {
	x.t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/visits", nil)
	req.RemoteAddr = remote
	if ua != "" {
		req.Header.Set("User-Agent", ua)
	}
	for i := 0; i+1 < len(header); i += 2 {
		req.Header.Add(header[i], header[i+1])
	}
	rec := httptest.NewRecorder()
	x.h.ServeHTTP(rec, req)
	return rec
}

func (x *visitHarness) count(day string) int64 {
	x.st.mu.Lock()
	defer x.st.mu.Unlock()
	return x.st.days[day]
}

func TestVisitSamePersonOncePerDay(t *testing.T) {
	x := newVisitHarness(t, 0)
	for range 3 {
		rec := x.visit(userA, uaA)
		if rec.Code != http.StatusNoContent || rec.Body.Len() != 0 || rec.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("got %d %q %v", rec.Code, rec.Body.String(), rec.Header())
		}
	}
	if n := x.count("2026-10-30"); n != 1 {
		t.Errorf("same person: count = %d, want 1", n)
	}
	// 주소가 다르거나 User-Agent가 다르면 다른 사람이다.
	x.visit(userB, uaA)
	x.visit(userA, uaB)
	x.visit(userA, "") // User-Agent 없음도 하나의 값이다
	x.visit(userA, "")
	if n := x.count("2026-10-30"); n != 4 {
		t.Errorf("count = %d, want 4", n)
	}
	// 같은 주소의 port만 다르면 같은 사람이다.
	x.visit("198.51.100.7:60000", uaA)
	if n := x.count("2026-10-30"); n != 4 {
		t.Errorf("other port: count = %d, want 4", n)
	}
}

func TestVisitSeoulMidnight(t *testing.T) {
	x := newVisitHarness(t, 0)
	// 2026-10-30 23:59:59 Asia/Seoul
	x.now = time.Date(2026, 10, 30, 14, 59, 59, 0, time.UTC)
	x.visit(userA, uaA)
	saltBefore := x.salt()
	x.visit(userA, uaA)
	// 2026-10-31 00:00:00 Asia/Seoul(UTC로는 아직 10-30): 날이 바뀌어 다시 센다.
	x.now = time.Date(2026, 10, 30, 15, 0, 0, 0, time.UTC)
	x.visit(userA, uaA)
	x.visit(userA, uaA)
	if a, b := x.count("2026-10-30"), x.count("2026-10-31"); a != 1 || b != 1 {
		t.Errorf("10-30 = %d, 10-31 = %d, want 1, 1", a, b)
	}
	// 날이 바뀌면 salt를 새로 만들고 전날 목록은 버린다.
	if bytes.Equal(saltBefore, x.salt()) {
		t.Error("salt did not change at midnight")
	}
	if n := x.seenLen(); n != 1 {
		t.Errorf("seen = %d, want 1 (only today)", n)
	}
	// 오늘 숫자도 Asia/Seoul 날짜로 읽는다.
	if code, body := x.summary(); code != http.StatusOK || body != `{"today":1,"total":1002}` {
		t.Errorf("summary: %d %s", code, body)
	}
	if got := x.st.today.Format(time.DateOnly); got != "2026-10-31" {
		t.Errorf("summary day = %s", got)
	}
}

func (x *visitHarness) salt() []byte {
	x.v.seen.mu.Lock()
	defer x.v.seen.mu.Unlock()
	return append([]byte(nil), x.v.seen.salt...)
}

func (x *visitHarness) seenLen() int {
	x.v.seen.mu.Lock()
	defer x.v.seen.mu.Unlock()
	return len(x.v.seen.seen)
}

func (x *visitHarness) summary() (int, string) {
	x.t.Helper()
	rec := httptest.NewRecorder()
	x.h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/visits/summary", nil))
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		x.t.Errorf("Content-Type = %q", ct)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-store" {
		x.t.Errorf("Cache-Control = %q", cc)
	}
	return rec.Code, strings.TrimSpace(rec.Body.String())
}

func TestVisitForwardedFor(t *testing.T) {
	x := newVisitHarness(t, 0)
	// 믿을 proxy에서 온 요청은 X-Forwarded-For의 마지막 주소로 사람을 나눈다.
	x.visit(proxy, uaA, "X-Forwarded-For", "203.0.113.1")
	x.visit(proxy, uaA, "X-Forwarded-For", "203.0.113.1")
	x.visit(proxy, uaA, "X-Forwarded-For", "203.0.113.2")
	x.visit(proxy, uaA, "X-Forwarded-For", "10.0.0.9, 203.0.113.2") // 앞쪽은 보지 않는다
	if n := x.count("2026-10-30"); n != 2 {
		t.Errorf("via proxy: count = %d, want 2", n)
	}
	// 믿지 않는 주소가 보낸 X-Forwarded-For는 보지 않는다(바꿔 가며 보내도 1명).
	x.visit(userA, uaB, "X-Forwarded-For", "203.0.113.50")
	x.visit(userA, uaB, "X-Forwarded-For", "203.0.113.51")
	if n := x.count("2026-10-30"); n != 3 {
		t.Errorf("untrusted XFF: count = %d, want 3", n)
	}
	// proxy가 읽을 수 없는 주소를 넘기면 세지 않고 204다.
	if rec := x.visit(proxy, uaA, "X-Forwarded-For", "not-an-ip"); rec.Code != http.StatusNoContent {
		t.Errorf("bad XFF: %d", rec.Code)
	}
	if n := x.count("2026-10-30"); n != 3 {
		t.Errorf("bad XFF counted: %d", n)
	}
}

func TestVisitMethodsAndPaths(t *testing.T) {
	x := newVisitHarness(t, 0)
	for _, c := range []struct {
		method, path string
		code         int
		allow, body  string
	}{
		{http.MethodGet, "/api/visits", 405, "POST", `{"error":"method not allowed"}`},
		{http.MethodPut, "/api/visits", 405, "POST", `{"error":"method not allowed"}`},
		{http.MethodPost, "/api/visits/summary", 405, "GET, HEAD", `{"error":"method not allowed"}`},
		{http.MethodGet, "/api/visits/none", 404, "", `{"error":"not found"}`},
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(c.method, c.path, nil)
		req.RemoteAddr = userA
		x.h.ServeHTTP(rec, req)
		if rec.Code != c.code || rec.Header().Get("Allow") != c.allow || strings.TrimSpace(rec.Body.String()) != c.body {
			t.Errorf("%s %s: got %d %q %s", c.method, c.path, rec.Code, rec.Header().Get("Allow"), rec.Body.String())
		}
	}
	if n := x.count("2026-10-30"); n != 0 {
		t.Errorf("count = %d", n)
	}
	// 본문이 있어도 읽지 않고 센다.
	req := httptest.NewRequest(http.MethodPost, "/api/visits", strings.NewReader(`{"x":1}`))
	req.RemoteAddr = userA
	rec := httptest.NewRecorder()
	x.h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent || x.count("2026-10-30") != 1 {
		t.Errorf("with body: %d", rec.Code)
	}
}

// DB 오류면 500이고, 그 사람은 다음 요청에서 다시 센다. 서버 기록에 주소, User-Agent, hash가 없다.
func TestVisitStoreErrorNoPersonalData(t *testing.T) {
	x := newVisitHarness(t, 0)
	var logs bytes.Buffer
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	x.st.err = errors.New("update visit_daily: boom")
	if rec := x.visit(proxy, uaA, "X-Forwarded-For", "203.0.113.77"); rec.Code != http.StatusInternalServerError ||
		strings.TrimSpace(rec.Body.String()) != `{"error":"internal error"}` {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
	if code, _ := x.summary(); code != http.StatusInternalServerError {
		t.Errorf("summary: %d", code)
	}
	x.st.err = nil
	x.visit(proxy, uaA, "X-Forwarded-For", "203.0.113.77")
	if n := x.count("2026-10-30"); n != 1 {
		t.Errorf("after DB error: count = %d, want 1", n)
	}
	out := logs.String()
	if !strings.Contains(out, "POST /api/visits: update visit_daily: boom") {
		t.Errorf("log = %q", out)
	}
	for _, s := range []string{"203.0.113.77", "172.18.0.5", uaA, "test-browser"} {
		if strings.Contains(out, s) {
			t.Errorf("log has %q: %q", s, out)
		}
	}
}

func TestVisitLimit(t *testing.T) {
	x := newVisitHarness(t, 2)
	var logs bytes.Buffer
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	x.visit(userA, uaA)
	x.visit(userB, uaA)
	x.visit(userA, uaB) // 상한을 넘어 세지 않는다
	x.visit(userB, uaB)
	x.visit(userA, uaA) // 이미 센 사람
	if n := x.count("2026-10-30"); n != 2 {
		t.Errorf("count = %d, want 2", n)
	}
	if c := strings.Count(logs.String(), "not counting more today"); c != 1 {
		t.Errorf("limit logged %d times: %q", c, logs.String())
	}
	// 다음 날에는 다시 센다.
	x.now = x.now.Add(24 * time.Hour)
	x.visit(userA, uaB)
	if n := x.count("2026-10-31"); n != 1 {
		t.Errorf("next day: count = %d", n)
	}
}

func TestVisitConcurrent(t *testing.T) {
	x := newVisitHarness(t, 0)
	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// 10명이 5번씩
			x.visit(userA, "ua-"+string(rune('a'+i%10)))
		}()
	}
	wg.Wait()
	if n := x.count("2026-10-30"); n != 10 {
		t.Errorf("count = %d, want 10", n)
	}
}
