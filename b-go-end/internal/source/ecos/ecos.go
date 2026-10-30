// Package ecos는 한국은행 ECOS StatisticSearch 제공처다.
package ecos

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/schware/aidev-KEDP001/b-go-end/internal/fetch"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/source"
)

const (
	baseURL = "https://ecos.bok.or.kr/api"
	// 한 페이지의 행 수. 전체 행 수가 넘으면 나누어 받는다(KEDP001-D022 21).
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
// 요청 시작·끝 행 번호는 req.Page 페이지의 것이다.
// 분류(Classes)가 있으면 ITEM_CODE2, ITEM_CODE3… 자리에 차례로 붙인다.
func searchURL(key string, req source.Request) string {
	ind := req.Indicator
	page := req.PageNumber()
	u := fmt.Sprintf("%s/StatisticSearch/%s/json/kr/%d/%d/%s/%s/%s/%s/%s",
		baseURL, url.PathEscape(key), (page-1)*pageSize+1, page*pageSize,
		url.PathEscape(ind.StatCode), url.PathEscape(ind.Cycle),
		url.PathEscape(req.Start), url.PathEscape(req.End), url.PathEscape(ind.ItemCode))
	for _, c := range ind.Classes {
		u += "/" + url.PathEscape(c)
	}
	return u
}

// FetchItems는 통계표 하나의 StatisticItemList 원본 응답(JSON)을 돌려준다. 항목 코드를 확인할 때 쓴다.
func (s *Source) FetchItems(ctx context.Context, statCode string) ([]byte, error) {
	return s.Get(ctx, func(key string) string { return itemListURL(key, statCode) })
}

// itemListURL은 키가 들어간 요청 주소를 만든다. 이 주소는 fetch.Client를 거쳐서만 기록한다.
func itemListURL(key, statCode string) string {
	return fmt.Sprintf("%s/StatisticItemList/%s/json/kr/1/%d/%s",
		baseURL, url.PathEscape(key), pageSize, url.PathEscape(statCode))
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

// LastPage는 list_total_count로 마지막 페이지 번호를 돌려준다. 읽을 수 없으면 1이다.
func (s *Source) LastPage(body []byte) int {
	var r searchResponse
	if json.Unmarshal(body, &r) != nil || r.StatisticSearch == nil {
		return 1
	}
	return source.Pages(r.StatisticSearch.ListTotalCount, pageSize)
}

// Period는 t가 들어 있는 기간을 ECOS 요청 형식으로 돌려준다(설계서 5.1).
// D 20261001, M 202401, Q 2025Q1, A 2023
func (s *Source) Period(cycle string, t time.Time) (string, error) {
	switch cycle {
	case "D":
		return t.Format("20060102"), nil
	case "M":
		return t.Format("200601"), nil
	case "Q":
		return fmt.Sprintf("%dQ%d", t.Year(), source.Quarter(t)), nil
	case "A":
		return t.Format("2006"), nil
	}
	return "", source.UnsupportedCycle(s.Name(), cycle)
}

// noDataCode는 "해당하는 데이터가 없음" 오류 코드다.
const noDataCode = "INFO-200"

// Parse는 StatisticSearch 응답(한 페이지)을 읽는다. ECOS 오류 응답이면 *source.APIError를 돌려준다(예: INFO-100 인증키 오류).
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
	return r.StatisticSearch.Row, nil
}
