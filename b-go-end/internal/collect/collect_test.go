package collect

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/schware/aidev-KEDP001/b-go-end/internal/config"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/fetch"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/kedpv1"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/source"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/source/ecos"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/source/kosis"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/store"
)

// 실제 API는 부르지 않는다. 제공처 요청은 httptest 서버로 보낸다(fetch.Client.SetHost).

// testKey는 가짜 키다. 실제 키 모양이 아니어야 commit 검사(gitleaks)에 걸리지 않는다.
const testKey = "fakefakefakefake"

type run struct {
	source, indicator, trigger string
	status, code               string
	rows                       int
}

type fakeStore struct {
	targets []store.Target
	runs    []*run
}

func (s *fakeStore) Targets(context.Context, time.Time) ([]store.Target, error) {
	return s.targets, nil
}
func (s *fakeStore) CollectTime(context.Context) (string, error) { return "06:00", nil }

func (s *fakeStore) StartRun(_ context.Context, src, ind, trigger string) (int64, error) {
	s.runs = append(s.runs, &run{source: src, indicator: ind, trigger: trigger, status: "running"})
	return int64(len(s.runs)), nil
}

func (s *fakeStore) FinishRun(_ context.Context, id int64, status, code string, rows int) error {
	r := s.runs[id-1]
	if r.status != "running" {
		return fmt.Errorf("run %d is not running", id)
	}
	r.status, r.code, r.rows = status, code, rows
	return nil
}

// fakeProcessor는 받은 요청을 남기고 respond의 결과를 돌려준다.
type fakeProcessor struct {
	got     []*kedpv1.ProcessPageRequest
	respond func(*kedpv1.ProcessPageRequest) (*kedpv1.ProcessPageResponse, error)
}

func (p *fakeProcessor) ProcessPage(_ context.Context, in *kedpv1.ProcessPageRequest, _ ...grpc.CallOption) (*kedpv1.ProcessPageResponse, error) {
	p.got = append(p.got, in)
	return p.respond(in)
}

func success(n int32) func(*kedpv1.ProcessPageRequest) (*kedpv1.ProcessPageResponse, error) {
	return func(*kedpv1.ProcessPageRequest) (*kedpv1.ProcessPageResponse, error) {
		return &kedpv1.ProcessPageResponse{Status: kedpv1.ProcessStatus_PROCESS_STATUS_SUCCESS, InsertedCount: n, UnchangedCount: 1, SkippedEmptyCount: 2}, nil
	}
}

var today = time.Date(2026, 10, 30, 6, 0, 0, 0, Seoul)

func date(y int, m time.Month, d int) *time.Time {
	t := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	return &t
}

func ecosTarget(id string) store.Target {
	return store.Target{
		Indicator:  config.Indicator{ID: id, Source: "ecos", StatCode: "722Y001", ItemCode: "0101000", Cycle: "M"},
		FirstStart: date(2016, 10, 30),
	}
}

// newCollector는 handler를 제공처 서버로 쓰는 Collector를 만든다. ECOS 키는 ecosKey다.
func newCollector(t *testing.T, handler http.HandlerFunc, ecosKey string, st *fakeStore, p *fakeProcessor) *Collector {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	client := fetch.New(5*time.Second, testKey)
	if err := client.SetHost(srv.URL); err != nil {
		t.Fatal(err)
	}
	return &Collector{
		Store:     st,
		Processor: p,
		Sources: map[string]source.Source{
			"ecos":  ecos.New(ecosKey, client),
			"kosis": kosis.New(testKey, client),
		},
		HasKey: client.HasKey,
		Now:    func() time.Time { return today },
	}
}

func ecosBody(total int) string {
	return fmt.Sprintf(`{"StatisticSearch":{"list_total_count":%d,"row":[]}}`, total)
}

