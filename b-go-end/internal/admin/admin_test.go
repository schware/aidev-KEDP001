package admin

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/schware/aidev-KEDP001/b-go-end/internal/store"
)

var seoul = time.FixedZone("KST", 9*60*60)

// fakeStore는 받은 값을 남기는 가짜 DB다.
type fakeStore struct {
	list        []store.AdminIndicator
	err         error
	added       *store.NewIndicator
	edited      *store.Edit
	order       []store.OrderItem
	collectTime string
	runs        []store.SourceRun
}

func (f *fakeStore) AdminIndicators(context.Context) ([]store.AdminIndicator, error) {
	return f.list, f.err
}

func (f *fakeStore) AdminIndicator(_ context.Context, id string) (store.AdminIndicator, error) {
	for _, a := range f.list {
		if a.ID == id {
			return a, nil
		}
	}
	if f.added != nil && f.added.ID == id {
		return store.AdminIndicator{ID: id, Name: f.added.Name}, nil
	}
	return store.AdminIndicator{}, store.ErrNotFound
}

func (f *fakeStore) AddIndicator(_ context.Context, n store.NewIndicator) error {
	if f.err != nil {
		return f.err
	}
	f.added = &n
	return nil
}

func (f *fakeStore) UpdateIndicator(_ context.Context, _ string, e store.Edit) error {
	f.edited = &e
	return f.err
}

func (f *fakeStore) SaveOrder(_ context.Context, items []store.OrderItem) error {
	f.order = items
	return f.err
}

func (f *fakeStore) CollectTime(context.Context) (string, error) { return f.collectTime, f.err }

func (f *fakeStore) SetCollectTime(_ context.Context, v string) (string, error) {
	old := f.collectTime
	f.collectTime = v
	return old, f.err
}

func (f *fakeStore) SourceRuns(context.Context) ([]store.SourceRun, error) { return f.runs, f.err }

const (
	testAccount  = "admin"
	testPassword = "correct horse battery"
	allowedAddr  = "192.0.2.10:50000"
	otherAddr    = "198.51.100.7:50000"
)

func testAccounts(t *testing.T) Accounts {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	return Accounts{testAccount: h}
}

type harness struct {
	t   *testing.T
	h   http.Handler
	st  *fakeStore
	now time.Time
}

func newHarness(t *testing.T, st *fakeStore) *harness {
	x := &harness{t: t, st: st, now: time.Date(2026, 10, 30, 1, 0, 0, 0, time.UTC)}
	x.h = Handler(Config{
		Store:    st,
		AllowIPs: []netip.Prefix{netip.MustParsePrefix("192.0.2.0/24")},
		Accounts: testAccounts(t),
		KeySet:   map[string]bool{"ecos": true, "kosis": true},
		Now:      func() time.Time { return x.now },
		Loc:      seoul,
	})
	return x
}

type response struct {
	code   int
	body   string
	header http.Header
}

// do는 요청을 보낸다. body가 있으면 Content-Type: application/json이다.
func (x *harness) do(method, path, body, remote, cookie string, header ...string) response {
	x.t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, r)
	req.RemoteAddr = remote
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for i := 0; i+1 < len(header); i += 2 {
		req.Header.Set(header[i], header[i+1])
	}
	if cookie != "" {
		req.AddCookie(&http.Cookie{Name: CookieName, Value: cookie})
	}
	rec := httptest.NewRecorder()
	x.h.ServeHTTP(rec, req)
	if ct := rec.Header().Get("Content-Type"); rec.Code != http.StatusNoContent && ct != "application/json; charset=utf-8" {
		x.t.Errorf("%s %s: Content-Type = %q", method, path, ct)
	}
	return response{rec.Code, strings.TrimSpace(rec.Body.String()), rec.Header()}
}

