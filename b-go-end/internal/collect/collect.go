// Package collect는 DB indicator의 지표를 수집해 Rust 처리 서버로 넘긴다(KEDP001-D019, D022, b-proto/README.md 흐름).
//
// 지표마다: 수집 기록을 running으로 넣기 → 수집 범위 정하기 → 페이지마다 제공처 응답 받기 → Rust ProcessPage → 수집 기록 고치기
package collect

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"
	_ "time/tzdata" // 서버에 시간대 정보가 없어도 Asia/Seoul을 읽는다

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/schware/aidev-KEDP001/b-go-end/internal/fetch"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/kedpv1"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/source"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/store"
)

// Go가 정한 collection_run.error_code다. Rust에 응답을 넘기지 못했거나 Rust가 결과를 주지 못했을 때 쓴다.
// 제공처 코드와 Rust 코드(HTTP_<상태 코드>, INVALID_RESPONSE)는 b-proto/README.md를 본다.
const (
	// 지표 정의로 수집 범위를 정할 수 없다(화면 설정 없음, 제공처가 받지 않는 주기, 모르는 제공처)
	CodeInvalidIndicator = "INVALID_INDICATOR"
	// .env에 키가 없어 요청하지 않았다
	CodeNoAPIKey = "NO_API_KEY"
	// 응답을 받지 못했다(시간 초과, 연결 실패)
	CodeFetchFailed = "FETCH_FAILED"
	// 응답 본문에 키가 들어 있어 Rust에 넘기지 않았다
	CodeResponseHasKey = "RESPONSE_HAS_KEY"
	// Rust에 연결하지 못했거나, Rust가 gRPC 오류 또는 알 수 없는 결과를 돌려줬다
	CodeProcessorFailed = "PROCESSOR_FAILED"
)

// processTimeout은 ProcessPage 한 번의 시간 제한이다.
const processTimeout = time.Minute

// Seoul은 수집 시각과 "오늘"의 시간대다(KEDP001-D019).
var Seoul = mustLoadLocation("Asia/Seoul")

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

// Store는 Collector가 쓰는 DB 일이다(*store.Store).
type Store interface {
	Targets(ctx context.Context, today time.Time) ([]store.Target, error)
	CollectTime(ctx context.Context) (string, error)
	StartRun(ctx context.Context, sourceCode, indicatorID, trigger string) (int64, error)
	FinishRun(ctx context.Context, id int64, status, errorCode string, rowCount int) error
}

// Processor는 Rust 처리 서버의 ProcessPage다(kedpv1.ProcessorServiceClient).
type Processor interface {
	ProcessPage(ctx context.Context, in *kedpv1.ProcessPageRequest, opts ...grpc.CallOption) (*kedpv1.ProcessPageResponse, error)
}

// Collector는 DB indicator의 지표를 수집한다.
type Collector struct {
	Store     Store
	Processor Processor
	// Sources는 제공처 코드(indicator.source_code)별 제공처다.
	Sources map[string]source.Source
	// HasKey는 응답 본문에 키가 들어 있는지 알려 준다(fetch.Client.HasKey). 들어 있으면 Rust에 넘기지 않는다.
	HasKey func(body []byte) bool
	// Now는 지금 시각이다. 비어 있으면 time.Now다(시험용).
	Now func() time.Time
}

func (c *Collector) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

// Summary는 수집 한 번의 지표별 결과 수다.
type Summary struct{ Success, NoData, Error int }

func (s Summary) String() string {
	return fmt.Sprintf("success %d, no_data %d, error %d", s.Success, s.NoData, s.Error)
}

// ErrProcessor는 Rust 처리 서버가 결과를 주지 못해 수집을 멈췄다는 오류다(KEDP001-D020).
var ErrProcessor = errors.New("processor failed; collection stopped")

