// Package datagokr는 공공데이터포털 금융위원회_증권상품시세정보(getETFPriceInfo_V2) 제공처다.
package datagokr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/schware/aidev-KEDP001/b-go-end/internal/fetch"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/source"
)

const (
	baseURL = "https://apis.data.go.kr/1160100/GetSecuritiesProductInfoService_V2/getETFPriceInfo_V2"
	// 한 페이지의 행 수. 전체 행 수가 넘으면 나누어 받는다(KEDP001-D022 21).
	pageSize = 1000
	// 기간(basDt) 형식
	dateLayout = "20060102"
	// okCode는 정상 응답의 resultCode다.
	okCode = "00"
)

// Source는 공공데이터포털 제공처다.
type Source struct{ source.Base }

var _ source.Source = (*Source)(nil)

// New는 공공데이터포털 제공처를 만든다. client는 key를 가리도록 만든 것이어야 한다.
// key는 포털이 주는 Decoding 키와 Encoding 키(이미 URL 인코딩된 형태) 모두 받는다.
func New(key string, client *fetch.Client) *Source {
	return &Source{source.NewBase("data-go-kr", "DATA_GO_KR_API_KEY", decodeKey(key), client)}
}

// decodeKey는 Encoding 키를 원래 형태로 푼다. 주소를 만들 때 다시 인코딩하므로 두 번 인코딩되지 않게 한다.
// 원래 형태에는 %가 없으므로 %가 있을 때만 푼다(원래 형태의 +를 공백으로 바꾸지 않기 위해).
func decodeKey(key string) string {
	if !strings.Contains(key, "%") {
		return key
	}
	if raw, err := url.QueryUnescape(key); err == nil {
		return raw
	}
	return key
}

// Fetch는 getETFPriceInfo_V2 원본 응답(JSON)을 돌려준다.
// 인증키 오류 등은 HTTP 4xx와 함께 본문으로 오므로, 그 본문을 읽어 *source.APIError로 돌려준다.
func (s *Source) Fetch(ctx context.Context, req source.Request) ([]byte, error) {
	endBasDt, err := endParam(req)
	if err != nil {
		return nil, err
	}
	body, err := s.Get(ctx, func(key string) string { return priceURL(key, req, endBasDt) })
	if err != nil {
		return nil, readStatusError(err)
	}
	return body, nil
}

// readStatusError는 HTTP 오류 응답 본문에 제공처 오류가 있으면 *source.APIError로 바꾼다. 없으면 err 그대로다.
func readStatusError(err error) error {
	var statusErr *fetch.StatusError
	if !errors.As(err, &statusErr) {
		return err
	}
	var apiErr *source.APIError
	if _, perr := Parse(statusErr.Body); errors.As(perr, &apiErr) {
		return fmt.Errorf("%w (status %d)", apiErr, statusErr.Code)
	}
	return err
}

// endParam은 요청 기간을 확인하고 endBasDt 값을 돌려준다.
// 요청의 Start·End는 둘 다 포함하는 기간(YYYYMMDD)이다. API의 endBasDt는 그 날을 포함하지 않으므로 하루를 더한다.
func endParam(req source.Request) (string, error) {
	if _, err := time.Parse(dateLayout, req.Start); err != nil {
		return "", fmt.Errorf("data-go-kr: start %q is not YYYYMMDD", req.Start)
	}
	end, err := time.Parse(dateLayout, req.End)
	if err != nil {
		return "", fmt.Errorf("data-go-kr: end %q is not YYYYMMDD", req.End)
	}
	return end.AddDate(0, 0, 1).Format(dateLayout), nil
}

// priceURL은 키가 들어간 요청 주소를 만든다. 이 주소는 fetch.Client를 거쳐서만 기록한다.
func priceURL(key string, req source.Request, endBasDt string) string {
	q := url.Values{}
	q.Set("serviceKey", key)
	q.Set("resultType", "json")
	q.Set("numOfRows", strconv.Itoa(pageSize))
	q.Set("pageNo", strconv.Itoa(req.PageNumber()))
	// likeSrtnCd는 부분 일치 검색이다. 6자리 종목 코드로는 그 종목만 온다(2026-10-25 세 종목 확인).
	q.Set("likeSrtnCd", req.Indicator.StatCode)
	q.Set("beginBasDt", req.Start)
	q.Set("endBasDt", endBasDt)
	return baseURL + "?" + q.Encode()
}

