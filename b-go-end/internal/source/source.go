// Package source는 제공처(ECOS, KOSIS, 공공데이터포털)가 따르는 공통 약속을 둔다.
package source

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/schware/aidev-KEDP001/b-go-end/internal/config"
)

// Request는 지표 하나와 조회 기간이다. 기간 형식은 지표 주기를 따른다(예: M이면 202401).
type Request struct {
	Indicator config.Indicator
	Start     string
	End       string
}

// SampleName은 이 요청의 응답 샘플 파일 이름이다.
// 통계 코드-항목 코드-분류 코드…-주기-시작-끝.json (분류 코드가 없으면 빠진다)
func (r Request) SampleName() string {
	ind := r.Indicator
	parts := append([]string{ind.StatCode, ind.ItemCode}, ind.Classes...)
	parts = append(parts, ind.Cycle, r.Start, r.End)
	return strings.Join(parts, "-") + ".json"
}

// Source는 지표 정보를 받아 원본 응답을 돌려준다.
type Source interface {
	Name() string
	Fetch(ctx context.Context, req Request) ([]byte, error)
	// Check는 원본 응답이 정상 응답인지 확인하고 행 수를 돌려준다. 오류 응답이면 *APIError를 돌려준다.
	Check(body []byte) (int, error)
}

// APIError는 제공처가 응답 본문으로 돌려준 오류다. 코드는 제공처마다 다르다.
type APIError struct {
	Source  string
	Code    string
	Message string
	// NoData는 "해당하는 데이터가 없음" 오류인지 나타낸다(ECOS INFO-200, KOSIS 30).
	NoData bool
}

func (e *APIError) Error() string { return fmt.Sprintf("%s: %s %s", e.Source, e.Code, e.Message) }

// NoData는 err가 "해당하는 데이터가 없음" 오류인지 알려 준다.
func NoData(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.NoData
}

// RequireKey는 키가 비어 있으면 오류를 돌려준다. 오류에는 환경 변수 이름만 넣는다.
func RequireKey(source, envName, key string) error {
	if key == "" {
		return fmt.Errorf("%s: %s is not set", source, envName)
	}
	return nil
}