// Run은 DB indicator의 지표를 모두 수집한다. trigger는 store.TriggerSchedule 또는 store.TriggerManual이다.
// 지표 하나가 실패해도(제공처 오류 등) 다음 지표를 수집한다.
// Rust가 결과를 주지 못하면 그 지표를 error로 기록하고 남은 지표는 수집하지 않는다(ErrProcessor).
// DB에 쓰지 못하면 멈추고 오류를 돌려준다.
func (c *Collector) Run(ctx context.Context, trigger string) (Summary, error) {
	var sum Summary
	today := c.now().In(Seoul)
	targets, err := c.Store.Targets(ctx, today)
	if err != nil {
		return sum, err
	}
	for _, t := range targets {
		status, err := c.collect(ctx, t, today, trigger)
		switch status {
		case store.StatusSuccess:
			sum.Success++
		case store.StatusNoData:
			sum.NoData++
		case store.StatusError:
			sum.Error++
		}
		if err != nil {
			return sum, err
		}
	}
	return sum, nil
}

// result는 지표 하나의 수집 결과다. collection_run에 쓴다.
type result struct {
	status, code string
	rows         int
	// detail은 기록(log)에만 쓰는 설명이다.
	detail string
}

func failed(code, detail string) result {
	return result{status: store.StatusError, code: code, detail: detail}
}

// collect는 지표 하나를 수집하고 수집 기록을 남긴다. 돌려주는 오류는 수집을 멈춰야 하는 것(DB, Rust)이다.
func (c *Collector) collect(ctx context.Context, t store.Target, today time.Time, trigger string) (string, error) {
	ind := t.Indicator
	runID, err := c.Store.StartRun(ctx, ind.Source, ind.ID, trigger)
	if err != nil {
		return "", err
	}
	r, stop := c.pages(ctx, runID, t, today)
	if err := c.Store.FinishRun(ctx, runID, r.status, r.code, r.rows); err != nil {
		return "", err
	}
	log.Printf("collect %s run %d: %s %s rows %d %s", ind.ID, runID, r.status, r.code, r.rows, r.detail)
	if stop {
		return r.status, ErrProcessor
	}
	return r.status, nil
}

// pages는 수집 범위의 페이지를 차례로 받아 Rust에 넘긴다. stop은 Rust가 결과를 주지 못했다는 뜻이다.
func (c *Collector) pages(ctx context.Context, runID int64, t store.Target, today time.Time) (r result, stop bool) {
	ind := t.Indicator
	src, ok := c.Sources[ind.Source]
	if !ok {
		return failed(CodeInvalidIndicator, fmt.Sprintf("source %q is not supported", ind.Source)), false
	}
	start, end, err := Range(src, t, today)
	if err != nil {
		return failed(CodeInvalidIndicator, err.Error()), false
	}

	r = result{status: store.StatusSuccess}
	r.detail = fmt.Sprintf("(%s-%s)", start, end)
	var inserted, updated, unchanged, empty int32
	last := 1
	for page := 1; page <= last; page++ {
		ex, err := fetchPage(ctx, src, source.Request{Indicator: ind, Start: start, End: end, Page: page})
		if ex == nil {
			var keyErr *source.KeyError
			if errors.As(err, &keyErr) {
				return failed(CodeNoAPIKey, err.Error()), false
			}
			return failed(CodeFetchFailed, fmt.Sprintf("page %d: %v", page, err)), false
		}
		if c.HasKey(ex.Body) {
			return failed(CodeResponseHasKey, fmt.Sprintf("page %d", page)), false
		}

		resp, err := c.process(ctx, runID, ind.ID, page, ex)
		if err != nil {
			return failed(CodeProcessorFailed, fmt.Sprintf("page %d: %v", page, err)), true
		}
		switch resp.GetStatus() {
		case kedpv1.ProcessStatus_PROCESS_STATUS_SUCCESS:
			inserted += resp.GetInsertedCount()
			updated += resp.GetUpdatedCount()
			unchanged += resp.GetUnchangedCount()
			empty += resp.GetSkippedEmptyCount()
		case kedpv1.ProcessStatus_PROCESS_STATUS_NO_DATA:
			if page == 1 {
				return result{status: store.StatusNoData, code: resp.GetErrorCode(), detail: r.detail}, false
			}
			// 다음 페이지가 비었다(받는 사이에 전체 행 수가 줄었다). 받은 데까지로 끝낸다.
			last = page
		default:
			// ERROR, 그리고 쓰지 않는 UNSPECIFIED(b-proto/README.md)
			code := resp.GetErrorCode()
			if code == "" {
				code = CodeProcessorFailed
			}
			return failed(code, fmt.Sprintf("page %d: %s %s", page, resp.GetStatus(), resp.GetErrorMessage())), false
		}
		if page == 1 {
			last = src.LastPage(ex.Body)
		}
	}
	r.rows = int(inserted + updated + unchanged)
	r.detail += fmt.Sprintf(" pages %d (inserted %d, updated %d, unchanged %d, empty %d)",
		last, inserted, updated, unchanged, empty)
	return r, false
}

