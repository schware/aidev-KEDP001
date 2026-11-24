package api

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/schware/aidev-KEDP001/b-go-end/internal/logx"
)

// 화면 동작 기록 API(specification/40.api/README.md 7장, KEDP001-D026 6장, Issue #93). 인증은 없다.
//
//	POST /api/events   {"event": "ui.card_open", "indicator": "cpi"}
//
// 받은 동작은 서버 로그에 comp=web 한 줄로만 남긴다. DB에는 쓰지 않는다.
// 응답은 언제나 204다. 받지 않은 동작(허용 목록 밖, 틀린 값, 너무 큼, 초당 상한 넘음)도 화면에 오류를 보이지 않는다.
// 접속 주소, User-Agent는 남기지 않는다.

// MaxEventBody는 본문의 최대 크기(바이트)다.
const MaxEventBody = 1 << 10

// MaxEventsPerSecond는 서버 전체에서 1초에 받는 동작 수의 기본 상한이다. 넘으면 버린다.
const MaxEventsPerSecond = 50

// DroppedReportEvery는 버린 동작 수(ui.dropped)를 남기는 간격이다.
const DroppedReportEvery = time.Minute

var (
	// pathPattern은 화면 경로와 API 경로다(query string 없음). 예: /, /indicator/cpi, /api/indicators/cpi
	pathPattern = regexp.MustCompile(`^/[A-Za-z0-9/_.-]{0,99}$`)
	// indicatorPattern은 지표 id다(관리자 지표 추가와 같은 규칙, 5.5).
	indicatorPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
)

// maxIndicatorLen은 지표 id의 최대 길이다.
const maxIndicatorLen = 64

// AdminActions는 ui.admin_action의 action으로 받는 값이다(관리자 화면의 버튼).
var AdminActions = map[string]bool{
	"login":              true,
	"logout":             true,
	"indicator_add":      true,
	"indicator_edit":     true,
	"display_order_save": true,
	"collect_time_save":  true,
	"visit_base_save":    true,
	"visit_reset":        true,
}

// field는 동작 하나가 받는 칸이다.
type field struct {
	name     string
	required bool
	check    func(json.RawMessage) (any, bool)
}

// eventFields는 받는 동작(허용 목록)과 칸이다(D026 4장). 칸은 이 차례로 로그에 나온다.
var eventFields = map[string][]field{
	"ui.page_view":    {{"page", true, pathValue}},
	"ui.card_open":    {{"indicator", true, indicatorValue}},
	"ui.related_open": {{"indicator", true, indicatorValue}, {"from", true, indicatorValue}},
	"ui.admin_action": {{"action", true, actionValue}},
	"ui.error":        {{"path", true, pathValue}, {"status", true, statusValue}},
}

// eventLevel은 동작의 level이다. 화면이 보낸 level은 쓰지 않는다.
func eventLevel(event string) slog.Level {
	if event == "ui.error" {
		return slog.LevelWarn
	}
	return slog.LevelInfo
}

func stringValue(raw json.RawMessage) (string, bool) {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", false
	}
	return s, true
}

func pathValue(raw json.RawMessage) (any, bool) {
	s, ok := stringValue(raw)
	return s, ok && pathPattern.MatchString(s)
}

func indicatorValue(raw json.RawMessage) (any, bool) {
	s, ok := stringValue(raw)
	return s, ok && len(s) <= maxIndicatorLen && indicatorPattern.MatchString(s)
}

func actionValue(raw json.RawMessage) (any, bool) {
	s, ok := stringValue(raw)
	return s, ok && AdminActions[s]
}

// statusValue는 API 응답 코드다. 0은 응답을 받지 못함(연결 실패)이다.
func statusValue(raw json.RawMessage) (any, bool) {
	var n int
	if err := json.Unmarshal(raw, &n); err != nil {
		return 0, false
	}
	return n, n == 0 || (n >= 100 && n <= 599)
}

// EventConfig는 화면 동작 기록 API 설정이다.
type EventConfig struct {
	// Now는 지금 시각이다. 비어 있으면 time.Now다.
	Now func() time.Time
	// PerSecond는 1초에 받는 동작 수의 상한이다. 0이면 MaxEventsPerSecond다.
	PerSecond int
}

// EventHandler는 POST /api/events handler다.
func EventHandler(c EventConfig) http.Handler {
	if c.Now == nil {
		c.Now = time.Now
	}
	if c.PerSecond <= 0 {
		c.PerSecond = MaxEventsPerSecond
	}
	return &events{EventConfig: c}
}

type events struct {
	EventConfig

	mu         sync.Mutex
	second     int64 // 지금 세고 있는 초(Unix)
	count      int   // 그 초에 받은 수
	dropped    int   // 마지막 ui.dropped 뒤로 버린 수
	lastReport time.Time
}

func (e *events) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		write(w, http.StatusMethodNotAllowed, Error{"method not allowed"})
		return
	}
	defer w.WriteHeader(http.StatusNoContent)
	if !e.allow(r) {
		return
	}
	event, args, reason := readEvent(w, r)
	if reason != "" {
		// 받지 않은 이유는 개발할 때만 본다. 받은 값은 남기지 않는다.
		logx.Debug(r.Context(), "ui.rejected", "reason", reason)
		return
	}
	logx.LogComp(r.Context(), logx.CompWeb, eventLevel(event), event, args...)
}

// allow는 초당 상한 안이면 true다. 넘으면 버린 수를 세고,
// 마지막으로 남긴 지 DroppedReportEvery가 지났으면 ui.dropped(WARN)를 남긴다.
// 버린 수는 다음 요청이 올 때 남기므로, 요청이 끊기면 다음 요청까지 남지 않는다.
func (e *events) allow(r *http.Request) bool {
	now := e.Now()
	e.mu.Lock()
	defer e.mu.Unlock()
	if s := now.Unix(); s != e.second {
		e.second, e.count = s, 0
	}
	ok := e.count < e.PerSecond
	if ok {
		e.count++
	} else {
		e.dropped++
	}
	if e.dropped > 0 && now.Sub(e.lastReport) >= DroppedReportEvery {
		logx.Warn(r.Context(), "ui.dropped", "count", e.dropped, "limit_per_second", e.PerSecond)
		e.dropped, e.lastReport = 0, now
	}
	return ok
}

// readEvent는 본문을 읽어 동작 이름과 로그에 붙일 칸을 돌려준다. 받지 않으면 reason이 있다.
// 모르는 칸은 버리고, 아는 칸의 값이 틀리거나 꼭 있어야 할 칸이 없으면 동작 전체를 받지 않는다.
func readEvent(w http.ResponseWriter, r *http.Request) (event string, args []any, reason string) {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, MaxEventBody))
	var body map[string]json.RawMessage
	if err := dec.Decode(&body); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			return "", nil, "too large"
		}
		return "", nil, "invalid JSON"
	}
	if _, err := dec.Token(); err != io.EOF {
		return "", nil, "invalid JSON"
	}
	raw, ok := body["event"]
	if !ok {
		return "", nil, "no event"
	}
	event, ok = stringValue(raw)
	fields, known := eventFields[event]
	if !ok || !known {
		return "", nil, "unknown event"
	}
	for _, f := range fields {
		v, has := body[f.name]
		if !has {
			if f.required {
				return "", nil, "missing " + f.name
			}
			continue
		}
		value, ok := f.check(v)
		if !ok {
			return "", nil, "invalid " + f.name
		}
		args = append(args, f.name, value)
	}
	return event, args, ""
}
