package collect

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/schware/aidev-KEDP001/b-go-end/internal/fetch"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/kedpv1"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/logx"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/source/ecos"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/store"
)

func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	t.Cleanup(logx.SetOutput(&buf, slog.LevelDebug))
	return &buf
}

// wantLines는 로그가 patterns와 차례로 맞는 줄을 (사이에 다른 줄이 있어도) 가졌는지 본다.
func wantLines(t *testing.T, logs string, patterns ...string) {
	t.Helper()
	rest := strings.Split(strings.TrimSpace(logs), "\n")
	for _, p := range patterns {
		re := regexp.MustCompile(p)
		found := false
		for len(rest) > 0 {
			line := rest[0]
			rest = rest[1:]
			if re.MatchString(line) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("no line %q in:\n%s", p, logs)
			return
		}
	}
}

func TestRunLogSuccess(t *testing.T) {
	logs := captureLog(t)
	st := &fakeStore{targets: []store.Target{ecosTarget("ecos-base-rate")}}
	c := newCollector(t, func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, ecosBody(1)) }, testKey, st, &fakeProcessor{respond: success(1)})
	if _, err := c.Run(context.Background(), store.TriggerManual); err != nil {
		t.Fatal(err)
	}
	wantLines(t, logs.String(),
		`level=INFO comp=go event=collect\.start trigger=manual indicators=1$`,
		`level=DEBUG comp=go event=fetch\.request run=1 indicator=ecos-base-rate page=1 url=\S*\*\*\*\S* ms=\d+ status=200$`,
		`level=INFO comp=go event=collect\.indicator run=1 indicator=ecos-base-rate source=ecos status=success code="" rows=\d+ detail=`,
		`level=INFO comp=go event=collect\.finish trigger=manual success=1 no_data=0 error=0 ms=\d+$`,
	)
	if strings.Contains(logs.String(), testKey) {
		t.Errorf("log has key: %s", logs.String())
	}
}

// 응답을 받지 못하면 fetch.request와 collect.indicator는 WARN, 수집 끝은 일부 실패라 WARN이다.
func TestRunLogFetchFailed(t *testing.T) {
	logs := captureLog(t)
	st := &fakeStore{targets: []store.Target{ecosTarget("a")}}
	c := newCollector(t, func(http.ResponseWriter, *http.Request) {}, testKey, st, &fakeProcessor{respond: success(1)})
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
	wantLines(t, logs.String(),
		`level=WARN comp=go event=fetch\.request run=1 indicator=a page=1 url=\S+ ms=\d+ err=`,
		`level=WARN comp=go event=collect\.indicator run=1 indicator=a source=ecos status=error code=FETCH_FAILED `,
		`level=WARN comp=go event=collect\.finish trigger=manual success=0 no_data=0 error=1 ms=\d+$`,
	)
}

// Rust가 결과를 주지 못해 수집이 멈추면 collect.finish는 ERROR다. ERROR는 그 한 줄뿐이다.
func TestRunLogStopsWhenRustFails(t *testing.T) {
	logs := captureLog(t)
	st := &fakeStore{targets: []store.Target{ecosTarget("a"), ecosTarget("b")}}
	p := &fakeProcessor{respond: func(*kedpv1.ProcessPageRequest) (*kedpv1.ProcessPageResponse, error) {
		return nil, status.Error(codes.Unavailable, "dial tcp 127.0.0.1:50051: connection refused")
	}}
	c := newCollector(t, func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, ecosBody(1)) }, testKey, st, p)
	if _, err := c.Run(context.Background(), store.TriggerManual); err == nil {
		t.Fatal("want error")
	}
	wantLines(t, logs.String(),
		`level=WARN comp=go event=collect\.indicator run=1 indicator=a source=ecos status=error code=PROCESSOR_FAILED `,
		`level=ERROR comp=go event=collect\.finish trigger=manual success=0 no_data=0 error=1 ms=\d+ err="processor failed; collection stopped"$`,
	)
	if n := strings.Count(logs.String(), "level=ERROR"); n != 1 {
		t.Errorf("ERROR lines = %d:\n%s", n, logs.String())
	}
	if strings.Contains(logs.String(), "50051") {
		t.Errorf("log has Rust address: %s", logs.String())
	}
}

func TestLogFinishCanceled(t *testing.T) {
	logs := captureLog(t)
	logFinish(context.Background(), store.TriggerSchedule, Summary{Success: 2}, context.Canceled, time.Second)
	wantLines(t, logs.String(), `level=WARN comp=go event=collect\.finish trigger=schedule success=2 no_data=0 error=0 ms=1000 err=canceled$`)
}
