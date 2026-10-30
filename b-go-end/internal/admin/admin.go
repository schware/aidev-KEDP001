// Package admin은 관리자 화면(/admin/dashboard, /admin/api)이 쓰는 관리자 API다(specification/40.api/README.md 5장).
//
//	POST /api/admin/login              로그인
//	POST /api/admin/logout             로그아웃
//	GET  /api/admin/session            지금 로그인 상태
//	GET  /api/admin/indicators         전체 지표 목록(숨긴 지표 포함)
//	POST /api/admin/indicators         지표 추가
//	PUT  /api/admin/indicators/{id}    지표 수정
//	PUT  /api/admin/display-order      노출과 순서 저장(전체 목록)
//	GET  /api/admin/collect-time       수집 시각
//	PUT  /api/admin/collect-time       수집 시각 바꾸기
//	GET  /api/admin/sources            제공처별 상태
//
// 모든 요청은 허용 IP여야 하고(아니면 403), 로그인을 뺀 요청은 로그인해야 한다(아니면 401). KEDP001-D018.
// 관리자가 바꾼 내용은 서버 기록(log)에 남긴다. 비밀번호와 cookie 값은 남기지 않는다.
package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/schware/aidev-KEDP001/b-go-end/internal/store"
)

// Store는 관리자 API가 쓰는 DB 일이다(*store.Store).
type Store interface {
	AdminIndicators(ctx context.Context) ([]store.AdminIndicator, error)
	AdminIndicator(ctx context.Context, id string) (store.AdminIndicator, error)
	AddIndicator(ctx context.Context, n store.NewIndicator) error
	UpdateIndicator(ctx context.Context, id string, e store.Edit) error
	SaveOrder(ctx context.Context, items []store.OrderItem) error
	CollectTime(ctx context.Context) (string, error)
	SetCollectTime(ctx context.Context, v string) (string, error)
	SourceRuns(ctx context.Context) ([]store.SourceRun, error)
}

// Config는 관리자 API 설정이다.
type Config struct {
	Store Store
	// AllowIPs가 비어 있으면 모든 요청이 403이다.
	AllowIPs []netip.Prefix
	// TrustedProxies는 X-Forwarded-For를 믿을 proxy 주소다. 비어 있으면 127.0.0.1 하나다.
	TrustedProxies []netip.Prefix
	// Accounts가 비어 있으면 로그인할 수 없다.
	Accounts Accounts
	// KeySet은 제공처 코드별로 .env에 키가 있는지다. 키 값은 받지 않는다.
	KeySet map[string]bool
	// Now는 지금 시각이다. 비어 있으면 time.Now다.
	Now func() time.Time
	// Loc은 응답 시각의 시간대(Asia/Seoul)다.
	Loc *time.Location
}

type server struct {
	Config
	sessions sessions
}

// Handler는 관리자 API handler다.
func Handler(c Config) http.Handler {
	if c.Now == nil {
		c.Now = time.Now
	}
	if len(c.TrustedProxies) == 0 {
		c.TrustedProxies = []netip.Prefix{netip.MustParsePrefix("127.0.0.1/32")}
	}
	s := &server{Config: c}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/admin/login", methods(map[string]http.HandlerFunc{http.MethodPost: s.login}))
	mux.HandleFunc("/api/admin/logout", methods(map[string]http.HandlerFunc{http.MethodPost: s.auth(s.logout)}))
	mux.HandleFunc("/api/admin/session", methods(map[string]http.HandlerFunc{http.MethodGet: s.auth(s.session)}))
	mux.HandleFunc("/api/admin/indicators", methods(map[string]http.HandlerFunc{
		http.MethodGet: s.auth(s.listIndicators), http.MethodPost: s.auth(s.addIndicator)}))
	mux.HandleFunc("/api/admin/indicators/{id}", methods(map[string]http.HandlerFunc{http.MethodPut: s.auth(s.editIndicator)}))
	mux.HandleFunc("/api/admin/display-order", methods(map[string]http.HandlerFunc{http.MethodPut: s.auth(s.saveOrder)}))
	mux.HandleFunc("/api/admin/collect-time", methods(map[string]http.HandlerFunc{
		http.MethodGet: s.auth(s.getCollectTime), http.MethodPut: s.auth(s.setCollectTime)}))
	mux.HandleFunc("/api/admin/sources", methods(map[string]http.HandlerFunc{http.MethodGet: s.auth(s.sources)}))
	mux.HandleFunc("/api/admin/", func(w http.ResponseWriter, r *http.Request) {
		write(w, http.StatusNotFound, Error{Error: "not found"})
	})
	return s.allowIP(mux)
}

