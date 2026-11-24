package api

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/schware/aidev-KEDP001/b-go-end/internal/logx"
)

type eventHarness struct {
	t    *testing.T
	h    http.Handler
	now  time.Time
	logs *bytes.Buffer
}

func newEventHarness(t *testing.T, perSecond int) *eventHarness {
	x := &eventHarness{t: t, now: time.Date(2026, 11, 24, 3, 0, 0, 0, time.UTC), logs: &bytes.Buffer{}}
	t.Cleanup(logx.SetOutput(x.logs, slog.LevelDebug))
	x.h = logx.HTTP(EventHandler(EventConfig{Now: func() time.Time { return x.now }, PerSecond: perSecond}))
	return x
}

// post는 본문을 보내고 응답 코드를 돌려준다. 접속 주소와 User-Agent를 넣어 로그에 없는지 본다.
func (x *eventHarness) post(body string) int {
	r := httptest.NewRequest(http.MethodPost, "/api/events", strings.NewReader(body))
	r.RemoteAddr = "203.0.113.77:5555"
	r.Header.Set("User-Agent", "test-browser/1.0")
	rec := httptest.NewRecorder()
	x.h.ServeHTTP(rec, r)
	if rec.Body.Len() != 0 {
		x.t.Errorf("%s: body %q", body, rec.Body.String())
	}
	return rec.Code
}

// uiLines는 comp=web 줄만 돌려준다.
func (x *eventHarness) uiLines() []string {
	var out []string
	for _, l := range strings.Split(strings.TrimSpace(x.logs.String()), "\n") {
		if strings.Contains(l, " comp=web ") {
			out = append(out, l)
		}
	}
	return out
}

func TestEventsAccepted(t *testing.T) {
	x := newEventHarness(t, 0)
	for _, c := range []struct{ body, want string }{
		{`{"event":"ui.page_view","page":"/"}`, `level=INFO comp=web event=ui\.page_view req=\w+ page=/$`},
		{`{"event":"ui.page_view","page":"/admin/dashboard"}`, `level=INFO comp=web event=ui\.page_view req=\w+ page=/admin/dashboard$`},
		{`{"event":"ui.card_open","indicator":"cpi","page":"/"}`, `level=INFO comp=web event=ui\.card_open req=\w+ indicator=cpi$`},
		{`{"event":"ui.related_open","indicator":"kosis-kospi","from":"cpi"}`, `level=INFO comp=web event=ui\.related_open req=\w+ indicator=kosis-kospi from=cpi$`},
		{`{"event":"ui.admin_action","action":"display_order_save"}`, `level=INFO comp=web event=ui\.admin_action req=\w+ action=display_order_save$`},
		{`{"event":"ui.error","path":"/api/indicators/cpi","status":500}`, `level=WARN comp=web event=ui\.error req=\w+ path=/api/indicators/cpi status=500$`},
		{`{"event":"ui.error","path":"/api/indicators","status":0}`, `level=WARN comp=web event=ui\.error req=\w+ path=/api/indicators status=0$`},
		// level은 Go가 정한다. 모르는 칸(level, extra)은 버린다.
		{`{"event":"ui.card_open","indicator":"gdp","level":"ERROR","extra":"x y"}`, `level=INFO comp=web event=ui\.card_open req=\w+ indicator=gdp$`},
	} {
		x.logs.Reset()
		if code := x.post(c.body); code != http.StatusNoContent {
			t.Errorf("%s: code %d", c.body, code)
		}
		got := x.uiLines()
		if len(got) != 1 || !regexp.MustCompile(c.want).MatchString(got[0]) {
			t.Errorf("%s: lines %q", c.body, got)
		}
	}
}

