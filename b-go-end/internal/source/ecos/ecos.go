// Package ecos는 한국은행 ECOS StatisticSearch 제공처다.
package ecos

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"github.com/schware/aidev-KEDP001/b-go-end/internal/fetch"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/source"
)

const (
	baseURL = "https://ecos.bok.or.kr/api"
	// 한 번에 받는 최대 행 수. 넘으면 Parse가 오류를 낸다(페이지 나누기는 아직 없다).
	pageSize = 1000
)

// Source는 ECOS 제공처다.
type Source struct{ source.Base }

var _ source.Source = (*Source)(nil)

// New는 ECOS 제공처를 만든다. client는 key를 가리도록 만든 것이어야 한다.
func New(key string, client *fetch.Client) *Source {
	return &Source{source.NewBase("ecos", "ECOS_API_KEY", key, client)}
}

// Fetch는 StatisticSearch 원본 응답(JSON)을 돌려준다.
func (s *Source) Fetch(ctx context.Context, req source.Request) ([]byte, error) {
	return s.Get(ctx, func(key string) string { return searchURL(key, req) })
}

// searchURL은 키가 들어간 요청 주소를 만든다. 이 주소는 fetch.Client를 거쳐서만 기록한다.
func searchURL(key string, req source.Request) string {
	ind := req.Indicator
	return fmt.Sprintf("%s/StatisticSearch/%s/json/kr/1/%d/%s/%s/%s/%s/%s",
		baseURL, url.PathEscape(key), pageSize,
		url.PathEscape(ind.StatCode), url.PathEscape(ind.Cycle),
		url.PathEscape(req.Start), url.PathEscape(req.End), url.PathEscape(ind.ItemCode))
}

// Row는 StatisticSearch 응답의 한 행이다. 값은 응답 그대로 문자열로 둔다.
type Row struct {
	StatCode  string `json:"STAT_CODE"`
	StatName  string `json:"STAT_NAME"`
	ItemCode1 string `json:"ITEM_CODE1"`
	ItemName1 string `json:"ITEM_NAME1"`
	UnitName  string `json:"UNIT_NAME"`
	Time      string `json:"TIME"`
	DataValue string `json:"DATA_VALUE"`
}

type searchResponse struct {
	StatisticSearch *struct {
		ListTotalCount int   `json:"list_total_count"`
		Row            []Row `json:"row"`
	} `json:"StatisticSearch"`
	Result *struct {
		Code    string `json:"CODE"`
		Message string `json:"MESSAGE"`
	} `json:"RESULT"`
}

// Check는 정상 응답인지 확인하고 행 수를 돌려준다.
func (s *Source) Check(body []byte) (int, error) {
	return source.Count(Parse(body))
}

// noDataCode는 "해당하는 데이터가 없음" 오류 코드다.
const noDataCode = "INFO-200"

// Parse는 StatisticSearch 응답을 읽는다. ECOS 오류 응답이면 *source.APIError를 돌려준다(예: INFO-100 인증키 오류).
func Parse(body []byte) ([]Row, error) {
	var r searchResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("ecos: parse response: %w", err)
	}
	if r.Result != nil {
		return nil, &source.APIError{Source: "ecos", Code: r.Result.Code, Message: r.Result.Message, NoData: r.Result.Code == noDataCode}
	}
	if r.StatisticSearch == nil {
		return nil, errors.New("ecos: response has neither StatisticSearch nor RESULT")
	}
	if err := source.RequireAllRows("ecos", len(r.StatisticSearch.Row), r.StatisticSearch.ListTotalCount); err != nil {
		return nil, err
	}
	return r.StatisticSearch.Row, nil
}
