// Package source는 제공처(ECOS, KOSIS, 공공데이터포털)가 따르는 공통 약속을 둔다.
package source

import (
	"context"

	"github.com/schware/aidev-KEDP001/b-go-end/internal/config"
)

// Request는 지표 하나와 조회 기간이다. 기간 형식은 지표 주기를 따른다(예: M이면 202401).
type Request struct {
	Indicator config.Indicator
	Start     string
	End       string
}

// Source는 지표 정보를 받아 원본 응답을 돌려준다.
type Source interface {
	Name() string
	Fetch(ctx context.Context, req Request) ([]byte, error)
}
