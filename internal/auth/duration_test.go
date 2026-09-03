package auth

import (
	"testing"
	"time"
)

func TestParseDurationDays(t *testing.T) {
	cases := []struct {
		in   string
		want time.Duration
	}{
		{"7d", 7 * 24 * time.Hour},
		{"1d", 24 * time.Hour},
		{"30d", 30 * 24 * time.Hour},
		{"", 7 * 24 * time.Hour},
		{"12h", 12 * time.Hour},
		{"garbage", 7 * 24 * time.Hour},
	}
	for _, c := range cases {
		if got := parseDuration(c.in); got != c.want {
			t.Errorf("parseDuration(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}
