package admin

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/schware/aidev-KEDP001/b-go-end/internal/store"
)

// 입력 검사. 규칙은 DB 제약(b-sql-db/migrations/0001_schema.up.sql)과 같다.
// 제공처에 코드가 맞는지는 묻지 않는다. 다음 수집 결과가 API 관리(제공처 상태)에 보인다.

var (
	// indicator.id CHECK와 같다. 화면 주소 /indicator/{id}에도 쓴다.
	idPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	// 차트 기간은 N years, N months, N days만 받는다.
	periodPattern = regexp.MustCompile(`^[1-9][0-9]{0,3} (years|months|days)$`)
	// 수집 시각 HH:MM(00:00~23:59)
	clockPattern = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`)

	sources = []string{"ecos", "kosis", "data-go-kr"}
	cycles  = []string{"D", "M", "Q", "A"}
)

const (
	MinDecimals = 0
	MaxDecimals = 4
)

// fieldErrors는 칸 이름별 입력 오류다.
type fieldErrors map[string]string

func (e fieldErrors) Error() string { return fmt.Sprintf("invalid input: %v", map[string]string(e)) }

func required(missing map[string]bool) fieldErrors {
	e := fieldErrors{}
	for k, v := range missing {
		if v {
			e[k] = "required"
		}
	}
	return e
}

// text는 앞뒤 공백을 뺀 값이다. nil이면 빈 글자다.
func text(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

// display는 추가·수정에 같이 쓰는 화면 설정 칸이다.
type display struct {
	ChartPeriod *string  `json:"chart_period"`
	Decimals    *int     `json:"decimals"`
	Notice      *string  `json:"notice"`
	Related     []string `json:"related"`
}

func (d display) check(e fieldErrors, self string) {
	if d.ChartPeriod == nil {
		e["chart_period"] = "required"
	} else if !periodPattern.MatchString(*d.ChartPeriod) {
		e["chart_period"] = `must be "N years", "N months" or "N days" (N 1-9999)`
	}
	if d.Decimals != nil && (*d.Decimals < MinDecimals || *d.Decimals > MaxDecimals) {
		e["decimals"] = fmt.Sprintf("must be %d-%d or null", MinDecimals, MaxDecimals)
	}
	for i, r := range d.Related {
		switch {
		case r == self:
			e["related"] = "must not include the indicator itself"
		case slices.Contains(d.Related[:i], r):
			e["related"] = fmt.Sprintf("indicator %q is listed twice", r)
		case !idPattern.MatchString(r):
			e["related"] = fmt.Sprintf("indicator %q does not exist", r)
		}
	}
}

func checkName(e fieldErrors, name *string) {
	if text(name) == "" {
		e["name"] = "required"
	}
}

func checkCycle(e fieldErrors, cycle *string) {
	switch {
	case cycle == nil:
		e["cycle"] = "required"
	case !slices.Contains(cycles, *cycle):
		e["cycle"] = "must be one of D, M, Q, A"
	}
}

// addRequest는 POST /api/admin/indicators의 본문이다.
type addRequest struct {
	ID         *string  `json:"id"`
	Name       *string  `json:"name"`
	Source     *string  `json:"source"`
	StatCode   *string  `json:"stat_code"`
	ItemCode   *string  `json:"item_code"`
	Classes    []string `json:"classes"`
	OrgID      *string  `json:"org_id"`
	ValueField *string  `json:"value_field"`
	Cycle      *string  `json:"cycle"`
	Unit       *string  `json:"unit"`
	Visible    *bool    `json:"visible"`
	display
}

func (q addRequest) validate() (store.NewIndicator, fieldErrors) {
	e := fieldErrors{}
	id := text(q.ID)
	switch {
	case q.ID == nil:
		e["id"] = "required"
	case !idPattern.MatchString(id):
		e["id"] = "must be lowercase letters and digits joined by '-' (e.g. kosis-kospi)"
	}
	checkName(e, q.Name)
	src := text(q.Source)
	switch {
	case q.Source == nil:
		e["source"] = "required"
	case !slices.Contains(sources, src):
		e["source"] = "must be one of ecos, kosis, data-go-kr"
	}
	if text(q.StatCode) == "" {
		e["stat_code"] = "required"
	}
	checkCycle(e, q.Cycle)
	itemCode, orgID, valueField := text(q.ItemCode), text(q.OrgID), text(q.ValueField)
	// indicator_ecos_check, indicator_kosis_check, indicator_data_go_kr_check
	switch src {
	case "ecos":
		if itemCode == "" {
			e["item_code"] = "required for ecos"
		}
	case "kosis":
		if itemCode == "" {
			e["item_code"] = "required for kosis"
		}
		if orgID == "" {
			e["org_id"] = "required for kosis"
		}
	case "data-go-kr":
		if itemCode != "" {
			e["item_code"] = "must be empty for data-go-kr"
		}
		if valueField == "" {
			e["value_field"] = "required for data-go-kr"
		}
	}
	classes := make([]string, 0, len(q.Classes))
	for _, c := range q.Classes {
		if c = strings.TrimSpace(c); c == "" {
			e["classes"] = "must not contain empty codes"
		}
		classes = append(classes, c)
	}
	q.display.check(e, id)
	if len(e) > 0 {
		return store.NewIndicator{}, e
	}
	return store.NewIndicator{
		ID: id, Name: text(q.Name), Source: src, StatCode: text(q.StatCode), Cycle: *q.Cycle,
		OrgID: orgID, ItemCode: itemCode, ValueField: valueField, Unit: text(q.Unit),
		Classes: classes, Visible: q.Visible != nil && *q.Visible,
		ChartPeriod: *q.ChartPeriod, Decimals: q.Decimals, Notice: text(q.Notice), Related: nonNil(q.Related),
	}, nil
}

// editRequest는 PUT /api/admin/indicators/{id}의 본문이다. 바꿀 수 있는 칸만 받는다.
// 빠진 칸(unit, decimals, notice)은 null로 본다. related는 꼭 보낸다(관련 지표를 모두 지우려면 []).
type editRequest struct {
	Name  *string `json:"name"`
	Cycle *string `json:"cycle"`
	Unit  *string `json:"unit"`
	display
}

func (q editRequest) validate(id string) (store.Edit, fieldErrors) {
	e := fieldErrors{}
	checkName(e, q.Name)
	checkCycle(e, q.Cycle)
	if q.Related == nil {
		e["related"] = "required (use [] for none)"
	}
	q.display.check(e, id)
	if len(e) > 0 {
		return store.Edit{}, e
	}
	return store.Edit{
		Name: text(q.Name), Cycle: *q.Cycle, Unit: text(q.Unit),
		ChartPeriod: *q.ChartPeriod, Decimals: q.Decimals, Notice: text(q.Notice), Related: q.Related,
	}, nil
}

// orderRequest는 PUT /api/admin/display-order의 본문이다. 화면 설정이 있는 지표 전체를 보낸다.
type orderRequest struct {
	Indicators []struct {
		ID      *string `json:"id"`
		Visible *bool   `json:"visible"`
		Order   *int    `json:"order"`
	} `json:"indicators"`
}

func (q orderRequest) validate() ([]store.OrderItem, fieldErrors) {
	if q.Indicators == nil {
		return nil, fieldErrors{"indicators": "required"}
	}
	items := make([]store.OrderItem, 0, len(q.Indicators))
	for i, x := range q.Indicators {
		if x.ID == nil || x.Visible == nil || x.Order == nil {
			return nil, fieldErrors{"indicators": fmt.Sprintf("item %d needs id, visible and order", i)}
		}
		for _, it := range items {
			if it.ID == *x.ID {
				return nil, fieldErrors{"indicators": fmt.Sprintf("indicator %q is listed twice", *x.ID)}
			}
		}
		items = append(items, store.OrderItem{ID: *x.ID, Visible: *x.Visible, Order: *x.Order})
	}
	return items, nil
}

// describe는 서버 기록에 남길 지표 값이다.
func describe(a store.AdminIndicator) string {
	return fmt.Sprintf("(name %q, source %s, stat_code %s, item_code %s, classes %v, org_id %s, value_field %s, "+
		"cycle %s, unit %s, visible %t, chart_period %s, decimals %s, notice %s, related %v)",
		a.Name, a.Source, a.StatCode, str(a.ItemCode), a.Classes, str(a.OrgID), str(a.ValueField),
		a.Cycle, str(a.Unit), a.Visible, str(a.ChartPeriod), num(a.Decimals), str(a.Notice), a.Related)
}

// changes는 수정 전후에 달라진 칸이다.
func changes(b, a store.AdminIndicator) string {
	var out []string
	add := func(field, before, after string) {
		if before != after {
			out = append(out, fmt.Sprintf("%s %s -> %s", field, before, after))
		}
	}
	add("name", fmt.Sprintf("%q", b.Name), fmt.Sprintf("%q", a.Name))
	add("cycle", b.Cycle, a.Cycle)
	add("unit", str(b.Unit), str(a.Unit))
	add("chart_period", str(b.ChartPeriod), str(a.ChartPeriod))
	add("decimals", num(b.Decimals), num(a.Decimals))
	add("notice", str(b.Notice), str(a.Notice))
	add("related", fmt.Sprint(b.Related), fmt.Sprint(a.Related))
	if len(out) == 0 {
		return "no change"
	}
	return strings.Join(out, ", ")
}

// orderChanges는 노출·순서 저장 전후에 달라진 지표다.
func orderChanges(before []store.AdminIndicator, items []store.OrderItem) string {
	var out []string
	for _, it := range items {
		for _, b := range before {
			if b.ID != it.ID {
				continue
			}
			old := fmt.Sprintf("visible %t order %s", b.Visible, num(b.Order))
			now := fmt.Sprintf("visible %t order %d", it.Visible, it.Order)
			if old != now {
				out = append(out, fmt.Sprintf("%s %s -> %s", it.ID, old, now))
			}
		}
	}
	if len(out) == 0 {
		return "no change"
	}
	return strings.Join(out, ", ")
}

func str(p *string) string {
	if p == nil {
		return "null"
	}
	return fmt.Sprintf("%q", *p)
}

func num(p *int) string {
	if p == nil {
		return "null"
	}
	return fmt.Sprint(*p)
}
