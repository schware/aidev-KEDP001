package api

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"log"
	"net/http"
	"net/netip"
	"sync"
	"time"

	"github.com/schware/aidev-KEDP001/b-go-end/internal/admin"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/store"
)

// 방문자 수 API(specification/40.api/README.md 6장, Issue #48). 인증은 없다.
//
//	POST /api/visits           방문 하나. 그날 이미 센 사람이면 세지 않는다. 204
//	GET  /api/visits/summary   오늘과 전체 방문자 수
//
// 같은 사람은 접속 주소 + User-Agent + 그날의 salt로 만든 hash로 알아본다. salt와 그날 hash 목록은
// 서버 메모리에만 있고, 날이 바뀌면(Asia/Seoul 00:00) 함께 버린다. 서버를 다시 켜도 사라진다.
// 접속 주소, User-Agent, hash는 DB, 파일, 서버 기록 어디에도 남기지 않는다.

// MaxVisitorsPerDay는 하루에 기억하는 사람 수의 상한이다. 넘으면 그날은 더 세지 않는다(메모리 보호).
const MaxVisitorsPerDay = 200_000

// VisitStore는 방문자 수 API가 쓰는 DB 일이다(*store.Store).
type VisitStore interface {
	AddVisit(ctx context.Context, day time.Time) error
	VisitSummary(ctx context.Context, today time.Time) (store.VisitSummary, error)
}

// VisitConfig는 방문자 수 API 설정이다.
type VisitConfig struct {
	Store VisitStore
	// TrustedProxies는 X-Forwarded-For를 믿을 proxy 주소다(관리자 API와 같다, 40.api 5.1).
	// 비어 있으면 127.0.0.1 하나다.
	TrustedProxies []netip.Prefix
	// Now는 지금 시각이다. 비어 있으면 time.Now다.
	Now func() time.Time
	// Loc은 "오늘"의 시간대(Asia/Seoul)다.
	Loc *time.Location
	// Limit은 하루에 기억하는 사람 수의 상한이다. 0이면 MaxVisitorsPerDay다.
	Limit int
}

// VisitSummary는 GET /api/visits/summary의 응답이다.
type VisitSummary struct {
	Today int64 `json:"today"`
	Total int64 `json:"total"`
}

// VisitHandler는 방문자 수 API handler다. /api/visits, /api/visits/summary 두 경로를 받는다.
func VisitHandler(c VisitConfig) http.Handler {
	_, h := newVisitHandler(c)
	return h
}

func newVisitHandler(c VisitConfig) (*visits, http.Handler) {
	if c.Now == nil {
		c.Now = time.Now
	}
	if len(c.TrustedProxies) == 0 {
		c.TrustedProxies = []netip.Prefix{netip.MustParsePrefix("127.0.0.1/32")}
	}
	if c.Limit <= 0 {
		c.Limit = MaxVisitorsPerDay
	}
	v := &visits{VisitConfig: c, seen: newVisitors(c.Limit)}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/visits", v.add)
	mux.HandleFunc("/api/visits/summary", noStore(getOnly(v.summary)))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		write(w, http.StatusNotFound, Error{"not found"})
	})
	return v, mux
}

type visits struct {
	VisitConfig
	seen *visitors
}

func noStore(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		h(w, r)
	}
}

// add는 방문 하나를 센다. 본문은 읽지 않는다. 그날 이미 센 사람, 접속 주소를 알 수 없는 요청,
// 그날 상한을 넘은 요청은 세지 않고 똑같이 204를 돌려준다.
func (v *visits) add(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		write(w, http.StatusMethodNotAllowed, Error{"method not allowed"})
		return
	}
	ip, ok := admin.ClientIP(r.RemoteAddr, r.Header.Values("X-Forwarded-For"), v.TrustedProxies)
	if !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	day := v.Now().In(v.Loc)
	key, isNew, err := v.seen.mark(day.Format(time.DateOnly), ip, r.UserAgent())
	if err != nil {
		serverError(w, r, err)
		return
	}
	if isNew {
		if err := v.Store.AddVisit(r.Context(), day); err != nil {
			// 세지 못했으므로 다음 요청에서 다시 셀 수 있게 지운다.
			v.seen.unmark(day.Format(time.DateOnly), key)
			serverError(w, r, err)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (v *visits) summary(w http.ResponseWriter, r *http.Request) {
	s, err := v.Store.VisitSummary(r.Context(), v.Now().In(v.Loc))
	if err != nil {
		serverError(w, r, err)
		return
	}
	write(w, http.StatusOK, VisitSummary{Today: s.Today, Total: s.Total})
}

// visitors는 그날 센 사람의 hash 목록이다. 서버 메모리에만 있다.
type visitors struct {
	mu    sync.Mutex
	limit int
	day   string // YYYY-MM-DD, Asia/Seoul
	salt  []byte
	seen  map[[sha256.Size]byte]struct{}
	full  bool // 그날 상한에 닿았다고 기록했는지
}

func newVisitors(limit int) *visitors { return &visitors{limit: limit} }

// mark는 (ip, ua)가 day에 처음이면 목록에 넣고 true를 돌려준다. day가 바뀌었으면 salt와 목록을 새로 만든다.
// 목록이 상한에 닿았으면 넣지 않고 false다.
func (c *visitors) mark(day string, ip netip.Addr, ua string) ([sha256.Size]byte, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if day != c.day || c.salt == nil {
		salt := make([]byte, 32)
		if _, err := rand.Read(salt); err != nil {
			return [sha256.Size]byte{}, false, err
		}
		c.day, c.salt, c.seen, c.full = day, salt, map[[sha256.Size]byte]struct{}{}, false
	}
	m := hmac.New(sha256.New, c.salt)
	m.Write(ip.AsSlice())
	m.Write([]byte{0})
	m.Write([]byte(ua))
	var key [sha256.Size]byte
	copy(key[:], m.Sum(nil))
	if _, ok := c.seen[key]; ok {
		return key, false, nil
	}
	if len(c.seen) >= c.limit {
		if !c.full {
			c.full = true
			log.Printf("api: visits: %d visitors reached for %s; not counting more today", c.limit, day)
		}
		return key, false, nil
	}
	c.seen[key] = struct{}{}
	return key, true, nil
}

// unmark는 mark로 넣은 key를 지운다. 그사이 날이 바뀌었으면 아무것도 하지 않는다.
func (c *visitors) unmark(day string, key [sha256.Size]byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if day == c.day {
		delete(c.seen, key)
	}
}
