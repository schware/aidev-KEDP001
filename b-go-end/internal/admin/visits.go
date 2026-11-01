package admin

import (
	"errors"
	"net/http"
	"time"

	"github.com/schware/aidev-KEDP001/b-go-end/internal/store"
)

// 방문자 수 관리(specification/40.api/README.md 6장, Issue #48).

// VisitDays는 관리자 화면에 보이는 일별 방문자 수의 날 수(오늘 포함)다.
const VisitDays = 30

// MaxVisitBase는 시작값의 최댓값이다. 화면(JavaScript)이 정확히 읽을 수 있는 가장 큰 정수다.
const MaxVisitBase = 1<<53 - 1

// VisitDay는 하루의 방문자 수다. day는 Asia/Seoul의 날짜(YYYY-MM-DD)다.
type VisitDay struct {
	Day   string `json:"day"`
	Count int64  `json:"count"`
}

// Visits는 GET /api/admin/visits의 응답이다.
type Visits struct {
	Today int64      `json:"today"`
	Total int64      `json:"total"`
	Base  int64      `json:"base"`
	Days  []VisitDay `json:"days"`
}

func (s *server) writeVisits(w http.ResponseWriter, r *http.Request) {
	v, err := s.Store.VisitStats(r.Context(), s.Now().In(s.Loc), VisitDays)
	if err != nil {
		serverError(w, r, err)
		return
	}
	out := Visits{Today: v.Today, Total: v.Total, Base: v.Base, Days: make([]VisitDay, 0, len(v.Days))}
	for _, d := range v.Days {
		out.Days = append(out.Days, VisitDay{Day: d.Day.Format(time.DateOnly), Count: d.Count})
	}
	write(w, http.StatusOK, out)
}

func (s *server) getVisits(w http.ResponseWriter, r *http.Request) { s.writeVisits(w, r) }

func (s *server) setVisitBase(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Base *int64 `json:"base"`
	}
	baseErr := &store.FieldError{Field: "base", Reason: "must be an integer from 0 to 9007199254740991"}
	if err := decode(w, r, &req); err != nil {
		// 숫자가 아니거나, 소수이거나, int64를 넘으면 decode가 base 칸 오류를 낸다. 이유는 하나로 맞춘다.
		if fe := (*store.FieldError)(nil); errors.As(err, &fe) && fe.Field == "base" {
			err = baseErr
		}
		badRequest(w, err)
		return
	}
	if req.Base == nil || *req.Base < 0 || *req.Base > MaxVisitBase {
		badRequest(w, baseErr)
		return
	}
	old, err := s.Store.SetVisitBase(r.Context(), *req.Base)
	if err != nil {
		serverError(w, r, err)
		return
	}
	audit(r, "visit_total_base %s -> %d", old, *req.Base)
	s.writeVisits(w, r)
}

// resetVisits는 일별 기록을 모두 지우고 시작값을 0으로 한다. 본문은 읽지 않는다.
// 서버 메모리의 그날 hash 목록은 그대로라, 오늘 이미 센 사람은 오늘 다시 세지 않는다.
func (s *server) resetVisits(w http.ResponseWriter, r *http.Request) {
	res, err := s.Store.ResetVisits(r.Context())
	if err != nil {
		serverError(w, r, err)
		return
	}
	audit(r, "reset visits: deleted %d days (%d visits), visit_total_base %s -> 0", res.Days, res.Count, res.OldBase)
	s.writeVisits(w, r)
}