// login은 로그인하고 cookie 값을 돌려준다.
func (x *harness) login() string {
	x.t.Helper()
	res := x.do(http.MethodPost, "/api/admin/login", `{"account":"admin","password":"`+testPassword+`"}`, allowedAddr, "")
	if res.code != http.StatusOK {
		x.t.Fatalf("login: %d %s", res.code, res.body)
	}
	c := cookie(x.t, res)
	return c.Value
}

func cookie(t *testing.T, res response) *http.Cookie {
	t.Helper()
	cs := (&http.Response{Header: res.header}).Cookies()
	if len(cs) != 1 {
		t.Fatalf("cookies = %v", cs)
	}
	return cs[0]
}

// protected는 로그인해야 쓰는 요청이다.
var protected = []struct{ method, path, body string }{
	{http.MethodPost, "/api/admin/logout", "{}"},
	{http.MethodGet, "/api/admin/session", ""},
	{http.MethodGet, "/api/admin/indicators", ""},
	{http.MethodPost, "/api/admin/indicators", "{}"},
	{http.MethodPut, "/api/admin/indicators/x", "{}"},
	{http.MethodPut, "/api/admin/display-order", "{}"},
	{http.MethodGet, "/api/admin/collect-time", ""},
	{http.MethodPut, "/api/admin/collect-time", "{}"},
	{http.MethodGet, "/api/admin/sources", ""},
}

func TestNotAllowedIP(t *testing.T) {
	x := newHarness(t, &fakeStore{})
	token := x.login()
	all := append(protected, struct{ method, path, body string }{http.MethodPost, "/api/admin/login",
		`{"account":"admin","password":"` + testPassword + `"}`},
		struct{ method, path, body string }{http.MethodGet, "/api/admin/none", ""})
	for _, c := range all {
		// 로그인한 cookie가 있어도 허용 IP가 아니면 403이다.
		res := x.do(c.method, c.path, c.body, otherAddr, token)
		if res.code != http.StatusForbidden || res.body != `{"error":"forbidden"}` {
			t.Errorf("%s %s: got %d %s", c.method, c.path, res.code, res.body)
		}
	}
}

func TestEmptyAllowIPs(t *testing.T) {
	h := Handler(Config{Store: &fakeStore{}, Accounts: testAccounts(t), Loc: seoul})
	for _, remote := range []string{allowedAddr, "127.0.0.1:1234"} {
		req := httptest.NewRequest(http.MethodPost, "/api/admin/login",
			strings.NewReader(`{"account":"admin","password":"`+testPassword+`"}`))
		req.RemoteAddr = remote
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s: got %d", remote, rec.Code)
		}
	}
}

func TestNotLoggedIn(t *testing.T) {
	x := newHarness(t, &fakeStore{})
	for _, cookie := range []string{"", "unknown-token"} {
		for _, c := range protected {
			res := x.do(c.method, c.path, c.body, allowedAddr, cookie)
			if res.code != http.StatusUnauthorized || res.body != `{"error":"login required"}` {
				t.Errorf("%s %s cookie %q: got %d %s", c.method, c.path, cookie, res.code, res.body)
			}
		}
	}
}

func TestLoggedIn(t *testing.T) {
	st := &fakeStore{
		list:        []store.AdminIndicator{{ID: "a", Name: "A", Source: "ecos", Cycle: "M"}},
		collectTime: "06:00",
	}
	x := newHarness(t, st)
	token := x.login()
	for _, c := range []struct{ method, path string }{
		{http.MethodGet, "/api/admin/session"},
		{http.MethodGet, "/api/admin/indicators"},
		{http.MethodGet, "/api/admin/collect-time"},
		{http.MethodGet, "/api/admin/sources"},
	} {
		res := x.do(c.method, c.path, "", allowedAddr, token)
		if res.code != http.StatusOK {
			t.Errorf("%s %s: got %d %s", c.method, c.path, res.code, res.body)
		}
	}
	res := x.do(http.MethodGet, "/api/admin/session", "", allowedAddr, token)
	if want := `{"account":"admin","expires_at":"2026-10-30T18:00:00+09:00"}`; res.body != want {
		t.Errorf("session = %s, want %s", res.body, want)
	}
}

