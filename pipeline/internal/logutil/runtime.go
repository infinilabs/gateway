package logutil

import (
	"math"
	"time"
)

func FormatDuration(duration time.Duration) string {
	if duration <= 0 {
		return "0s"
	}
	return duration.Round(time.Millisecond).String()
}

func QPS(total int64, duration time.Duration) int64 {
	if total <= 0 {
		return 0
	}
	if duration <= 0 {
		return total
	}
	return int64(math.Ceil(float64(total) / duration.Seconds()))
}
