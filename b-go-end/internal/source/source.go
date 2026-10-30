// Package source는 제공처(ECOS, KOSIS, 공공데이터포털)가 따르는 공통 약속을 둔다.
package source

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/schware/aidev-KEDP001/b-go-end/internal/config"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/fetch"
)

// Request는 지표 하나와 조회 기간이다. 기간 형식은 지표 주기를 따른다(예: M이면 202401).
type Request struct {
	Indicator config.Indicator
	Start     string
	End       string
	// Page는 받을 페이지 번호다. 1부터 센다. 0이면 1이다(KEDP001-D022 21).
	Page int
}

// PageNumber는 받을 페이지 번호다. 0 이하면 1이다.
func (r Request) PageNumber() int { return max(r.Page, 1) }

// SampleName은 이 요청의 응답 샘플 파일 이름이다.
// 통계 코드-항목 코드-분류 코드…-주기-시작-끝.json (비어 있는 칸은 빠진다. 예: 공공데이터포털은 항목 코드가 없다)
func (r Request) SampleName() string {
	ind := r.Indicator
	all := append([]string{ind.StatCode, ind.ItemCode}, ind.Classes...)
	all = append(all, ind.Cycle, r.Start, r.End)
	var parts []string
	for _, p := range all {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, "-") + ".json"
}

// Source는 지표 정보를 받아 원본 응답을 돌려준다.
type Source interface {
	Name() string
	// Fetch는 req.Page 페이지의 원본 응답을 돌려준다.
	Fetch(ctx context.Context, req Request) ([]byte, error)
	// LastPage는 응답 본문에 적힌 전체 행 수로 마지막 페이지 번호를 돌려준다.
	// 페이지를 나누지 않는 제공처이거나 전체 행 수를 읽을 수 없으면(오류 응답 등) 1이다.
	LastPage(body []byte) int
	// Period는 날짜 t가 들어 있는 기간을 cycle(D/M/Q/A)의 요청 형식으로 돌려준다(설계서 5.1).
	// 제공처가 받지 않는 주기면 오류다.
	Period(cycle string, t time.Time) (string, error)
}

// Pages는 전체 행 수 total을 size개씩 나눈 페이지 수다. 0 이하면 1이다.
func Pages(total, size int) int {
	if total <= 0 || size <= 0 {
		return 1
	}
	return (total + size - 1) / size
}

// Quarter는 t의 분기(1~4)다.
func Quarter(t time.Time) int { return (int(t.Month())-1)/3 + 1 }

// UnsupportedCycle은 제공처가 받지 않는 주기의 오류다.
func UnsupportedCycle(source, cycle string) error {
	return fmt.Errorf("%s: cycle %q is not supported", source, cycle)
}

// APIError는 제공처가 응답 본문으로 돌려준 오류다. 코드는 제공처마다 다르다.
type APIError struct {
	Source  string
	Code    string
	Message string
	// NoData는 "해당하는 데이터가 없음" 오류인지 나타낸다(ECOS INFO-200, KOSIS 30, 공공데이터포털 totalCount 0).
	NoData bool
}

func (e *APIError) Error() string { return fmt.Sprintf("%s: %s %s", e.Source, e.Code, e.Message) }

// NoData는 err가 "해당하는 데이터가 없음" 오류인지 알려 준다.
func NoData(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.NoData
}

// KeyError는 키가 비어 있어 요청하지 않았다는 오류다. 환경 변수 이름만 들어 있다.
type KeyError struct{ Source, EnvName string }

func (e *KeyError) Error() string { return fmt.Sprintf("%s: %s is not set", e.Source, e.EnvName) }

// RequireKey는 키가 비어 있으면 *KeyError를 돌려준다.
func RequireKey(source, envName, key string) error {
	if key == "" {
		return &KeyError{Source: source, EnvName: envName}
	}
	return nil
}

// Base는 제공처가 함께 쓰는 이름, 키, 클라이언트다. 제공처 Source에 넣어(embed) 쓴다.
type Base struct {
	name    string
	envName string
	key     string
	client  *fetch.Client
}

// NewBase는 Base를 만든다. envName은 키의 환경 변수 이름이고, client는 key를 가리도록 만든 것이어야 한다.
func NewBase(name, envName, key string, client *fetch.Client) Base {
	return Base{name: name, envName: envName, key: key, client: client}
}

func (b Base) Name() string { return b.name }

// Get은 키를 넣어 만든 요청 주소로 요청을 보낸다. 키가 비어 있으면 요청하지 않는다.
// buildURL이 만든 주소는 fetch.Client를 거쳐서만 기록된다.
func (b Base) Get(ctx context.Context, buildURL func(key string) string) ([]byte, error) {
	if err := RequireKey(b.name, b.envName, b.key); err != nil {
		return nil, err
	}
	return b.client.Get(ctx, buildURL(b.key))
}