func TestLoginCookie(t *testing.T) {
	x := newHarness(t, &fakeStore{})
	res := x.do(http.MethodPost, "/api/admin/login", `{"account":"admin","password":"`+testPassword+`"}`, allowedAddr, "")
	c := cookie(t, res)
	if len(c.Value) != 43 || !c.HttpOnly || c.SameSite != http.SameSiteStrictMode || c.Path != "/api/admin" ||
		c.MaxAge != 8*60*60 || c.Secure {
		t.Errorf("cookie = %+v", c)
	}
	if res.body != `{"account":"admin","expires_at":"2026-10-30T18:00:00+09:00"}` {
		t.Errorf("body = %s", res.body)
	}
	if res.header.Get("Cache-Control") != "no-store" {
		t.Errorf("Cache-Control = %q", res.header.Get("Cache-Control"))
	}
}

func TestLoginFailedSameResponse(t *testing.T) {
	x := newHarness(t, &fakeStore{})
	var logs bytes.Buffer
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	for _, body := range []string{
		`{"account":"admin","password":"wrong password!!"}`,
		`{"account":"nobody","password":"` + testPassword + `"}`,
		`{"account":"","password":""}`,
	} {
		res := x.do(http.MethodPost, "/api/admin/login", body, allowedAddr, "")
		if res.code != http.StatusUnauthorized || res.body != `{"error":"login failed"}` || res.header.Get("Set-Cookie") != "" {
			t.Errorf("%s: got %d %s %v", body, res.code, res.body, res.header)
		}
	}
	if s := logs.String(); strings.Contains(s, "wrong password") || strings.Contains(s, "nobody") ||
		strings.Contains(s, testPassword) {
		t.Errorf("log has account or password: %s", s)
	}
}

func TestSessionExpires(t *testing.T) {
	x := newHarness(t, &fakeStore{})
	token := x.login()
	x.now = x.now.Add(SessionTTL - time.Second)
	if res := x.do(http.MethodGet, "/api/admin/session", "", allowedAddr, token); res.code != http.StatusOK {
		t.Errorf("before expiry: %d", res.code)
	}
	x.now = x.now.Add(time.Second)
	if res := x.do(http.MethodGet, "/api/admin/session", "", allowedAddr, token); res.code != http.StatusUnauthorized {
		t.Errorf("after expiry: %d", res.code)
	}
}

func TestLogout(t *testing.T) {
	x := newHarness(t, &fakeStore{})
	token := x.login()
	res := x.do(http.MethodPost, "/api/admin/logout", "{}", allowedAddr, token)
	if res.code != http.StatusNoContent || cookie(t, res).MaxAge != -1 {
		t.Errorf("logout: %d %v", res.code, res.header)
	}
	if res := x.do(http.MethodGet, "/api/admin/session", "", allowedAddr, token); res.code != http.StatusUnauthorized {
		t.Errorf("after logout: %d", res.code)
	}
}