// Error는 오류 응답이다. Fields는 입력 오류(400)일 때 칸 이름별 이유다.
type Error struct {
	Error  string            `json:"error"`
	Fields map[string]string `json:"fields,omitempty"`
}

// allowIP는 허용 IP가 아닌 요청을 403으로 막는다(KEDP001-D018).
func (s *server) allowIP(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		addr, ok := ClientIP(r.RemoteAddr, r.Header.Values("X-Forwarded-For"), s.TrustedProxies)
		if !ok || !allowed(s.AllowIPs, addr) {
			write(w, http.StatusForbidden, Error{Error: "forbidden"})
			return
		}
		h.ServeHTTP(w, r)
	})
}

type ctxKey struct{}

// account는 auth를 지난 요청의 계정이다.
func account(r *http.Request) string {
	v, _ := r.Context().Value(ctxKey{}).(string)
	return v
}

// auth는 로그인하지 않은 요청을 401로 막는다.
func (s *server) auth(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(CookieName)
		if err != nil {
			write(w, http.StatusUnauthorized, Error{Error: "login required"})
			return
		}
		ss, ok := s.sessions.get(c.Value, s.Now())
		if !ok {
			write(w, http.StatusUnauthorized, Error{Error: "login required"})
			return
		}
		h(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, ss.account)))
	}
}

// methods는 method별 handler를 고른다. 없는 method는 405다. GET이 있으면 HEAD도 받는다.
// 바꾸는 요청(POST, PUT)은 Content-Type이 application/json이어야 한다(아니면 415).
func methods(m map[string]http.HandlerFunc) http.HandlerFunc {
	var allow []string
	for _, k := range []string{http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut} {
		if m[k] != nil || (k == http.MethodHead && m[http.MethodGet] != nil) {
			allow = append(allow, k)
		}
	}
	return func(w http.ResponseWriter, r *http.Request) {
		h := m[r.Method]
		if h == nil && r.Method == http.MethodHead {
			h = m[http.MethodGet]
		}
		if h == nil {
			w.Header().Set("Allow", strings.Join(allow, ", "))
			write(w, http.StatusMethodNotAllowed, Error{Error: "method not allowed"})
			return
		}
		if r.Method == http.MethodPost || r.Method == http.MethodPut {
			mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil || mt != "application/json" {
				write(w, http.StatusUnsupportedMediaType, Error{Error: "content type must be application/json"})
				return
			}
		}
		h(w, r)
	}
}

// maxBody는 요청 본문의 최대 크기다.
const maxBody = 64 << 10

// errBadJSON은 본문을 JSON으로 읽을 수 없다는 오류다. 칸의 형식이 틀리면 *store.FieldError다.
var errBadJSON = errors.New("invalid JSON")

// decode는 본문 JSON 하나를 v로 읽는다. 모르는 칸이 있으면 오류다.
func decode(w http.ResponseWriter, r *http.Request, v any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		var typeErr *json.UnmarshalTypeError
		if errors.As(err, &typeErr) && typeErr.Field != "" {
			return &store.FieldError{Field: typeErr.Field, Reason: "must be " + typeName(typeErr.Type.String())}
		}
		if f, ok := strings.CutPrefix(err.Error(), "json: unknown field "); ok {
			return &store.FieldError{Field: strings.Trim(f, `"`), Reason: "unknown field"}
		}
		return errBadJSON
	}
	if _, err := dec.Token(); err != io.EOF {
		return errBadJSON
	}
	return nil
}

