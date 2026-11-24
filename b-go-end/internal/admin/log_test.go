package admin

import (
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/schware/aidev-KEDP001/b-go-end/internal/logx"
)

func lineMatches(t *testing.T, logs, pattern string) {
	t.Helper()
	if !regexp.MustCompile(`(?m)` + pattern).MatchString(logs) {
		t.Errorf("no line %q in:\n%s", pattern, logs)
	}
}

// 허용 IP 거부는 http.request 줄(403, WARN)에 접속 주소를 붙인다(D026 7장 예외).
func TestLogForbiddenIP(t *testing.T) {
	x := newHarness(t, &fakeStore{})
	x.h = logx.HTTP(x.h)
	logs := captureLog(t)
	x.do(http.MethodGet, "/api/admin/indicators", "", otherAddr, "")
	lineMatches(t, logs.String(), `level=WARN comp=go event=http\.request req=\w+ method=GET path=/api/admin/indicators status=403 ms=\d+ ip=198\.51\.100\.7$`)
}

func TestLogLoginLogout(t *testing.T) {
	x := newHarness(t, &fakeStore{})
	x.h = logx.HTTP(x.h)
	logs := captureLog(t)
	x.do(http.MethodPost, "/api/admin/login", `{"account":"admin","password":"nope nope nope"}`, allowedAddr, "")
	token := x.login()
	x.do(http.MethodPost, "/api/admin/logout", "{}", allowedAddr, token)
	x.do(http.MethodGet, "/api/admin/session", "", allowedAddr, "")
	s := logs.String()
	lineMatches(t, s, `level=WARN comp=go event=admin\.login req=\w+ result=failed ip=192\.0\.2\.10$`)
	lineMatches(t, s, `level=WARN comp=go event=http\.request req=\w+ method=POST path=/api/admin/login status=401 `)
	lineMatches(t, s, `level=INFO comp=go event=admin\.login req=\w+ result=ok account=admin ip=192\.0\.2\.10$`)
	lineMatches(t, s, `level=INFO comp=go event=admin\.logout req=\w+ account=admin$`)
	// 로그인하지 않은 상태 확인(401)은 정상 동작이라 INFO다.
	lineMatches(t, s, `level=INFO comp=go event=http\.request req=\w+ method=GET path=/api/admin/session status=401 `)
	if strings.Contains(s, "nope") || strings.Contains(s, testPassword) || strings.Contains(s, token) {
		t.Errorf("log has password or cookie: %s", s)
	}
}
