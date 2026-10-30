package collect

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/schware/aidev-KEDP001/b-go-end/internal/store"
)

// ParseClock은 수집 시각(HH:MM, 24시간)을 읽는다.
func ParseClock(clock string) (hour, minute int, err error) {
	t, err := time.Parse("15:04", clock)
	if err != nil {
		return 0, 0, fmt.Errorf("collect_time %q is not HH:MM", clock)
	}
	return t.Hour(), t.Minute(), nil
}

// at은 날짜 d(Asia/Seoul)의 수집 시각이다.
func at(d time.Time, hour, minute int) time.Time {
	d = d.In(Seoul)
	return time.Date(d.Year(), d.Month(), d.Day(), hour, minute, 0, 0, Seoul)
}

// Due는 prev 뒤부터 now까지(prev < 시각 <= now) 수집 시각 clock(Asia/Seoul)이 지나갔는지 알려 준다.
// prev와 now의 사이는 하루보다 짧아야 한다.
func Due(clock string, prev, now time.Time) (bool, error) {
	h, m, err := ParseClock(clock)
	if err != nil {
		return false, err
	}
	for _, d := range []time.Time{prev, now} {
		if t := at(d, h, m); prev.Before(t) && !now.Before(t) {
			return true, nil
		}
	}
	return false, nil
}

// Next는 now 뒤의 첫 수집 시각이다.
func Next(clock string, now time.Time) (time.Time, error) {
	h, m, err := ParseClock(clock)
	if err != nil {
		return time.Time{}, err
	}
	t := at(now, h, m)
	if !t.After(now) {
		t = at(now.In(Seoul).AddDate(0, 0, 1), h, m)
	}
	return t, nil
}

// Schedule은 ctx가 끝날 때까지 하루 1회 수집 시각마다 Run(trigger schedule)을 부른다(KEDP001-D019).
// 수집 시각(app_setting collect_time)은 tick마다 DB에서 다시 읽으므로, 바뀌면 다음 수집부터 따른다.
// 시작했을 때 오늘 수집 시각이 이미 지났으면 다음 날부터 수집한다.
// 수집이 실패해도(DB, Rust 오류 포함) 기록만 하고 다음 수집 시각을 기다린다.
func (c *Collector) Schedule(ctx context.Context, tick time.Duration) error {
	ticker := time.NewTicker(tick)
	defer ticker.Stop()
	prev := c.now()
	clock := ""
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
		now := c.now()
		v, err := c.Store.CollectTime(ctx)
		if err != nil {
			log.Printf("schedule: %v", err)
			continue // prev를 그대로 두어, DB가 돌아오면 지나간 수집 시각을 놓치지 않는다
		}
		if v != clock {
			if next, err := Next(v, now); err == nil {
				log.Printf("schedule: collect_time %s, next %s", v, next.Format(time.DateTime+" MST"))
			}
			clock = v
		}
		due, err := Due(v, prev, now)
		if err != nil {
			log.Printf("schedule: %v", err)
			prev = now
			continue
		}
		prev = now
		if !due {
			continue
		}
		log.Printf("schedule: collection started")
		sum, err := c.Run(ctx, store.TriggerSchedule)
		if err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("schedule: collection failed: %v (%s)", err, sum)
		} else {
			log.Printf("schedule: collection finished (%s)", sum)
		}
		if next, err := Next(v, c.now()); err == nil {
			log.Printf("schedule: next %s", next.Format(time.DateTime+" MST"))
		}
	}
}