func typeName(t string) string {
	t = strings.TrimPrefix(t, "*")
	switch {
	case t == "string":
		return "a string"
	case t == "bool":
		return "true or false"
	case strings.HasPrefix(t, "int"):
		return "an integer"
	case strings.HasPrefix(t, "[]"):
		return "an array of " + strings.TrimPrefix(typeName(t[2:]), "a ") + "s"
	}
	return t
}

// badRequest는 decode·검사 오류를 400으로 돌려준다. 다른 오류면 false다.
func badRequest(w http.ResponseWriter, err error) bool {
	var fe *store.FieldError
	var fes fieldErrors
	switch {
	case errors.Is(err, errBadJSON):
		write(w, http.StatusBadRequest, Error{Error: "invalid JSON"})
	case errors.As(err, &fe):
		write(w, http.StatusBadRequest, Error{Error: "invalid input", Fields: map[string]string{fe.Field: fe.Reason}})
	case errors.As(err, &fes):
		write(w, http.StatusBadRequest, Error{Error: "invalid input", Fields: fes})
	default:
		return false
	}
	return true
}

func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("admin: write response: %v", err)
	}
}

// serverError는 오류를 기록하고 화면에는 내용 없이 500을 돌려준다.
func serverError(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("admin: %s %s: %v", r.Method, r.URL.Path, err)
	write(w, http.StatusInternalServerError, Error{Error: "internal error"})
}

// audit은 관리자가 바꾼 내용을 서버 기록에 남긴다.
func audit(r *http.Request, format string, args ...any) {
	log.Printf("admin audit: account %q: %s", account(r), fmt.Sprintf(format, args...))
}

// Session은 로그인 상태다.
type Session struct {
	Account   string `json:"account"`
	ExpiresAt string `json:"expires_at"`
}

type loginRequest struct {
	Account  *string `json:"account"`
	Password *string `json:"password"`
}

func (s *server) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decode(w, r, &req); err != nil {
		badRequest(w, err)
		return
	}
	if req.Account == nil || req.Password == nil {
		write(w, http.StatusBadRequest, Error{Error: "invalid input", Fields: required(map[string]bool{
			"account": req.Account == nil, "password": req.Password == nil})})
		return
	}
	ip, _ := ClientIP(r.RemoteAddr, r.Header.Values("X-Forwarded-For"), s.TrustedProxies)
	// 계정이 틀렸든 비밀번호가 틀렸든 같은 응답이다. 기록에도 받은 계정 이름을 넣지 않는다(비밀번호를 잘못 넣었을 수 있다).
	if !s.Accounts.verify(*req.Account, *req.Password) {
		log.Printf("admin: login failed from %s", ip)
		write(w, http.StatusUnauthorized, Error{Error: "login failed"})
		return
	}
	if c, err := r.Cookie(CookieName); err == nil {
		s.sessions.delete(c.Value)
	}
	now := s.Now()
	token, ss, err := s.sessions.create(*req.Account, now)
	if err != nil {
		serverError(w, r, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: CookieName, Value: token, Path: CookiePath,
		MaxAge: int(SessionTTL / time.Second), Expires: ss.expires,
		HttpOnly: true, SameSite: http.SameSiteStrictMode,
		// Secure는 배포 Issue에서 HTTPS와 함께 켠다.
	})
	log.Printf("admin: account %q logged in from %s", ss.account, ip)
	write(w, http.StatusOK, s.sessionOut(ss))
}

func (s *server) sessionOut(ss session) Session {
	return Session{Account: ss.account, ExpiresAt: ss.expires.In(s.Loc).Format(time.RFC3339)}
}

func (s *server) logout(w http.ResponseWriter, r *http.Request) {
	c, _ := r.Cookie(CookieName) // auth를 지났으므로 있다
	s.sessions.delete(c.Value)
	http.SetCookie(w, &http.Cookie{
		Name: CookieName, Value: "", Path: CookiePath, MaxAge: -1,
		HttpOnly: true, SameSite: http.SameSiteStrictMode,
	})
	log.Printf("admin: account %q logged out", account(r))
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) session(w http.ResponseWriter, r *http.Request) {
	c, _ := r.Cookie(CookieName)
	ss, ok := s.sessions.get(c.Value, s.Now())
	if !ok {
		write(w, http.StatusUnauthorized, Error{Error: "login required"})
		return
	}
	write(w, http.StatusOK, s.sessionOut(ss))
}

