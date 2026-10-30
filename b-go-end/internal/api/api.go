// Package api는 사용자 화면이 읽는 조회 API다(specification/40.api/README.md). DB는 읽기만 한다.
//
//	GET /api/indicators        메인 화면 카드 목록
//	GET /api/indicators/{id}   지표 상세
//
// 화면 개발은 Vite proxy로 연결하므로 CORS 헤더는 두지 않는다.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/schware/aidev-KEDP001/b-go-end/internal/store"
)

// RecentRows는 상세 화면 최근 데이터 표의 행 수다.
const RecentRows = 10

// Store는 API가 읽는 DB 일이다(*store.Store).
type Store interface {
	Cards(ctx context.Context) ([]store.Card, error)
	Detail(ctx context.Context, id string, today time.Time, recent int) (store.Detail, error)
}

// Point는 관측값 하나다. period는 기간 시작일(YYYY-MM-DD)이다.
type Point struct {
	Period string  `json:"period"`
	Value  float64 `json:"value"`
}

// Card는 메인 화면 카드 하나다.
type Card struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Cycle  string  `json:"cycle"`
	Unit   *string `json:"unit"`
	Latest *Point  `json:"latest"`
}

// CardList는 GET /api/indicators의 응답이다.
type CardList struct {
	Indicators []Card `json:"indicators"`
}

// Source는 제공처다.
type Source struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// Related는 관련 지표 하나다.
type Related struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Detail은 GET /api/indicators/{id}의 응답이다.
type Detail struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Cycle    string  `json:"cycle"`
	Unit     *string `json:"unit"`
	Notice   *string `json:"notice"`
	Source   Source  `json:"source"`
	StatCode string  `json:"stat_code"`
	// LastCollectedAt은 Asia/Seoul 시각(RFC 3339)이다.
	LastCollectedAt *string   `json:"last_collected_at"`
	Latest          *Point    `json:"latest"`
	ChartPeriod     *string   `json:"chart_period"`
	Chart           []Point   `json:"chart"`
	Recent          []Point   `json:"recent"`
	Related         []Related `json:"related"`
}

// Error는 오류 응답이다.
type Error struct {
	Error string `json:"error"`
}

// Handler는 조회 API handler다. now는 지금 시각이고, loc은 "오늘"과 시각 표시의 시간대(Asia/Seoul)다.
func Handler(st Store, now func() time.Time, loc *time.Location) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/indicators", getOnly(func(w http.ResponseWriter, r *http.Request) {
		cards, err := st.Cards(r.Context())
		if err != nil {
			serverError(w, r, err)
			return
		}
		out := CardList{Indicators: make([]Card, 0, len(cards))}
		for _, c := range cards {
			out.Indicators = append(out.Indicators, Card{
				ID: c.ID, Name: c.Name, Cycle: c.Cycle, Unit: c.Unit, Latest: point(c.Latest),
			})
		}
		write(w, http.StatusOK, out)
	}))
	mux.HandleFunc("/api/indicators/{id}", getOnly(func(w http.ResponseWriter, r *http.Request) {
		d, err := st.Detail(r.Context(), r.PathValue("id"), now().In(loc), RecentRows)
		if errors.Is(err, store.ErrNotFound) {
			write(w, http.StatusNotFound, Error{"indicator not found"})
			return
		}
		if err != nil {
			serverError(w, r, err)
			return
		}
		out := Detail{
			ID: d.ID, Name: d.Name, Cycle: d.Cycle, Unit: d.Unit, Notice: d.Notice,
			Source:      Source{Code: d.SourceCode, Name: d.SourceName},
			StatCode:    d.StatCode,
			Latest:      point(d.Latest),
			ChartPeriod: d.ChartPeriod,
			Chart:       points(d.Chart),
			Recent:      points(d.Recent),
			Related:     make([]Related, 0, len(d.Related)),
		}
		if d.LastCollectedAt != nil {
			s := d.LastCollectedAt.In(loc).Format(time.RFC3339)
			out.LastCollectedAt = &s
		}
		for _, x := range d.Related {
			out.Related = append(out.Related, Related{ID: x.ID, Name: x.Name})
		}
		write(w, http.StatusOK, out)
	}))
	// 그 밖의 주소는 404다. 오류도 JSON이다.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		write(w, http.StatusNotFound, Error{"not found"})
	})
	return mux
}

// getOnly는 GET(과 HEAD)만 받는다. 다른 method는 405다.
func getOnly(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			write(w, http.StatusMethodNotAllowed, Error{"method not allowed"})
			return
		}
		h(w, r)
	}
}

func point(p *store.Point) *Point {
	if p == nil {
		return nil
	}
	return &Point{Period: p.Period.Format(time.DateOnly), Value: p.Value}
}

// points는 비어 있어도 null이 아니라 []다.
func points(list []store.Point) []Point {
	out := make([]Point, 0, len(list))
	for _, p := range list {
		out = append(out, *point(&p))
	}
	return out
}

func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("api: write response: %v", err)
	}
}

// serverError는 오류를 기록하고 화면에는 내용 없이 500을 돌려준다.
func serverError(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("api: %s %s: %v", r.Method, r.URL.Path, err)
	write(w, http.StatusInternalServerError, Error{"internal error"})
}

// Serve는 ln으로 받은 요청을 h로 처리한다. ctx가 끝나면 처리 중인 요청을 기다렸다가(최대 5초) 멈춘다.
func Serve(ctx context.Context, ln net.Listener, h http.Handler) error {
	srv := &http.Server{
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       time.Minute,
	}
	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ln) }()
	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	if err := <-errc; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