// fetchPage는 페이지 하나를 요청하고 받은 응답 기록을 돌려준다.
// 상태 코드가 200이 아니거나 제공처 오류 응답이어도 응답을 받았으면 기록을 돌려준다. 원본 응답은 Rust가 남긴다(KEDP001-D022 13).
// 응답을 받지 못했으면 nil과 오류다.
func fetchPage(ctx context.Context, src source.Source, req source.Request) (*fetch.Exchange, error) {
	var ex *fetch.Exchange
	_, err := src.Fetch(fetch.WithRecorder(ctx, func(e fetch.Exchange) { ex = &e }), req)
	return ex, err
}

// process는 받은 응답 하나를 Rust ProcessPage로 넘긴다.
func (c *Collector) process(ctx context.Context, runID int64, indicatorID string, page int, ex *fetch.Exchange) (*kedpv1.ProcessPageResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, processTimeout)
	defer cancel()
	resp, err := c.Processor.ProcessPage(ctx, &kedpv1.ProcessPageRequest{
		RunId:         runID,
		IndicatorId:   indicatorID,
		Page:          int32(page),
		ReceivedAt:    timestamppb.New(ex.ReceivedAt),
		HttpStatus:    int32(ex.Status),
		RequestMasked: ex.URL,
		Body:          ex.Body,
	})
	if err != nil {
		return nil, GRPCError(err)
	}
	return resp, nil
}

// GRPCError는 gRPC 오류를 기록해도 되는 글로 바꾼다.
// 연결하지 못한 오류(UNAVAILABLE)의 메시지에는 Rust 주소가 들어 있어 상태 이름만 쓴다.
func GRPCError(err error) error {
	st := status.Convert(err)
	if st.Code() == codes.Unavailable {
		return fmt.Errorf("gRPC %s", st.Code())
	}
	return fmt.Errorf("gRPC %s: %s", st.Code(), st.Message())
}

// Range는 수집 범위를 제공처 요청 형식으로 돌려준다.
// 관측값이 있으면 마지막 관측값의 기간부터, 없으면 오늘에서 차트 기간을 뺀 날부터(KEDP001-D022 12) 오늘까지다.
// 마지막 기간을 다시 받는 것은 값이 고쳐졌을 수 있어서다(KEDP001-D022 4).
func Range(src source.Source, t store.Target, today time.Time) (start, end string, err error) {
	from := t.LastPeriod
	if from == nil {
		from = t.FirstStart
	}
	if from == nil {
		return "", "", errors.New("indicator_display has no chart_period")
	}
	cycle := t.Indicator.Cycle
	if start, err = src.Period(cycle, *from); err != nil {
		return "", "", err
	}
	if end, err = src.Period(cycle, today); err != nil {
		return "", "", err
	}
	return start, end, nil
}
