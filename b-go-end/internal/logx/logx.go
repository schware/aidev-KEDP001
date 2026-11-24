// Package logx는 KEDP 로그 형식(KEDP001-D026)으로 한 줄씩 남긴다.
//
//	time=2026-11-24T06:00:01.123+09:00 level=INFO comp=go event=http.request req=4f1c… method=GET path=/api/indicators status=200 ms=12
//
// 모든 줄에 time(Asia/Seoul, 밀리초), level(DEBUG, INFO, WARN, ERROR), comp=go, event가 있다.
// 표준 라이브러리 log/slog의 TextHandler를 쓰므로 공백이 있는 값은 큰따옴표로 감싼다.
// ctx에 With로 붙인 값(req, run, indicator, page 등)은 event 바로 뒤에 나온다.
//
// 키, 비밀번호, cookie, 요청·응답 본문, 일반 사용자의 접속 주소와 User-Agent는 넘기지 않는다(D026 7장).
package logx

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync/atomic"
	"time"
	_ "time/tzdata" // 서버에 시간대 정보가 없어도 Asia/Seoul을 읽는다
)

// Comp는 이 서버의 comp 값이다.
const Comp = "go"

// timeFormat은 RFC 3339에 밀리초를 붙인 꼴이다.
const timeFormat = "2006-01-02T15:04:05.000Z07:00"

var seoul = mustLoadLocation("Asia/Seoul")

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

var (
	level  = new(slog.LevelVar) // 기본 INFO
	logger atomic.Pointer[slog.Logger]
)

func init() { logger.Store(newLogger(os.Stderr)) }

func newLogger(w io.Writer) *slog.Logger {
	h := slog.NewTextHandler(w, &slog.HandlerOptions{Level: level, ReplaceAttr: replace})
	return slog.New(h.WithAttrs([]slog.Attr{slog.String("comp", Comp)}))
}

// replace는 시각을 Asia/Seoul 밀리초로 바꾸고, 비어 있는 msg를 뺀다(event를 attr로 남기므로 msg는 쓰지 않는다).
func replace(groups []string, a slog.Attr) slog.Attr {
	if len(groups) > 0 {
		return a
	}
	switch a.Key {
	case slog.TimeKey:
		if t, ok := a.Value.Any().(time.Time); ok {
			return slog.String(slog.TimeKey, t.In(seoul).Format(timeFormat))
		}
	case slog.MessageKey:
		if a.Value.String() == "" {
			return slog.Attr{}
		}
	}
	return a
}

// Setup은 w에 lv 이상만 남기게 한다. 서버를 시작할 때 한 번 부른다.
func Setup(w io.Writer, lv slog.Level) {
	level.Set(lv)
	logger.Store(newLogger(w))
}

// SetOutput은 시험용이다. w에 lv 이상을 남기게 하고, 되돌리는 함수를 돌려준다.
func SetOutput(w io.Writer, lv slog.Level) (restore func()) {
	oldLevel, oldLogger := level.Level(), logger.Load()
	Setup(w, lv)
	return func() {
		level.Set(oldLevel)
		logger.Store(oldLogger)
	}
}

// Level은 지금 남기는 가장 낮은 level이다.
func Level() slog.Level { return level.Level() }

// ParseLevel은 KEDP_LOG_LEVEL 값을 읽는다. 비어 있으면 INFO다.
// DEBUG, INFO, WARN, ERROR가 아니면 INFO와 false다(D026 3장).
func ParseLevel(s string) (slog.Level, bool) {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "":
		return slog.LevelInfo, true
	case "DEBUG":
		return slog.LevelDebug, true
	case "INFO":
		return slog.LevelInfo, true
	case "WARN":
		return slog.LevelWarn, true
	case "ERROR":
		return slog.LevelError, true
	}
	return slog.LevelInfo, false
}

type attrsKey struct{}

// With는 이 ctx로 남기는 모든 줄에 args(key, value 쌍)를 붙이는 ctx를 돌려준다.
func With(ctx context.Context, args ...any) context.Context {
	prev, _ := ctx.Value(attrsKey{}).([]any)
	next := make([]any, 0, len(prev)+len(args))
	next = append(append(next, prev...), args...)
	return context.WithValue(ctx, attrsKey{}, next)
}

// Log는 event 한 줄을 lv로 남긴다. args는 key, value 쌍이다.
func Log(ctx context.Context, lv slog.Level, event string, args ...any) {
	l := logger.Load()
	if ctx == nil {
		ctx = context.Background()
	}
	if !l.Enabled(ctx, lv) {
		return
	}
	ctxArgs, _ := ctx.Value(attrsKey{}).([]any)
	all := make([]any, 0, 2+len(ctxArgs)+len(args))
	all = append(all, "event", event)
	all = append(all, ctxArgs...)
	all = append(all, args...)
	l.Log(ctx, lv, "", all...)
}

// Debug는 개발 중 흐름을 따라가기 위한 줄이다. 운영 기본값(INFO)에서는 남지 않는다.
func Debug(ctx context.Context, event string, args ...any) { Log(ctx, slog.LevelDebug, event, args...) }

// Info는 정상 동작이다.
func Info(ctx context.Context, event string, args ...any) { Log(ctx, slog.LevelInfo, event, args...) }

// Warn은 하나가 실패했거나 이상하지만 서버는 계속 정상인 경우다.
func Warn(ctx context.Context, event string, args ...any) { Log(ctx, slog.LevelWarn, event, args...) }

// Error는 서버 쪽 문제로 기능이 실패한 경우다. 같은 실패는 결과를 정하는 곳에서 한 번만 남긴다.
func Error(ctx context.Context, event string, args ...any) { Log(ctx, slog.LevelError, event, args...) }