// LastPage는 totalCount로 마지막 페이지 번호를 돌려준다. 읽을 수 없으면 1이다.
func (s *Source) LastPage(body []byte) int {
	var r priceResponse
	if json.Unmarshal(body, &r) != nil || r.Response == nil {
		return 1
	}
	return source.Pages(r.Response.Body.TotalCount, pageSize)
}

// Period는 t를 기간(basDt) 형식으로 돌려준다. 이 API에는 일 주기만 있다.
func (s *Source) Period(cycle string, t time.Time) (string, error) {
	if cycle != "D" {
		return "", source.UnsupportedCycle(s.Name(), cycle)
	}
	return t.Format(dateLayout), nil
}

// Row는 getETFPriceInfo_V2 응답의 한 행(종목 하나의 하루)이다. 값은 응답 그대로 문자열로 둔다.
type Row struct {
	BasDt      string `json:"basDt"`       // 기준일자
	SrtnCd     string `json:"srtnCd"`      // 단축코드(종목 코드)
	IsinCd     string `json:"isinCd"`      // ISIN 코드
	ItmsNm     string `json:"itmsNm"`      // 종목명
	Clpr       string `json:"clpr"`        // 종가
	Vs         string `json:"vs"`          // 전일 대비
	FltRt      string `json:"fltRt"`       // 등락률
	Nav        string `json:"nav"`         // 순자산가치(NAV)
	Mkp        string `json:"mkp"`         // 시가
	Hipr       string `json:"hipr"`        // 고가
	Lopr       string `json:"lopr"`        // 저가
	Trqu       string `json:"trqu"`        // 거래량
	TrPrc      string `json:"trPrc"`       // 거래대금
	MrktTotAmt string `json:"mrktTotAmt"`  // 시가총액
	StLstgCnt  string `json:"stLstgCnt"`   // 상장좌수
	BssIdxNm   string `json:"bssIdxIdxNm"` // 기초지수명
	BssIdxClpr string `json:"bssIdxClpr"`  // 기초지수 종가
	NPptTotAmt string `json:"nPptTotAmt"`  // 순자산총액
}

type priceResponse struct {
	Response *struct {
		Header struct {
			ResultCode string `json:"resultCode"`
			ResultMsg  string `json:"resultMsg"`
		} `json:"header"`
		Body struct {
			TotalCount int `json:"totalCount"`
			Items      struct {
				Item []Row `json:"item"`
			} `json:"items"`
		} `json:"body"`
	} `json:"response"`
	// 게이트웨이 오류(인증키, 서비스 이름 등). HTTP 200이나 4xx와 함께 온다.
	ServiceError *struct {
		Header struct {
			ErrMsg           string `json:"errMsg"`
			ReturnAuthMsg    string `json:"returnAuthMsg"`
			ReturnReasonCode string `json:"returnReasonCode"`
		} `json:"cmmMsgHeader"`
	} `json:"OpenAPI_ServiceResponse"`
}

// Parse는 getETFPriceInfo_V2 응답(한 페이지)을 읽는다. 오류 응답이면 *source.APIError를 돌려준다(예: 30 등록되지 않은 서비스키).
// 데이터가 없으면 이 API는 정상 코드(00)와 totalCount 0을 주는데, 다른 제공처와 맞추려고 NoData 오류로 돌려준다.
func Parse(body []byte) ([]Row, error) {
	var r priceResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("data-go-kr: parse response: %w", err)
	}
	if e := r.ServiceError; e != nil {
		return nil, &source.APIError{Source: "data-go-kr", Code: e.Header.ReturnReasonCode,
			Message: e.Header.ErrMsg + " " + e.Header.ReturnAuthMsg}
	}
	if r.Response == nil {
		return nil, errors.New("data-go-kr: response has neither response nor OpenAPI_ServiceResponse")
	}
	h, b := r.Response.Header, r.Response.Body
	if h.ResultCode != okCode {
		return nil, &source.APIError{Source: "data-go-kr", Code: h.ResultCode, Message: h.ResultMsg}
	}
	if b.TotalCount == 0 {
		return nil, &source.APIError{Source: "data-go-kr", Code: h.ResultCode, Message: "totalCount 0", NoData: true}
	}
	return b.Items.Item, nil
}
