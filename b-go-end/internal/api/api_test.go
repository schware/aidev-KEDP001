package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/schware/aidev-KEDP001/b-go-end/internal/store"
)

var seoul = time.FixedZone("KST", 9*60*60)

type fakeStore struct {
	cards  []store.Card
	detail map[string]store.Detail
	err    error
	// Detail이 받은 값
	today  time.Time
	recent int
}

func (f *fakeStore) Cards(context.Context) ([]store.Card, error) { return f.cards, f.err }

func (f *fakeStore) Detail(_ context.Context, id string, today time.Time, recent int) (store.Detail, error) {
	f.today, f.recent = today, recent
	if f.err != nil {
		return store.Detail{}, f.err
	}
	d, ok := f.detail[id]
	if !ok {
		return store.Detail{}, store.ErrNotFound
	}
	return d, nil
}

func ptr[T any](v T) *T { return &v }

func date(s string) time.Time {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		panic(err)
	}
	return t
}

func get(t *testing.T, st Store, now time.Time, method, path string) (int, string) {
	t.Helper()
	h := Handler(st, func() time.Time { return now }, seoul)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("%s %s: Content-Type = %q", method, path, ct)
	}
	b, _ := io.ReadAll(rec.Body)
	return rec.Code, strings.TrimSpace(string(b))
}

func TestCards(t *testing.T) {
	st := &fakeStore{cards: []store.Card{
		{ID: "ecos-base-rate", Name: "기준금리", Cycle: "M", Unit: ptr("연%"),
			Latest: &store.Point{Period: date("2026-09-01"), Value: 2.5}},
		{ID: "data-go-kr-kodex-200", Name: "KODEX 200", Cycle: "D"},
	}}
	code, body := get(t, st, time.Now(), http.MethodGet, "/api/indicators")
	want := `{"indicators":[` +
		`{"id":"ecos-base-rate","name":"기준금리","cycle":"M","unit":"연%","latest":{"period":"2026-09-01","value":2.5}},` +
		`{"id":"data-go-kr-kodex-200","name":"KODEX 200","cycle":"D","unit":null,"latest":null}]}`
	if code != http.StatusOK || body != want {
		t.Errorf("got %d %s\nwant %s", code, body, want)
	}
}

func TestCardsEmpty(t *testing.T) {
	code, body := get(t, &fakeStore{}, time.Now(), http.MethodGet, "/api/indicators")
	if code != http.StatusOK || body != `{"indicators":[]}` {
		t.Errorf("got %d %s", code, body)
	}
}

func TestDetail(t *testing.T) {
	st := &fakeStore{detail: map[string]store.Detail{"ecos-base-rate": {
		ID: "ecos-base-rate", Name: "기준금리", Cycle: "M", StatCode: "722Y001",
		SourceCode: "ecos", SourceName: "ECOS", Unit: ptr("연%"), ChartPeriod: ptr("10 years"),
		Latest:          &store.Point{Period: date("2026-09-01"), Value: 2.5},
		LastCollectedAt: ptr(time.Date(2026, 10, 24, 21, 0, 3, 500, time.UTC)),
		Chart:           []store.Point{{Period: date("2026-08-01"), Value: 2.5}, {Period: date("2026-09-01"), Value: 2.5}},
		Recent:          []store.Point{{Period: date("2026-09-01"), Value: 2.5}},
		Related:         []store.Related{{ID: "kosis-employment-rate", Name: "고용률"}},
	}}}
	// 서울은 이미 다음 날이다.
	now := time.Date(2026, 10, 29, 16, 30, 0, 0, time.UTC)
	code, body := get(t, st, now, http.MethodGet, "/api/indicators/ecos-base-rate")
	want := `{"id":"ecos-base-rate","name":"기준금리","cycle":"M","unit":"연%","notice":null,` +
		`"source":{"code":"ecos","name":"ECOS"},"stat_code":"722Y001","last_collected_at":"2026-10-25T06:00:03+09:00",` +
		`"latest":{"period":"2026-09-01","value":2.5},"chart_period":"10 years",` +
		`"chart":[{"period":"2026-08-01","value":2.5},{"period":"2026-09-01","value":2.5}],` +
		`"recent":[{"period":"2026-09-01","value":2.5}],"related":[{"id":"kosis-employment-rate","name":"고용률"}]}`
	if code != http.StatusOK || body != want {
		t.Errorf("got %d %s\nwant %s", code, body, want)
	}
	if got := st.today.Format(time.DateOnly); got != "2026-10-30" {
		t.Errorf("today = %s, want 2026-10-30 (Asia/Seoul)", got)
	}
	if st.recent != RecentRows {
		t.Errorf("recent = %d", st.recent)
	}
}

func TestDetailEmpty(t *testing.T) {
	// 관측값, 수집 기록, 화면 설정, 관련 지표가 없다.
	st := &fakeStore{detail: map[string]store.Detail{"x": {
		ID: "x", Name: "X", Cycle: "A", StatCode: "S", SourceCode: "kosis", SourceName: "KOSIS",
	}}}
	code, body := get(t, st, time.Now(), http.MethodGet, "/api/indicators/x")
	want := `{"id":"x","name":"X","cycle":"A","unit":null,"notice":null,"source":{"code":"kosis","name":"KOSIS"},` +
		`"stat_code":"S","last_collected_at":null,"latest":null,"chart_period":null,"chart":[],"recent":[],"related":[]}`
	if code != http.StatusOK || body != want {
		t.Errorf("got %d %s\nwant %s", code, body, want)
	}
}

func TestNotFound(t *testing.T) {
	st := &fakeStore{}
	for _, path := range []string{"/api/indicators/none", "/api/indicators/a/b", "/api/other", "/"} {
		code, body := get(t, st, time.Now(), http.MethodGet, path)
		if code != http.StatusNotFound || !strings.HasPrefix(body, `{"error":`) {
			t.Errorf("%s: got %d %s", path, code, body)
		}
	}
	code, body := get(t, st, time.Now(), http.MethodGet, "/api/indicators/none")
	if body != `{"error":"indicator not found"}` {
		t.Errorf("got %d %s", code, body)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	h := Handler(&fakeStore{}, time.Now, seoul)
	for _, path := range []string{"/api/indicators", "/api/indicators/x"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, path, nil))
		if rec.Code != http.StatusMethodNotAllowed || rec.Header().Get("Allow") != "GET, HEAD" ||
			strings.TrimSpace(rec.Body.String()) != `{"error":"method not allowed"}` {
			t.Errorf("POST %s: got %d %s", path, rec.Code, rec.Body)
		}
	}
}

func TestServerErrorHidesDetail(t *testing.T) {
	st := &fakeStore{err: errors.New("read cards: secret detail")}
	for _, path := range []string{"/api/indicators", "/api/indicators/x"} {
		code, body := get(t, st, time.Now(), http.MethodGet, path)
		if code != http.StatusInternalServerError || body != `{"error":"internal error"}` {
			t.Errorf("%s: got %d %s", path, code, body)
		}
	}
}
