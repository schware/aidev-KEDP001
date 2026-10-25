package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/schware/aidev-KEDP001/b-go-end/internal/config"
	"github.com/schware/aidev-KEDP001/b-go-end/internal/source"
)

// Request는 요청 하나다. 명령 인자나 목록 파일에서 받은 값을 그대로 제공처에 넣는다(indicators.yaml을 거치지 않는다).
type Request struct {
	Source string `yaml:"source"` // ecos | kosis | data-go-kr
	// API는 제공처 안의 요청 종류다. 비우면 제공처의 통계 자료 요청이다.
	API      string   `yaml:"api"`
	Stat     string   `yaml:"stat"`      // ECOS 통계 코드, KOSIS tblId, 공공데이터포털 종목 코드
	Item     string   `yaml:"item"`      // ECOS ITEM_CODE1, KOSIS itmId
	Classes  []string `yaml:"classes"`   // KOSIS objL1, objL2…, ECOS ITEM_CODE2, ITEM_CODE3…
	Cycle    string   `yaml:"cycle"`     // 제공처 요청에 넣는 주기 값 그대로
	Start    string   `yaml:"start"`     // 조회 시작 기간
	End      string   `yaml:"end"`       // 조회 끝 기간
	MetaType string   `yaml:"meta_type"` // KOSIS getMeta type (예: ITM, TBL, PRD)
	Name     string   `yaml:"name"`      // 저장 파일 이름. 비우면 요청 값으로 만든다
}

// 요청 종류
const (
	apiSearch = "search" // ECOS StatisticSearch
	apiItems  = "items"  // ECOS StatisticItemList
	apiData   = "data"   // KOSIS statisticsParameterData
	apiMeta   = "meta"   // KOSIS getMeta
	apiPrice  = "price"  // 공공데이터포털 getETFPriceInfo_V2
)

// apis는 제공처마다 받는 요청 종류다. 첫째가 기본값이다.
var apis = map[string][]string{
	"ecos":       {apiSearch, apiItems},
	"kosis":      {apiData, apiMeta},
	"data-go-kr": {apiPrice},
}

// normalize는 기본값을 채우고 요청 종류마다 꼭 필요한 값이 있는지 확인한다.
func (r *Request) normalize() error {
	kinds, ok := apis[r.Source]
	if !ok {
		return fmt.Errorf("source %q: want ecos, kosis or data-go-kr", r.Source)
	}
	if r.API == "" {
		r.API = kinds[0]
	}
	if !contains(kinds, r.API) {
		return fmt.Errorf("%s: api %q: want one of %s", r.Source, r.API, strings.Join(kinds, ", "))
	}

	need := map[string]string{"stat": r.Stat}
	switch r.API {
	case apiSearch, apiData:
		need["item"], need["cycle"], need["start"], need["end"] = r.Item, r.Cycle, r.Start, r.End
	case apiMeta:
		need["meta_type"] = r.MetaType
	case apiPrice:
		need["start"], need["end"] = r.Start, r.End
		// 일 단위만 있다. 요청에는 넣지 않고 파일 이름에만 쓴다.
		r.Cycle = "D"
	}
	var missing []string
	for _, name := range []string{"stat", "item", "cycle", "start", "end", "meta_type"} {
		if v, ok := need[name]; ok && v == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("%s %s: missing %s", r.Source, r.API, strings.Join(missing, ", "))
	}

	if r.Name == "" {
		r.Name = r.defaultName()
	}
	if !strings.HasSuffix(r.Name, ".json") {
		r.Name += ".json"
	}
	if r.Name != filepath.Base(r.Name) || strings.HasPrefix(r.Name, ".") {
		return fmt.Errorf("name %q: want a file name without folder", r.Name)
	}
	return nil
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// defaultName은 저장 파일 이름이다. 통계 자료는 cmd/server와 같은 이름(source.Request.SampleName)을 쓴다.
func (r Request) defaultName() string {
	switch r.API {
	case apiItems:
		return r.Stat + "-items.json"
	case apiMeta:
		return r.Stat + "-meta-" + r.MetaType + ".json"
	}
	return r.sourceRequest().SampleName()
}

// sourceRequest는 통계 자료 요청을 internal/source의 요청으로 바꾼다.
func (r Request) sourceRequest() source.Request {
	return source.Request{
		Indicator: config.Indicator{Source: r.Source, StatCode: r.Stat, ItemCode: r.Item, Classes: r.Classes, Cycle: r.Cycle},
		Start:     r.Start,
		End:       r.End,
	}
}

// infoName은 요청 정보 파일 이름이다.
func (r Request) infoName() string {
	return strings.TrimSuffix(r.Name, ".json") + ".request.json"
}

type listFile struct {
	Requests []Request `yaml:"requests"`
}

// loadList는 요청 목록 파일을 읽는다. 모르는 칸(예: 키)이 있으면 오류다.
func loadList(path string) ([]Request, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	var f listFile
	if err := dec.Decode(&f); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if len(f.Requests) == 0 {
		return nil, fmt.Errorf("%s: no requests", path)
	}
	return f.Requests, nil
}

// prepare는 요청을 모두 확인한다. 하나라도 틀리면 아무것도 부르지 않도록 요청 전에 부른다.
// 같은 폴더에 같은 이름으로 저장할 요청이 둘이면 오류다.
func prepare(reqs []Request) error {
	var errs []error
	seen := map[string]int{}
	for i := range reqs {
		if err := reqs[i].normalize(); err != nil {
			errs = append(errs, fmt.Errorf("request %d: %w", i+1, err))
			continue
		}
		key := reqs[i].Source + "/" + reqs[i].Name
		if j, ok := seen[key]; ok {
			errs = append(errs, fmt.Errorf("request %d: same file as request %d (%s); set name", i+1, j, key))
		}
		seen[key] = i + 1
	}
	return errors.Join(errs...)
}
