package collect

import (
	"testing"
	"time"
)

func seoul(day, hour, minute, sec int) time.Time {
	return time.Date(2026, 10, day, hour, minute, sec, 0, Seoul)
}

func TestDue(t *testing.T) {
	tests := []struct {
		name      string
		clock     string
		prev, now time.Time
		want      bool
	}{
		{"before", "06:00", seoul(30, 5, 59, 0), seoul(30, 5, 59, 30), false},
		{"crossed", "06:00", seoul(30, 5, 59, 50), seoul(30, 6, 0, 20), true},
		{"exactly at", "06:00", seoul(30, 5, 59, 30), seoul(30, 6, 0, 0), true},
		{"already passed", "06:00", seoul(30, 6, 0, 0), seoul(30, 6, 0, 30), false},
		{"crossed midnight", "00:00", seoul(30, 23, 59, 50), seoul(31, 0, 0, 20), true},
		{"next day not yet", "06:00", seoul(30, 23, 59, 50), seoul(31, 0, 0, 20), false},
		// 서버 시간대와 상관없이 Asia/Seoul로 본다(06:00 KST = 전날 21:00 UTC)
		{"utc input", "06:00", seoul(30, 5, 59, 50).UTC(), seoul(30, 6, 0, 20).UTC(), true},
	}
	for _, tt := range tests {
		got, err := Due(tt.clock, tt.prev, tt.now)
		if err != nil || got != tt.want {
			t.Errorf("%s: Due = %v, %v; want %v", tt.name, got, err, tt.want)
		}
	}
}

// 수집 시각이 바뀌면 다음 수집부터 새 시각을 따른다(KEDP001-D019).
func TestDueClockChanged(t *testing.T) {
	prev, now := seoul(30, 6, 59, 50), seoul(30, 7, 0, 20)
	if got, _ := Due("06:00", prev, now); got {
		t.Error("06:00 is not due at 07:00")
	}
	if got, _ := Due("07:00", prev, now); !got {
		t.Error("07:00 is due at 07:00")
	}
}

func TestDueBadClock(t *testing.T) {
	for _, clock := range []string{"", "6", "24:00", "06:60", "06:00:00"} {
		if _, err := Due(clock, seoul(30, 0, 0, 0), seoul(30, 0, 0, 30)); err == nil {
			t.Errorf("Due(%q) has no error", clock)
		}
	}
}

func TestNext(t *testing.T) {
	tests := []struct {
		now, want time.Time
	}{
		{seoul(30, 5, 0, 0), seoul(30, 6, 0, 0)},
		{seoul(30, 6, 0, 0), seoul(31, 6, 0, 0)},
		{seoul(31, 23, 0, 0), time.Date(2026, 11, 1, 6, 0, 0, 0, Seoul)},
	}
	for _, tt := range tests {
		got, err := Next("06:00", tt.now)
		if err != nil || !got.Equal(tt.want) {
			t.Errorf("Next(%v) = %v, %v; want %v", tt.now, got, err, tt.want)
		}
	}
}