func TestRunPages(t *testing.T) {
	var paths []string
	handler := func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		fmt.Fprint(w, ecosBody(2500))
	}
	st := &fakeStore{targets: []store.Target{ecosTarget("ecos-base-rate")}}
	p := &fakeProcessor{respond: success(10)}
	c := newCollector(t, handler, testKey, st, p)

	sum, err := c.Run(context.Background(), store.TriggerManual)
	if err != nil {
		t.Fatal(err)
	}
	if sum != (Summary{Success: 1}) {
		t.Errorf("summary = %v", sum)
	}
	// 2500행은 1000행씩 3페이지다. 차트 기간(10년)의 시작 달부터 이번 달까지 요청한다.
	if len(paths) != 3 {
		t.Fatalf("requests = %d, want 3", len(paths))
	}
	for i, want := range []string{"/1/1000/722Y001/M/201610/202610/0101000", "/1001/2000/", "/2001/3000/"} {
		if !strings.Contains(paths[i], want) {
			t.Errorf("request %d = %s, want %s", i+1, strings.ReplaceAll(paths[i], testKey, "***"), want)
		}
	}
	if len(p.got) != 3 {
		t.Fatalf("ProcessPage calls = %d, want 3", len(p.got))
	}
	for i, req := range p.got {
		if req.RunId != 1 || req.IndicatorId != "ecos-base-rate" || req.Page != int32(i+1) || req.HttpStatus != 200 {
			t.Errorf("request %d = run %d, %s, page %d, status %d", i+1, req.RunId, req.IndicatorId, req.Page, req.HttpStatus)
		}
		if !strings.Contains(req.RequestMasked, "***") || strings.Contains(req.RequestMasked, testKey) {
			t.Errorf("request %d: request_masked is not masked", i+1)
		}
		if req.ReceivedAt == nil || string(req.Body) != ecosBody(2500) {
			t.Errorf("request %d: received_at or body is wrong", i+1)
		}
	}
	// 행 수는 페이지마다 (새로 넣음 10 + 그대로 1)을 더한 것이다. 빈 값은 세지 않는다.
	want := run{source: "ecos", indicator: "ecos-base-rate", trigger: "manual", status: "success", rows: 33}
	if *st.runs[0] != want {
		t.Errorf("run = %+v, want %+v", *st.runs[0], want)
	}
}

func TestRunNoData(t *testing.T) {
	st := &fakeStore{targets: []store.Target{ecosTarget("a")}}
	p := &fakeProcessor{respond: func(*kedpv1.ProcessPageRequest) (*kedpv1.ProcessPageResponse, error) {
		return &kedpv1.ProcessPageResponse{Status: kedpv1.ProcessStatus_PROCESS_STATUS_NO_DATA, ErrorCode: "INFO-200"}, nil
	}}
	c := newCollector(t, func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, `{"RESULT":{}}`) }, testKey, st, p)

	sum, err := c.Run(context.Background(), store.TriggerSchedule)
	if err != nil {
		t.Fatal(err)
	}
	if sum != (Summary{NoData: 1}) || st.runs[0].status != "no_data" || st.runs[0].code != "INFO-200" || st.runs[0].trigger != "schedule" {
		t.Errorf("summary %v, run %+v", sum, *st.runs[0])
	}
}

// HTTP 오류 응답도 원본 응답을 남기도록 Rust에 넘기고, Rust 결과로 기록한다(KEDP001-D022 13).
func TestRunHTTPErrorIsProcessed(t *testing.T) {
	st := &fakeStore{targets: []store.Target{ecosTarget("a"), ecosTarget("b")}}
	p := &fakeProcessor{respond: func(in *kedpv1.ProcessPageRequest) (*kedpv1.ProcessPageResponse, error) {
		return &kedpv1.ProcessPageResponse{Status: kedpv1.ProcessStatus_PROCESS_STATUS_ERROR, ErrorCode: fmt.Sprintf("HTTP_%d", in.HttpStatus)}, nil
	}}
	c := newCollector(t, func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "bad gateway", 502) }, testKey, st, p)

	sum, err := c.Run(context.Background(), store.TriggerManual)
	if err != nil {
		t.Fatal(err)
	}
	// 지표 하나가 실패해도 다음 지표를 수집한다.
	if sum != (Summary{Error: 2}) || len(p.got) != 2 {
		t.Fatalf("summary %v, calls %d", sum, len(p.got))
	}
	if p.got[0].HttpStatus != 502 || string(p.got[0].Body) != "bad gateway\n" {
		t.Errorf("request = status %d, body %q", p.got[0].HttpStatus, p.got[0].Body)
	}
	if st.runs[0].status != "error" || st.runs[0].code != "HTTP_502" {
		t.Errorf("run = %+v", *st.runs[0])
	}
}

func TestRunWithoutRust(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		key     string
		target  store.Target
		code    string
	}{
		{"no key", nil, "", ecosTarget("a"), CodeNoAPIKey},
		{"response has key", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, testKey) }, testKey, ecosTarget("a"), CodeResponseHasKey},
		{"no chart period", nil, testKey, store.Target{Indicator: ecosTarget("a").Indicator}, CodeInvalidIndicator},
		{"unknown source", nil, testKey, store.Target{Indicator: config.Indicator{ID: "a", Source: "x"}}, CodeInvalidIndicator},
		{"unsupported cycle", nil, testKey, store.Target{
			Indicator: config.Indicator{ID: "a", Source: "kosis", Cycle: "D"}, FirstStart: date(2026, 1, 1)}, CodeInvalidIndicator},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := tt.handler
			if handler == nil {
				handler = func(http.ResponseWriter, *http.Request) { t.Error("provider was called") }
			}
			st := &fakeStore{targets: []store.Target{tt.target}}
			p := &fakeProcessor{respond: success(1)}
			c := newCollector(t, handler, tt.key, st, p)

			if _, err := c.Run(context.Background(), store.TriggerManual); err != nil {
				t.Fatal(err)
			}
			if len(p.got) != 0 {
				t.Errorf("ProcessPage was called")
			}
			if r := st.runs[0]; r.status != "error" || r.code != tt.code {
				t.Errorf("run = %+v, want error %s", *r, tt.code)
			}
		})
	}
}