// Indicator는 관리자 지표 목록의 지표 하나다.
type Indicator struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Source     string   `json:"source"`
	StatCode   string   `json:"stat_code"`
	ItemCode   *string  `json:"item_code"`
	Classes    []string `json:"classes"`
	OrgID      *string  `json:"org_id"`
	ValueField *string  `json:"value_field"`
	Cycle      string   `json:"cycle"`
	Unit       *string  `json:"unit"`
	SourceUnit *string  `json:"source_unit"`
	Visible    bool     `json:"visible"`
	// Order와 ChartPeriod는 화면 설정이 없으면 null이다.
	Order       *int     `json:"order"`
	ChartPeriod *string  `json:"chart_period"`
	Decimals    *int     `json:"decimals"`
	Notice      *string  `json:"notice"`
	Related     []string `json:"related"`
}

// IndicatorList는 GET /api/admin/indicators의 응답이다.
type IndicatorList struct {
	Indicators []Indicator `json:"indicators"`
}

func indicatorOut(a store.AdminIndicator) Indicator {
	return Indicator{
		ID: a.ID, Name: a.Name, Source: a.Source, StatCode: a.StatCode, ItemCode: a.ItemCode,
		Classes: nonNil(a.Classes), OrgID: a.OrgID, ValueField: a.ValueField, Cycle: a.Cycle,
		Unit: a.Unit, SourceUnit: a.SourceUnit, Visible: a.Visible, Order: a.Order,
		ChartPeriod: a.ChartPeriod, Decimals: a.Decimals, Notice: a.Notice, Related: nonNil(a.Related),
	}
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func (s *server) writeIndicators(w http.ResponseWriter, r *http.Request) {
	list, err := s.Store.AdminIndicators(r.Context())
	if err != nil {
		serverError(w, r, err)
		return
	}
	out := IndicatorList{Indicators: make([]Indicator, 0, len(list))}
	for _, a := range list {
		out.Indicators = append(out.Indicators, indicatorOut(a))
	}
	write(w, http.StatusOK, out)
}

func (s *server) listIndicators(w http.ResponseWriter, r *http.Request) { s.writeIndicators(w, r) }

func (s *server) addIndicator(w http.ResponseWriter, r *http.Request) {
	var req addRequest
	if err := decode(w, r, &req); err != nil {
		badRequest(w, err)
		return
	}
	n, errs := req.validate()
	if len(errs) > 0 {
		badRequest(w, errs)
		return
	}
	err := s.Store.AddIndicator(r.Context(), n)
	if errors.Is(err, store.ErrExists) {
		write(w, http.StatusConflict, Error{Error: "indicator already exists"})
		return
	}
	if badRequest(w, err) {
		return
	}
	if err != nil {
		serverError(w, r, err)
		return
	}
	a, err := s.Store.AdminIndicator(r.Context(), n.ID)
	if err != nil {
		serverError(w, r, err)
		return
	}
	audit(r, "add indicator %s %s", a.ID, describe(a))
	write(w, http.StatusCreated, indicatorOut(a))
}

func (s *server) editIndicator(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req editRequest
	if err := decode(w, r, &req); err != nil {
		badRequest(w, err)
		return
	}
	e, errs := req.validate(id)
	if len(errs) > 0 {
		badRequest(w, errs)
		return
	}
	before, err := s.Store.AdminIndicator(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		write(w, http.StatusNotFound, Error{Error: "indicator not found"})
		return
	}
	if err != nil {
		serverError(w, r, err)
		return
	}
	err = s.Store.UpdateIndicator(r.Context(), id, e)
	if errors.Is(err, store.ErrNotFound) {
		write(w, http.StatusNotFound, Error{Error: "indicator not found"})
		return
	}
	if badRequest(w, err) {
		return
	}
	if err != nil {
		serverError(w, r, err)
		return
	}
	after, err := s.Store.AdminIndicator(r.Context(), id)
	if err != nil {
		serverError(w, r, err)
		return
	}
	audit(r, "edit indicator %s: %s", id, changes(before, after))
	write(w, http.StatusOK, indicatorOut(after))
}

func (s *server) saveOrder(w http.ResponseWriter, r *http.Request) {
	var req orderRequest
	if err := decode(w, r, &req); err != nil {
		badRequest(w, err)
		return
	}
	items, errs := req.validate()
	if len(errs) > 0 {
		badRequest(w, errs)
		return
	}
	before, err := s.Store.AdminIndicators(r.Context())
	if err != nil {
		serverError(w, r, err)
		return
	}
	err = s.Store.SaveOrder(r.Context(), items)
	if badRequest(w, err) {
		return
	}
	if err != nil {
		serverError(w, r, err)
		return
	}
	audit(r, "save display order: %s", orderChanges(before, items))
	s.writeIndicators(w, r)
}

// CollectTime은 수집 시각이다(HH:MM, Asia/Seoul).
type CollectTime struct {
	CollectTime string `json:"collect_time"`
}

func (s *server) getCollectTime(w http.ResponseWriter, r *http.Request) {
	v, err := s.Store.CollectTime(r.Context())
	if err != nil {
		serverError(w, r, err)
		return
	}
	write(w, http.StatusOK, CollectTime{v})
}

func (s *server) setCollectTime(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CollectTime *string `json:"collect_time"`
	}
	if err := decode(w, r, &req); err != nil {
		badRequest(w, err)
		return
	}
	if req.CollectTime == nil || !clockPattern.MatchString(*req.CollectTime) {
		badRequest(w, &store.FieldError{Field: "collect_time", Reason: "must be HH:MM (00:00-23:59)"})
		return
	}
	old, err := s.Store.SetCollectTime(r.Context(), *req.CollectTime)
	if err != nil {
		serverError(w, r, err)
		return
	}
	audit(r, "collect_time %s -> %s", old, *req.CollectTime)
	write(w, http.StatusOK, CollectTime{*req.CollectTime})
}