func TestContentType(t *testing.T) {
	x := newHarness(t, &fakeStore{})
	token := x.login()
	for _, ct := range []string{"", "text/plain", "application/x-www-form-urlencoded", "multipart/form-data"} {
		res := x.do(http.MethodPut, "/api/admin/collect-time", "", allowedAddr, token, "Content-Type", ct)
		if res.code != http.StatusUnsupportedMediaType {
			t.Errorf("%q: got %d %s", ct, res.code, res.body)
		}
	}
	res := x.do(http.MethodPut, "/api/admin/collect-time", `{"collect_time":"07:30"}`, allowedAddr, token,
		"Content-Type", "application/json; charset=utf-8")
	if res.code != http.StatusOK {
		t.Errorf("charset: got %d %s", res.code, res.body)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	x := newHarness(t, &fakeStore{})
	token := x.login()
	res := x.do(http.MethodDelete, "/api/admin/indicators/x", "", allowedAddr, token)
	if res.code != http.StatusMethodNotAllowed || res.header.Get("Allow") != "PUT" {
		t.Errorf("got %d %v", res.code, res.header)
	}
	res = x.do(http.MethodGet, "/api/admin/login", "", allowedAddr, "")
	if res.code != http.StatusMethodNotAllowed || res.header.Get("Allow") != "POST" {
		t.Errorf("got %d %v", res.code, res.header)
	}
}

func TestForwardedFor(t *testing.T) {
	x := newHarness(t, &fakeStore{})
	body := `{"account":"admin","password":"` + testPassword + `"}`
	for _, c := range []struct {
		remote, xff string
		want        int
	}{
		// Vite proxy(127.0.0.1)가 넘긴 허용 IP
		{"127.0.0.1:40000", "192.0.2.20", http.StatusOK},
		// 마지막 주소만 본다. 앞의 주소는 요청한 쪽이 꾸밀 수 있다.
		{"127.0.0.1:40000", "192.0.2.20, 198.51.100.7", http.StatusForbidden},
		{"127.0.0.1:40000", "198.51.100.7, 192.0.2.20", http.StatusOK},
		{"127.0.0.1:40000", "not-an-ip", http.StatusForbidden},
		{"127.0.0.1:40000", "192.0.2.20,", http.StatusForbidden},
		// proxy가 아니면 X-Forwarded-For를 보지 않는다.
		{otherAddr, "192.0.2.20", http.StatusForbidden},
		{allowedAddr, "198.51.100.7", http.StatusOK},
	} {
		res := x.do(http.MethodPost, "/api/admin/login", body, c.remote, "", "X-Forwarded-For", c.xff)
		if res.code != c.want {
			t.Errorf("%s %q: got %d, want %d", c.remote, c.xff, res.code, c.want)
		}
	}
}

func TestClientIP(t *testing.T) {
	for _, c := range []struct {
		remote string
		xff    []string
		want   string
	}{
		{"192.0.2.1:1", nil, "192.0.2.1"},
		{"[::ffff:192.0.2.1]:1", nil, "192.0.2.1"},
		{"[2001:db8::1]:1", []string{"192.0.2.9"}, "2001:db8::1"},
		{"127.0.0.1:1", nil, "127.0.0.1"},
		{"127.0.0.1:1", []string{"192.0.2.8", "192.0.2.9"}, "192.0.2.9"},
		{"127.0.0.1:1", []string{" ::ffff:192.0.2.7 "}, "192.0.2.7"},
		{"bad", nil, ""},
	} {
		got, ok := ClientIP(c.remote, c.xff, []netip.Prefix{netip.MustParsePrefix("127.0.0.1/32")})
		if (c.want == "" && ok) || (c.want != "" && got.String() != c.want) {
			t.Errorf("%s %v: got %v %t, want %s", c.remote, c.xff, got, ok, c.want)
		}
	}
}

// TestClientIPTrusted는 믿을 proxy(KEDP_TRUSTED_PROXIES)에서 온 요청만 X-Forwarded-For를 쓰는지 본다.
func TestClientIPTrusted(t *testing.T) {
	// compose의 nginx container가 있는 Docker 내부 network 같은 곳
	trusted := []netip.Prefix{netip.MustParsePrefix("10.89.0.0/24"), netip.MustParsePrefix("192.0.2.53/32")}
	for _, c := range []struct {
		remote string
		xff    []string
		want   string
	}{
		{"10.89.0.5:1", []string{"203.0.113.9"}, "203.0.113.9"},
		{"192.0.2.53:1", []string{"198.51.100.1, 203.0.113.9"}, "203.0.113.9"},
		{"10.89.0.5:1", nil, "10.89.0.5"},
		// 믿을 proxy가 아니면 X-Forwarded-For를 보지 않는다. 127.0.0.1도 목록에 없으면 믿지 않는다.
		{"10.89.1.5:1", []string{"203.0.113.9"}, "10.89.1.5"},
		{"127.0.0.1:1", []string{"203.0.113.9"}, "127.0.0.1"},
		{"10.89.0.5:1", []string{"bad"}, ""},
	} {
		got, ok := ClientIP(c.remote, c.xff, trusted)
		if (c.want == "" && ok) || (c.want != "" && got.String() != c.want) {
			t.Errorf("%s %v: got %v %t, want %s", c.remote, c.xff, got, ok, c.want)
		}
	}
	if _, ok := ClientIP("10.89.0.5:1", []string{"203.0.113.9"}, nil); !ok {
		t.Error("nil trusted: want RemoteAddr")
	}
}

// TestTrustedProxiesConfig는 Config.TrustedProxies로 허용 IP 검사가 바뀌는지 본다.
func TestTrustedProxiesConfig(t *testing.T) {
	x := newHarness(t, &fakeStore{})
	x.h = Handler(Config{
		Store:          x.st,
		AllowIPs:       []netip.Prefix{netip.MustParsePrefix("192.0.2.0/24")},
		TrustedProxies: []netip.Prefix{netip.MustParsePrefix("10.89.0.0/24")},
		Accounts:       testAccounts(t),
		Now:            func() time.Time { return x.now },
		Loc:            seoul,
	})
	for _, c := range []struct {
		remote, xff string
		want        int
	}{
		// nginx(믿을 proxy)가 실제 접속 주소를 넘긴다.
		{"10.89.0.5:40000", "192.0.2.20", http.StatusUnauthorized},
		{"10.89.0.5:40000", "198.51.100.7", http.StatusForbidden},
		// 믿지 않는 곳의 X-Forwarded-For는 보지 않는다(127.0.0.1도 목록에 없으면 같다).
		{"127.0.0.1:40000", "192.0.2.20", http.StatusForbidden},
		{"198.51.100.7:40000", "192.0.2.20", http.StatusForbidden},
		{allowedAddr, "198.51.100.7", http.StatusUnauthorized},
	} {
		res := x.do(http.MethodGet, "/api/admin/session", "", c.remote, "", "X-Forwarded-For", c.xff)
		if res.code != c.want {
			t.Errorf("%s %q: got %d, want %d", c.remote, c.xff, res.code, c.want)
		}
	}
}

func TestAddIndicator(t *testing.T) {
	st := &fakeStore{}
	x := newHarness(t, st)
	token := x.login()
	res := x.do(http.MethodPost, "/api/admin/indicators", `{
		"id":"kosis-kospi","name":" 코스피 ","source":"kosis","stat_code":"DT_1","item_code":"T1","classes":["00"],
		"org_id":"101","cycle":"M","unit":"","visible":true,"chart_period":"10 years","decimals":2,
		"notice":null,"related":["ecos-base-rate"]}`, allowedAddr, token)
	if res.code != http.StatusCreated {
		t.Fatalf("got %d %s", res.code, res.body)
	}
	want := store.NewIndicator{ID: "kosis-kospi", Name: "코스피", Source: "kosis", StatCode: "DT_1", Cycle: "M",
		OrgID: "101", ItemCode: "T1", Classes: []string{"00"}, Visible: true, ChartPeriod: "10 years",
		Decimals: ptr(2), Related: []string{"ecos-base-rate"}}
	if !reflect.DeepEqual(*st.added, want) {
		t.Errorf("added %+v\nwant  %+v", *st.added, want)
	}
}

func TestAddIndicatorInvalid(t *testing.T) {
	x := newHarness(t, &fakeStore{})
	token := x.login()
	for _, c := range []struct{ body, want string }{
		{`{}`, `{"error":"invalid input","fields":{"chart_period":"required","cycle":"required","id":"required",` +
			`"name":"required","source":"required","stat_code":"required"}}`},
		{`{"id":"Bad_ID","name":"x","source":"ecos","stat_code":"S","cycle":"Y","chart_period":"10 year","decimals":5,` +
			`"related":["Bad_ID"]}`,
			`{"error":"invalid input","fields":{"chart_period":"must be \"N years\", \"N months\" or \"N days\" (N 1-9999)",` +
				`"cycle":"must be one of D, M, Q, A","decimals":"must be 0-4 or null",` +
				`"id":"must be lowercase letters and digits joined by '-' (e.g. kosis-kospi)",` +
				`"item_code":"required for ecos","related":"must not include the indicator itself"}}`},
		{`{"id":"k","name":"x","source":"kosis","stat_code":"S","cycle":"M","chart_period":"1 years"}`,
			`{"error":"invalid input","fields":{"item_code":"required for kosis","org_id":"required for kosis"}}`},
		{`{"id":"d","name":"x","source":"data-go-kr","stat_code":"069500","item_code":"x","cycle":"D","chart_period":"30 days",` +
			`"related":["a","a"],"classes":[" "]}`,
			`{"error":"invalid input","fields":{"classes":"must not contain empty codes",` +
				`"item_code":"must be empty for data-go-kr","related":"indicator \"a\" is listed twice",` +
				`"value_field":"required for data-go-kr"}}`},
		{`{"decimals":2.5}`, `{"error":"invalid input","fields":{"decimals":"must be an integer"}}`},
		{`{"related":"a"}`, `{"error":"invalid input","fields":{"related":"must be an array of strings"}}`},
		{`{"sort_order":1}`, `{"error":"invalid input","fields":{"sort_order":"unknown field"}}`},
		{`{"id":"a"} {}`, `{"error":"invalid JSON"}`},
		{`{`, `{"error":"invalid JSON"}`},
	} {
		res := x.do(http.MethodPost, "/api/admin/indicators", c.body, allowedAddr, token)
		if res.code != http.StatusBadRequest || res.body != c.want {
			t.Errorf("%s:\ngot  %d %s\nwant %s", c.body, res.code, res.body, c.want)
		}
	}
}

func TestAddIndicatorStoreErrors(t *testing.T) {
	body := `{"id":"e","name":"x","source":"ecos","stat_code":"S","item_code":"I","cycle":"M","chart_period":"5 years","related":["none"]}`
	for _, c := range []struct {
		err  error
		code int
		want string
	}{
		{store.ErrExists, http.StatusConflict, `{"error":"indicator already exists"}`},
		{&store.FieldError{Field: "related", Reason: `indicator "none" does not exist`}, http.StatusBadRequest,
			`{"error":"invalid input","fields":{"related":"indicator \"none\" does not exist"}}`},
		{errors.New("secret detail"), http.StatusInternalServerError, `{"error":"internal error"}`},
	} {
		x := newHarness(t, &fakeStore{err: c.err})
		token := x.login()
		res := x.do(http.MethodPost, "/api/admin/indicators", body, allowedAddr, token)
		if res.code != c.code || res.body != c.want {
			t.Errorf("%v: got %d %s", c.err, res.code, res.body)
		}
	}
}

func TestEditIndicator(t *testing.T) {
	st := &fakeStore{list: []store.AdminIndicator{{ID: "a", Name: "A", Cycle: "M"}}}
	x := newHarness(t, st)
	token := x.login()
	res := x.do(http.MethodPut, "/api/admin/indicators/a",
		`{"name":"A2","cycle":"Q","chart_period":"24 months","notice":"  ","related":[]}`, allowedAddr, token)
	if res.code != http.StatusOK {
		t.Fatalf("got %d %s", res.code, res.body)
	}
	want := store.Edit{Name: "A2", Cycle: "Q", ChartPeriod: "24 months", Related: []string{}}
	if !reflect.DeepEqual(*st.edited, want) {
		t.Errorf("edited %+v, want %+v", *st.edited, want)
	}
	// id, 제공처, 코드는 바꾸지 않는다. related는 꼭 보낸다.
	for _, c := range []struct{ body, want string }{
		{`{"name":"A","cycle":"M","chart_period":"1 years","related":[],"source":"kosis"}`,
			`{"error":"invalid input","fields":{"source":"unknown field"}}`},
		{`{"name":"A","cycle":"M","chart_period":"1 years","related":[],"id":"b"}`,
			`{"error":"invalid input","fields":{"id":"unknown field"}}`},
		{`{"name":"A","cycle":"M","chart_period":"1 years"}`,
			`{"error":"invalid input","fields":{"related":"required (use [] for none)"}}`},
		{`{"name":"A","cycle":"M","chart_period":"1 years","related":["a"]}`,
			`{"error":"invalid input","fields":{"related":"must not include the indicator itself"}}`},
	} {
		res := x.do(http.MethodPut, "/api/admin/indicators/a", c.body, allowedAddr, token)
		if res.code != http.StatusBadRequest || res.body != c.want {
			t.Errorf("%s:\ngot  %d %s\nwant %s", c.body, res.code, res.body, c.want)
		}
	}
	res = x.do(http.MethodPut, "/api/admin/indicators/none",
		`{"name":"A","cycle":"M","chart_period":"1 years","related":[]}`, allowedAddr, token)
	if res.code != http.StatusNotFound || res.body != `{"error":"indicator not found"}` {
		t.Errorf("not found: got %d %s", res.code, res.body)
	}
}

func TestSaveOrder(t *testing.T) {
	st := &fakeStore{list: []store.AdminIndicator{{ID: "a"}, {ID: "b"}}}
	x := newHarness(t, st)
	token := x.login()
	res := x.do(http.MethodPut, "/api/admin/display-order",
		`{"indicators":[{"id":"b","visible":true,"order":1},{"id":"a","visible":false,"order":2}]}`, allowedAddr, token)
	if res.code != http.StatusOK || !strings.HasPrefix(res.body, `{"indicators":[`) {
		t.Fatalf("got %d %s", res.code, res.body)
	}
	want := []store.OrderItem{{ID: "b", Visible: true, Order: 1}, {ID: "a", Visible: false, Order: 2}}
	if !reflect.DeepEqual(st.order, want) {
		t.Errorf("order %+v", st.order)
	}
	for _, c := range []struct{ body, want string }{
		{`{}`, `{"error":"invalid input","fields":{"indicators":"required"}}`},
		{`{"indicators":[{"id":"a","visible":true}]}`,
			`{"error":"invalid input","fields":{"indicators":"item 0 needs id, visible and order"}}`},
		{`{"indicators":[{"id":"a","visible":true,"order":1},{"id":"a","visible":true,"order":2}]}`,
			`{"error":"invalid input","fields":{"indicators":"indicator \"a\" is listed twice"}}`},
	} {
		res := x.do(http.MethodPut, "/api/admin/display-order", c.body, allowedAddr, token)
		if res.code != http.StatusBadRequest || res.body != c.want {
			t.Errorf("%s:\ngot  %d %s\nwant %s", c.body, res.code, res.body, c.want)
		}
	}
}

func TestCollectTime(t *testing.T) {
	st := &fakeStore{collectTime: "06:00"}
	x := newHarness(t, st)
	token := x.login()
	res := x.do(http.MethodPut, "/api/admin/collect-time", `{"collect_time":"23:59"}`, allowedAddr, token)
	if res.code != http.StatusOK || res.body != `{"collect_time":"23:59"}` || st.collectTime != "23:59" {
		t.Errorf("got %d %s", res.code, res.body)
	}
	for _, v := range []string{`"24:00"`, `"6:00"`, `"06:60"`, `"0600"`, `null`, `600`} {
		res := x.do(http.MethodPut, "/api/admin/collect-time", `{"collect_time":`+v+`}`, allowedAddr, token)
		if res.code != http.StatusBadRequest {
			t.Errorf("%s: got %d %s", v, res.code, res.body)
		}
	}
	if st.collectTime != "23:59" {
		t.Errorf("collect_time changed to %s", st.collectTime)
	}
}

func TestSources(t *testing.T) {
	fin := time.Date(2026, 10, 24, 21, 0, 3, 0, time.UTC)
	st := &fakeStore{runs: []store.SourceRun{
		{Code: "data-go-kr", Name: "공공데이터포털", Status: "success", FinishedAt: &fin},
		{Code: "ecos", Name: "ECOS", Status: "error", FinishedAt: &fin},
		{Code: "kosis", Name: "KOSIS"},
	}}
	x := newHarness(t, st)
	token := x.login()
	res := x.do(http.MethodGet, "/api/admin/sources", "", allowedAddr, token)
	want := `{"sources":[` +
		`{"code":"data-go-kr","name":"공공데이터포털","key_set":false,"last_collected_at":"2026-10-25T06:00:03+09:00","status":"no_key"},` +
		`{"code":"ecos","name":"ECOS","key_set":true,"last_collected_at":"2026-10-25T06:00:03+09:00","status":"error"},` +
		`{"code":"kosis","name":"KOSIS","key_set":true,"last_collected_at":null,"status":"unchecked"}]}`
	if res.code != http.StatusOK || res.body != want {
		t.Errorf("got %d %s\nwant %s", res.code, res.body, want)
	}
}

func TestSourceStatus(t *testing.T) {
	for _, c := range []struct {
		key  bool
		run  string
		want string
	}{
		{true, "success", StatusConnected},
		{true, "no_data", StatusConnected},
		{true, "error", StatusError},
		{true, "", StatusUnchecked},
		{false, "success", StatusNoKey},
		{false, "", StatusNoKey},
	} {
		if got := sourceStatus(c.key, c.run); got != c.want {
			t.Errorf("%t %q: got %s, want %s", c.key, c.run, got, c.want)
		}
	}
}

func TestLoadAccounts(t *testing.T) {
	h, _ := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.MinCost)
	dir := t.TempDir()
	write := func(content string) string {
		p := filepath.Join(dir, "admin")
		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	a, err := LoadAccounts(write("# comment\n\nadmin:" + string(h) + "\n"))
	if err != nil || len(a) != 1 || !a.verify("admin", testPassword) || a.verify("admin", "wrong") {
		t.Errorf("got %v %v", a, err)
	}
	for _, content := range []string{
		"admin:" + testPassword + "\n", // hash가 아니다
		"admin " + string(h) + "\n",    // ':'가 없다
		"ad min:" + string(h) + "\n",   // 계정 이름
		"admin:" + string(h) + "\nadmin:" + string(h) + "\n",
	} {
		_, err := LoadAccounts(write(content))
		if err == nil || strings.Contains(err.Error(), testPassword) || strings.Contains(err.Error(), dir) {
			t.Errorf("%q: err = %v", content, err)
		}
	}
	if _, err := LoadAccounts(filepath.Join(dir, "none")); err == nil || strings.Contains(err.Error(), dir) {
		t.Errorf("missing file: %v", err)
	}
}

func TestHashLine(t *testing.T) {
	line, err := HashLine("admin", testPassword)
	if err != nil {
		t.Fatal(err)
	}
	name, hash, _ := strings.Cut(line, ":")
	if name != "admin" || strings.Contains(line, testPassword) ||
		bcrypt.CompareHashAndPassword([]byte(hash), []byte(testPassword)) != nil {
		t.Errorf("line = %s", line)
	}
	if cost, _ := bcrypt.Cost([]byte(hash)); cost != HashCost {
		t.Errorf("cost = %d", cost)
	}
	for _, c := range []struct{ account, pw string }{
		{"admin", "short pass1"},           // 11자
		{"admin", strings.Repeat("a", 73)}, // 72바이트 넘음
		{"bad name", testPassword},
		{"", testPassword},
	} {
		if _, err := HashLine(c.account, c.pw); err == nil {
			t.Errorf("%q %q: no error", c.account, c.pw)
		}
	}
	// 한글 12자(36바이트)는 된다.
	if _, err := HashLine("admin", strings.Repeat("가", 12)); err != nil {
		t.Errorf("korean: %v", err)
	}
}

func ptr[T any](v T) *T { return &v }
