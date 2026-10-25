// Package kosis는 KOSIS 통계자료(statisticsParameterData) 제공처다.
package kosis

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"github.com/schware/aidev-KEDP001/b-go-end/internal/fetch"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/source"
)

const (
	baseURL = "https://kosis.kr/openapi/Param/statisticsParameterData.do"
	metaURL = "https://kosis.kr/openapi/statisticsData.do"
	// 조사 대상 통계표는 모두 통계청(101)이다.
	orgID = "101"
	// noDataCode는 "데이터가 존재하지 않습니다" 오류 코드다.
	noDataCode = "30"
)

// Source는 KOSIS 제공처다.
type Source struct{ source.Base }

var _ source.Source = (*Source)(nil)

// New는 KOSIS 제공처를 만든다. client는 key를 가리도록 만든 것이어야 한다.
func New(key string, client *fetch.Client) *Source {
	return &Source{source.NewBase("kosis", "KOSIS_API_KEY", key, client)}
}

// Fetch는 statisticsParameterData 원본 응답(JSON)을 돌려준다.
func (s *Source) Fetch(ctx context.Context, req source.Request) ([]byte, error) {
	return s.Get(ctx, func(key string) string { return dataURL(key, req) })
}

// dataURL은 키가 들어간 요청 주소를 만든다. 이 주소는 fetch.Client를 거쳐서만 기록한다.
func dataURL(key string, req source.Request) string {
	ind := req.Indicator
	q := url.Values{}
	q.Set("method", "getList")
	q.Set("apiKey", key)
	q.Set("format", "json")
	// jsonVD=Y가 없으면 key에 따옴표가 없는 JSON이 온다.
	q.Set("jsonVD", "Y")
	q.Set("orgId", orgID)
	q.Set("tblId", ind.StatCode)
	q.Set("itmId", ind.ItemCode)
	for i, c := range ind.Classes {
		q.Set(fmt.Sprintf("objL%d", i+1), c)
	}
	q.Set("prdSe", ind.Cycle)
	q.Set("startPrdDe", req.Start)
	q.Set("endPrdDe", req.End)
	return baseURL + "?" + q.Encode()
}

// FetchMeta는 통계표 하나의 getMeta 원본 응답(JSON)을 돌려준다. metaType은 요청의 type 값 그대로다(예: ITM, TBL, PRD).
func (s *Source) FetchMeta(ctx context.Context, tblID, metaType string) ([]byte, error) {
	return s.Get(ctx, func(key string) string { return getMetaURL(key, tblID, metaType) })
}

// getMetaURL은 키가 들어간 요청 주소를 만든다. 이 주소는 fetch.Client를 거쳐서만 기록한다.
func getMetaURL(key, tblID, metaType string) string {
	q := url.Values{}
	q.Set("method", "getMeta")
	q.Set("type", metaType)
	q.Set("apiKey", key)
	q.Set("format", "json")
	q.Set("jsonVD", "Y")
	q.Set("orgId", orgID)
	q.Set("tblId", tblID)
	return metaURL + "?" + q.Encode()
}

// Check는 정상 응답인지 확인하고 행 수를 돌려준다.
func (s *Source) Check(body []byte) (int, error) {
	return source.Count(Parse(body))
}

// Row는 statisticsParameterData 응답의 한 행이다. 값은 응답 그대로 문자열로 둔다.
// 분류는 지금 조사한 통계표에 하나(C1)만 있어 C1만 읽는다.
type Row struct {
	OrgID    string `json:"ORG_ID"`
	TblID    string `json:"TBL_ID"`
	TblNm    string `json:"TBL_NM"`
	ItmID    string `json:"ITM_ID"`
	ItmNm    string `json:"ITM_NM"`
	C1       string `json:"C1"`
	C1Nm     string `json:"C1_NM"`
	C1ObjNm  string `json:"C1_OBJ_NM"`
	UnitNm   string `json:"UNIT_NM"` // 통계표에 따라 없다(예: 출생아 수)
	PrdSe    string `json:"PRD_SE"`  // 요청의 Y는 응답에서 A로 온다
	PrdDe    string `json:"PRD_DE"`
	DT       string `json:"DT"`
	LstChnDe string `json:"LST_CHN_DE"`
}

type errorResponse struct {
	Err    string `json:"err"`
	ErrMsg string `json:"errMsg"`
}

// Parse는 statisticsParameterData 응답을 읽는다.
// 정상 응답은 행 배열이고, 오류 응답은 {"err","errMsg"} 객체다. 오류면 *source.APIError를 돌려준다(예: 11 인증키 오류).
func Parse(body []byte) ([]Row, error) {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) > 0 && trimmed[0] == '{' {
		var e errorResponse
		if err := json.Unmarshal(trimmed, &e); err != nil {
			return nil, fmt.Errorf("kosis: parse response: %w", err)
		}
		if e.Err == "" {
			return nil, errors.New("kosis: response is neither rows nor err")
		}
		return nil, &source.APIError{Source: "kosis", Code: e.Err, Message: e.ErrMsg, NoData: e.Err == noDataCode}
	}
	var rows []Row
	if err := json.Unmarshal(trimmed, &rows); err != nil {
		return nil, fmt.Errorf("kosis: parse response: %w", err)
	}
	return rows, nil
}