func TestRunFetchFailed(t *testing.T) {
	st := &fakeStore{targets: []store.Target{ecosTarget("a")}}
	p := &fakeProcessor{respond: success(1)}
	c := newCollector(t, func(http.ResponseWriter, *http.Request) {}, testKey, st, p)
	// 닫힌 서버로 보내 응답을 받지 못하게 한다.
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()
	client := fetch.New(time.Second, testKey)
	if err := client.SetHost(srv.URL); err != nil {
		t.Fatal(err)
	}
	c.Sources["ecos"] = ecos.New(testKey, client)

	if _, err := c.Run(context.Background(), store.TriggerManual); err != nil {
		t.Fatal(err)
	}
	if r := st.runs[0]; r.status != "error" || r.code != CodeFetchFailed || len(p.got) != 0 {
		t.Errorf("run = %+v, calls %d", *r, len(p.got))
	}
}

// Rust가 결과를 주지 못하면 그 지표를 error로 남기고 남은 지표는 수집하지 않는다(KEDP001-D020).
func TestRunStopsWhenRustFails(t *testing.T) {
	st := &fakeStore{targets: []store.Target{ecosTarget("a"), ecosTarget("b")}}
	p := &fakeProcessor{respond: func(*kedpv1.ProcessPageRequest) (*kedpv1.ProcessPageResponse, error) {
		return nil, status.Error(codes.Unavailable, "dial tcp 127.0.0.1:50051: connection refused")
	}}
	c := newCollector(t, func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, ecosBody(1)) }, testKey, st, p)

	sum, err := c.Run(context.Background(), store.TriggerManual)
	if !errors.Is(err, ErrProcessor) {
		t.Fatalf("err = %v, want ErrProcessor", err)
	}
	if sum != (Summary{Error: 1}) || len(st.runs) != 1 {
		t.Fatalf("summary %v, runs %d", sum, len(st.runs))
	}
	if r := st.runs[0]; r.status != "error" || r.code != CodeProcessorFailed {
		t.Errorf("run = %+v", *r)
	}
}

func TestRunUnspecifiedStatus(t *testing.T) {
	st := &fakeStore{targets: []store.Target{ecosTarget("a")}}
	p := &fakeProcessor{respond: func(*kedpv1.ProcessPageRequest) (*kedpv1.ProcessPageResponse, error) {
		return &kedpv1.ProcessPageResponse{}, nil
	}}
	c := newCollector(t, func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, ecosBody(1)) }, testKey, st, p)

	if _, err := c.Run(context.Background(), store.TriggerManual); err != nil {
		t.Fatal(err)
	}
	if r := st.runs[0]; r.status != "error" || r.code != CodeProcessorFailed {
		t.Errorf("run = %+v", *r)
	}
}

func TestGRPCErrorHidesAddress(t *testing.T) {
	err := GRPCError(status.Error(codes.Unavailable, "dial tcp 127.0.0.1:50051: connection refused"))
	if strings.Contains(err.Error(), "127.0.0.1") {
		t.Errorf("err = %v", err)
	}
	err = GRPCError(status.Error(codes.NotFound, "indicator x not found"))
	if err.Error() != "gRPC NotFound: indicator x not found" {
		t.Errorf("err = %v", err)
	}
}

func TestRange(t *testing.T) {
	ecosSrc := ecos.New("", nil)
	kosisSrc := kosis.New("", nil)
	tests := []struct {
		name       string
		src        source.Source
		target     store.Target
		start, end string
	}{
		{"first: chart period", ecosSrc, ecosTarget("a"), "201610", "202610"},
		{"next: from last period", ecosSrc, store.Target{
			Indicator: ecosTarget("a").Indicator, LastPeriod: date(2026, 9, 1), FirstStart: date(2016, 10, 30)}, "202609", "202610"},
		{"kosis year", kosisSrc, store.Target{
			Indicator: config.Indicator{Source: "kosis", Cycle: "A"}, FirstStart: date(2006, 10, 30)}, "2006", "2026"},
	}
	for _, tt := range tests {
		start, end, err := Range(tt.src, tt.target, today)
		if err != nil || start != tt.start || end != tt.end {
			t.Errorf("%s: Range = %s, %s, %v; want %s, %s", tt.name, start, end, err, tt.start, tt.end)
		}
	}
}
