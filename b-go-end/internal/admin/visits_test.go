package admin

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/schware/aidev-KEDP001/b-go-end/internal/store"
)

func (f *fakeStore) VisitStats(_ context.Context, today time.Time, days int) (store.VisitStats, error) {
	f.visitToday, f.visitDays = today, days
	return f.visits, f.err
}

func (f *fakeStore) SetVisitBase(_ context.Context, base int64) (string, error) {
	f.visitBase = &base
	return "1000", f.err
}

func (f *fakeStore) ResetVisits(context.Context) (store.VisitReset, error) {
	f.reset = true
	return store.VisitReset{Days: 3, Count: 42, OldBase: "1000"}, f.err
}

func visitStats() store.VisitStats {
	return store.VisitStats{Today: 2, Total: 1045, Base: 1000, Days: []store.VisitDay{
		{Day: time.Date(2026, 10, 29, 0, 0, 0, 0, time.UTC), Count: 0},
		{Day: time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC), Count: 2},
	}}
}

const visitsBody = `{"today":2,"total":1045,"base":1000,"days":[{"day":"2026-10-29","count":0},{"day":"2026-10-30","count":2}]}`

func TestGetVisits(t *testing.T) {
	st := &fakeStore{visits: visitStats()}
	x := newHarness(t, st)
	// 2026-10-29 15:00 UTC = 2026-10-30 00:00 Asia/Seoul
	x.now = time.Date(2026, 10, 29, 15, 0, 0, 0, time.UTC)
	token := x.login()
	res := x.do(http.MethodGet, "/api/admin/visits", "", allowedAddr, token)
	if res.code != http.StatusOK || res.body != visitsBody {
		t.Errorf("got %d %s", res.code, res.body)
	}
	if st.visitDays != VisitDays || st.visitToday.Format(time.DateOnly) != "2026-10-30" {
		t.Errorf("store got today %v, days %d", st.visitToday, st.visitDays)
	}
	if res.header.Get("Cache-Control") != "no-store" {
		t.Errorf("Cache-Control = %q", res.header.Get("Cache-Control"))
	}
}

func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	return &buf
}

func TestSetVisitBase(t *testing.T) {
	st := &fakeStore{visits: visitStats()}
	x := newHarness(t, st)
	token := x.login()
	logs := captureLog(t)
	res := x.do(http.MethodPut, "/api/admin/visits/base", `{"base":1000}`, allowedAddr, token)
	if res.code != http.StatusOK || res.body != visitsBody || st.visitBase == nil || *st.visitBase != 1000 {
		t.Errorf("got %d %s", res.code, res.body)
	}
	if !strings.Contains(logs.String(), `admin audit: account "admin": visit_total_base 1000 -> 1000`) {
		t.Errorf("audit log: %q", logs.String())
	}
	for _, b := range []string{`{"base":0}`, `{"base":9007199254740991}`} {
		if res := x.do(http.MethodPut, "/api/admin/visits/base", b, allowedAddr, token); res.code != http.StatusOK {
			t.Errorf("%s: got %d %s", b, res.code, res.body)
		}
	}

	st.visitBase = nil
	want := `{"error":"invalid input","fields":{"base":"must be an integer from 0 to 9007199254740991"}}`
	for _, b := range []string{`{"base":-1}`, `{"base":1.5}`, `{"base":"10"}`, `{"base":null}`, `{}`,
		`{"base":9007199254740992}`, `{"base":99999999999999999999}`} {
		res := x.do(http.MethodPut, "/api/admin/visits/base", b, allowedAddr, token)
		if res.code != http.StatusBadRequest || res.body != want {
			t.Errorf("%s: got %d %s", b, res.code, res.body)
		}
	}
	if res := x.do(http.MethodPut, "/api/admin/visits/base", `{"base":1,"x":1}`, allowedAddr, token); res.code != http.StatusBadRequest {
		t.Errorf("unknown field: got %d %s", res.code, res.body)
	}
	if st.visitBase != nil {
		t.Errorf("bad input changed base to %d", *st.visitBase)
	}
}

func TestResetVisits(t *testing.T) {
	st := &fakeStore{visits: visitStats()}
	x := newHarness(t, st)
	token := x.login()
	logs := captureLog(t)
	// GET은 405다.
	if res := x.do(http.MethodGet, "/api/admin/visits/reset", "", allowedAddr, token); res.code != http.StatusMethodNotAllowed || st.reset {
		t.Errorf("GET: got %d %s", res.code, res.body)
	}
	res := x.do(http.MethodPost, "/api/admin/visits/reset", "{}", allowedAddr, token)
	if res.code != http.StatusOK || res.body != visitsBody || !st.reset {
		t.Errorf("got %d %s", res.code, res.body)
	}
	if !strings.Contains(logs.String(),
		`admin audit: account "admin": reset visits: deleted 3 days (42 visits), visit_total_base 1000 -> 0`) {
		t.Errorf("audit log: %q", logs.String())
	}
}

func TestVisitsStoreError(t *testing.T) {
	st := &fakeStore{err: context.DeadlineExceeded}
	x := newHarness(t, st)
	token := x.login()
	captureLog(t)
	for _, c := range []struct{ method, path, body string }{
		{http.MethodGet, "/api/admin/visits", ""},
		{http.MethodPut, "/api/admin/visits/base", `{"base":1}`},
		{http.MethodPost, "/api/admin/visits/reset", "{}"},
	} {
		res := x.do(c.method, c.path, c.body, allowedAddr, token)
		if res.code != http.StatusInternalServerError || res.body != `{"error":"internal error"}` {
			t.Errorf("%s %s: got %d %s", c.method, c.path, res.code, res.body)
		}
	}
}
