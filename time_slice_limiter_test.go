package ratelimit

import (
	"testing"
	"time"
)

const shortUsableTimeSlice = time.Second / 10

func TestTimeSliceLimiterLongTerm(t *testing.T) {
	limiter := NewTimeSliceLimiter(time.Hour, 20)
	for i := 0; i < 19; i++ {
		if limiter.Limit("A") || limiter.Limit("B") {
			t.Error("unexpected limit failure for iteration", i)
		}
	}
	if limiter.Limit("A") {
		t.Error("unexpected limit failure for iteration 19 (A)")
	}
	if !limiter.Limit("A") {
		t.Error("expected ID to reach limit (A)")
	}
	if limiter.Limit("B") {
		t.Error("unexpected limit failure for iteration 19 (B)")
	}
	if !limiter.Limit("B") {
		t.Error("expected ID to reach limit (B)")
	}

	for i := 19; i > -10; i-- {
		if limiter.Decrement("C") != int64(i) {
			t.Error("unexpected decrement value", i)
		}
	}
}

func TestTimeSliceLimiterResetting(t *testing.T) {
	limiter := NewTimeSliceLimiter(shortUsableTimeSlice, 10)
	if limiter.Get("B") != 10 {
		t.Error("unexpected count")
	}
	limiter.Limit("B")
	if limiter.Get("B") != 9 {
		t.Error("unexpected count")
	}
	for i := 0; i < 10; i++ {
		if limiter.Limit("A") {
			t.Error("initial limit should not fail", i)
		}
		if limiter.Get("A") != 9-int64(i) {
			t.Error("unexpected count")
		}
	}
	limiter.Limit("C")
	if limiter.Get("C") != 9 {
		t.Error("unexpected count")
	}
	time.Sleep(shortUsableTimeSlice * 2)
	if limiter.Get("A") != 10 {
		t.Error("unexpected count")
	}
	if limiter.Limit("A") {
		t.Error("delayed limit should not fail")
	}
	if limiter.Get("A") != 9 {
		t.Error("unexpected count")
	}

	for i := 0; i < 9; i++ {
		if limiter.Limit("A") {
			t.Error("limit should not fail")
		}
		if limiter.Get("A") != 8-int64(i) {
			t.Error("unexpected count")
		}
	}
	if limiter.Get("A") != 0 {
		t.Error("unexpected count")
	}
	if !limiter.Limit("A") {
		t.Error("limit should fail")
	}
	if limiter.Get("A") != -1 {
		t.Error("unexpected count")
	}
	time.Sleep(shortUsableTimeSlice * 2)
	if limiter.Limit("A") {
		t.Error("delayed limit should not fail")
	}
	if limiter.Get("A") != 9 {
		t.Error("unexpected count")
	}
}

func TestTimeSliceLimiterSweeping(t *testing.T) {
	limiter := NewTimeSliceLimiter(shortUsableTimeSlice, 1)
	for i := 0; i < 2; i++ {
		limiter.Limit("A")
		limiter.Limit("B")
		limiter.Limit("C")

		limiter.accessLock.Lock()
		if len(limiter.sliceMap) != 3 {
			t.Error("unexpected slice map count", i)
		}
		if !limiter.sweepRunning {
			t.Error("expected sweep to be running", i)
		}
		limiter.accessLock.Unlock()

		time.Sleep(shortUsableTimeSlice * 2)
		limiter.accessLock.Lock()
		if len(limiter.sliceMap) != 0 {
			t.Error("unexpected slice map count", i)
		}
		if limiter.earliestSlice != nil || limiter.latestSlice != nil {
			t.Error("unexpected slice linked list", i)
		}
		if limiter.sweepRunning {
			t.Error("did not expect sweep to be running", i)
		}
		limiter.accessLock.Unlock()
	}
}
