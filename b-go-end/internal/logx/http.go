package logx

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"
)

// RequestIDHeader는 요청 ID를 주고받는 헤더다. nginx가 만들어 넘기고, Go는 응답에도 돌려준다(D026 5장).
const RequestIDHeader = "X-Request-ID"

// maxRequestID는 받아들이는 요청 ID의 최대 길이다. nginx $request_id는 32자다.
const maxRequestID = 64

type requestKey struct{}

// request는 요청 하나가 끝날 때 http.request 줄에 붙일 것이다. handler가 SetError, AddAttrs로 채운다.
type request struct {
	err   error
	attrs []any
}

// HTTP는 요청마다 끝날 때 http.request 한 줄을 남기는 middleware다.
//   - 요청 ID: X-Request-ID를 받아 쓰고, 없거나 형식이 틀리면 만든다. 그 요청의 모든 줄에 req로 붙이고 응답 헤더로 돌려준다.
//   - level: 응답 코드로 정한다(StatusLevel).
//   - path에는 query string을 넣지 않는다. 접속 주소와 User-Agent는 남기지 않는다.
func HTTP(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		id := r.Header.Get(RequestIDHeader)
		if !validRequestID(id) {
			id = newRequestID()
		}
		w.Header().Set(RequestIDHeader, id)
		info := &request{}
		ctx := context.WithValue(With(r.Context(), "req", id), requestKey{}, info)
		rec := &statusRecorder{ResponseWriter: w}
		h.ServeHTTP(rec, r.WithContext(ctx))

		status := rec.status
		if status == 0 {
			status = http.StatusOK
		}
		args := []any{"method", r.Method, "path", r.URL.Path, "status", status, "ms", time.Since(start).Milliseconds()}
		args = append(args, info.attrs...)
		if info.err != nil {
			args = append(args, "err", info.err.Error())
		}
		Log(ctx, StatusLevel(r.Method, r.URL.Path, status), "http.request", args...)
	})
}

// StatusLevel은 응답 코드에 맞는 level이다(D026 3장).
// 2xx, 3xx, 404, 로그인하지 않은 GET /api/admin/session의 401은 INFO, 그 밖의 4xx는 WARN, 5xx는 ERROR다.
func StatusLevel(method, path string, status int) slog.Level {
	switch {
	case status >= 500:
		return slog.LevelError
	case status == http.StatusNotFound:
		return slog.LevelInfo
	case status == http.StatusUnauthorized && method == http.MethodGet && path == "/api/admin/session":
		return slog.LevelInfo
	case status >= 400:
		return slog.LevelWarn
	}
	return slog.LevelInfo
}

// SetError는 이 요청의 http.request 줄에 err를 붙인다. 500을 돌려줄 때 쓴다.
// HTTP middleware 밖(시험 등)이면 바로 http.error 한 줄(ERROR)로 남긴다.
func SetError(r *http.Request, err error) {
	if info, ok := r.Context().Value(requestKey{}).(*request); ok {
		info.err = err
		return
	}
	Error(r.Context(), "http.error", "method", r.Method, "path", r.URL.Path, "err", err.Error())
}

// AddAttrs는 이 요청의 http.request 줄에 args(key, value 쌍)를 붙인다.
// HTTP middleware 밖이면 아무것도 하지 않는다.
func AddAttrs(r *http.Request, args ...any) {
	if info, ok := r.Context().Value(requestKey{}).(*request); ok {
		info.attrs = append(info.attrs, args...)
	}
}

// validRequestID는 로그에 그대로 넣어도 되는 요청 ID인지 본다(영문, 숫자, -, _, ., 64자까지).
func validRequestID(id string) bool {
	if id == "" || len(id) > maxRequestID {
		return false
	}
	for _, c := range id {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '-', c == '_', c == '.':
		default:
			return false
		}
	}
	return true
}

// newRequestID는 nginx $request_id와 같은 꼴(16바이트, hex 32자)을 만든다.
func newRequestID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b) // crypto/rand.Read는 실패하지 않는다
	return hex.EncodeToString(b)
}

// statusRecorder는 handler가 쓴 응답 코드를 기억한다.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	if s.status == 0 {
		s.status = code
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	return s.ResponseWriter.Write(b)
}

// Unwrap은 http.ResponseController가 원래 ResponseWriter를 찾게 한다.
func (s *statusRecorder) Unwrap() http.ResponseWriter { return s.ResponseWriter }