// 제공처 상태(specification/40.api 5장)
const (
	StatusConnected = "connected"
	StatusError     = "error"
	StatusUnchecked = "unchecked"
	StatusNoKey     = "no_key"
)

// SourceState는 제공처 하나의 상태다.
type SourceState struct {
	Code            string  `json:"code"`
	Name            string  `json:"name"`
	KeySet          bool    `json:"key_set"`
	LastCollectedAt *string `json:"last_collected_at"`
	Status          string  `json:"status"`
}

// SourceList는 GET /api/admin/sources의 응답이다.
type SourceList struct {
	Sources []SourceState `json:"sources"`
}

// sourceStatus는 키 설정 여부와 마지막으로 끝난 수집 기록의 상태로 제공처 상태를 정한다. 실제 연결 시험은 하지 않는다.
func sourceStatus(keySet bool, runStatus string) string {
	switch {
	case !keySet:
		return StatusNoKey
	case runStatus == store.StatusSuccess || runStatus == store.StatusNoData:
		return StatusConnected
	case runStatus == store.StatusError:
		return StatusError
	}
	return StatusUnchecked
}

func (s *server) sources(w http.ResponseWriter, r *http.Request) {
	runs, err := s.Store.SourceRuns(r.Context())
	if err != nil {
		serverError(w, r, err)
		return
	}
	out := SourceList{Sources: make([]SourceState, 0, len(runs))}
	for _, x := range runs {
		st := SourceState{Code: x.Code, Name: x.Name, KeySet: s.KeySet[x.Code], Status: sourceStatus(s.KeySet[x.Code], x.Status)}
		if x.FinishedAt != nil {
			v := x.FinishedAt.In(s.Loc).Format(time.RFC3339)
			st.LastCollectedAt = &v
		}
		out.Sources = append(out.Sources, st)
	}
	write(w, http.StatusOK, out)
}