// 받지 않는 요청도 모두 204이고, comp=web 줄이 없다. 받은 값은 로그에 남지 않는다.
func TestEventsRejected(t *testing.T) {
	x := newEventHarness(t, 0)
	big := `{"event":"ui.page_view","page":"/","pad":"` + strings.Repeat("a", MaxEventBody) + `"}`
	for _, c := range []struct{ body, reason string }{
		{``, "invalid JSON"},
		{`not json`, "invalid JSON"},
		{`{"event":"ui.page_view","page":"/"} {}`, "invalid JSON"},
		{`[1,2]`, "invalid JSON"},
		{big, "too large"},
		{`{"page":"/"}`, "no event"},
		{`{"event":"ui.unknown_secret","page":"/"}`, "unknown event"},
		{`{"event":"http.request","page":"/"}`, "unknown event"},
		{`{"event":5}`, "unknown event"},
		{`{"event":"ui.page_view"}`, "missing page"},
		{`{"event":"ui.page_view","page":"/a?key=secret"}`, "invalid page"},
		{`{"event":"ui.page_view","page":"relative"}`, "invalid page"},
		{`{"event":"ui.page_view","page":"/a b"}`, "invalid page"},
		{`{"event":"ui.page_view","page":"/` + strings.Repeat("a", 100) + `"}`, "invalid page"},
		{`{"event":"ui.card_open","indicator":"CPI"}`, "invalid indicator"},
		{`{"event":"ui.card_open","indicator":"a\nlevel=ERROR"}`, "invalid indicator"},
		{`{"event":"ui.card_open","indicator":3}`, "invalid indicator"},
		{`{"event":"ui.related_open","indicator":"cpi"}`, "missing from"},
		{`{"event":"ui.admin_action","action":"drop_table"}`, "invalid action"},
		{`{"event":"ui.error","path":"/api/x","status":"500"}`, "invalid status"},
		{`{"event":"ui.error","path":"/api/x","status":700}`, "invalid status"},
		{`{"event":"ui.error","path":"/api/x","status":1.5}`, "invalid status"},
	} {
		x.logs.Reset()
		if code := x.post(c.body); code != http.StatusNoContent {
			t.Errorf("%.40q: code %d", c.body, code)
		}
		if got := x.uiLines(); len(got) != 0 {
			t.Errorf("%.40q: ui lines %q", c.body, got)
		}
		want := `level=DEBUG comp=go event=ui.rejected req=`
		if s := x.logs.String(); !strings.Contains(s, want) || !strings.Contains(s, `reason="`+c.reason+`"`) {
			t.Errorf("%.40q: want reason %q, log %q", c.body, c.reason, s)
		}
		for _, s := range []string{"secret", "drop_table", "203.0.113.77", "test-browser"} {
			if strings.Contains(x.logs.String(), s) {
				t.Errorf("%.40q: log has %q: %s", c.body, s, x.logs.String())
			}
		}
	}
}

func TestEventsMethod(t *testing.T) {
	x := newEventHarness(t, 0)
	rec := httptest.NewRecorder()
	x.h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/events", nil))
	if rec.Code != http.StatusMethodNotAllowed || rec.Header().Get("Allow") != "POST" {
		t.Errorf("GET: %d %v", rec.Code, rec.Header())
	}
}

// 초당 상한을 넘으면 버리고, 1분에 한 번 ui.dropped(WARN)를 남긴다.
func TestEventsRateLimit(t *testing.T) {
	x := newEventHarness(t, 3)
	body := `{"event":"ui.page_view","page":"/"}`
	for range 5 {
		if code := x.post(body); code != http.StatusNoContent {
			t.Fatalf("code %d", code)
		}
	}
	if n := len(x.uiLines()); n != 3 {
		t.Errorf("accepted %d, want 3", n)
	}
	// 첫 버림에서 바로 한 번 남기고, 1분 안의 다음 버림은 세기만 한다.
	if n := strings.Count(x.logs.String(), "event=ui.dropped"); n != 1 || !strings.Contains(x.logs.String(), "level=WARN comp=go event=ui.dropped req=") {
		t.Errorf("dropped lines %d: %s", n, x.logs.String())
	}

	// 다음 초에는 다시 받는다. 1분이 지나 다음 요청이 오면 그동안 버린 수(1)를 남긴다.
	x.logs.Reset()
	x.now = x.now.Add(DroppedReportEvery)
	x.post(body)
	if n := len(x.uiLines()); n != 1 {
		t.Errorf("next second accepted %d", n)
	}
	if !regexp.MustCompile(`event=ui\.dropped req=\w+ count=1 limit_per_second=3`).MatchString(x.logs.String()) {
		t.Errorf("dropped report: %s", x.logs.String())
	}
	x.logs.Reset()
	x.post(body)
	if strings.Contains(x.logs.String(), "ui.dropped") {
		t.Errorf("dropped again with nothing dropped: %s", x.logs.String())
	}
}
