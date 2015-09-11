package ratelimit

import (
	"sync"
	"sync/atomic"
	"time"
)

// TimeSliceLimiter limits the number of operations per unit of time. It does not control the rate
// at which those operations are performed within the unit of time.
//
// For example, a TimeSliceLimiter which limits the user to 50 operations per hour will not complain
// if the user performs 49 operations at a rate of 100op/hour, as long as they do not make two more
// operations within the same hour.
type TimeSliceLimiter struct {
	timeSlice  time.Duration
	maxCounter int64

	accessLock    sync.RWMutex
	sweepRunning  bool
	earliestSlice *timeSlice
	latestSlice   *timeSlice
	sliceMap      map[string]*timeSlice
}

// NewTimeSliceLimiter generates a TimeSliceLimiter with the given parameters.
func NewTimeSliceLimiter(sliceTime time.Duration, maxCounter int64) *TimeSliceLimiter {
	return &TimeSliceLimiter{timeSlice: sliceTime, maxCounter: maxCounter,
		sliceMap: map[string]*timeSlice{}}
}

// Get returns the number of operations a given ID is allowed to perform in the current time slice.
// If the ID has never been limited, this will be the maximum counter value. This may be negative.
func (t *TimeSliceLimiter) Get(id string) int64 {
	t.accessLock.RLock()
	defer t.accessLock.RUnlock()
	if slice, ok := t.sliceMap[id]; ok {
		return slice.Get()
	} else {
		return t.maxCounter
	}
}

// Limit decrements a "counter" for a given ID. The counter starts at the maximum counter value and
// resets after the time slice goes by.
//
// This returns false if the counter has reached a value below zero (i.e. if the user has used all
// their operations for this time slice).
func (t *TimeSliceLimiter) Limit(id string) bool {
	t.accessLock.RLock()
	if slice, ok := t.sliceMap[id]; ok && !slice.Expired() {
		defer t.accessLock.RUnlock()
		return slice.Decrement() >= 0
	}
	t.accessLock.RUnlock()

	expiration := time.Now().Add(t.timeSlice)

	t.accessLock.Lock()
	defer t.accessLock.Unlock()

	// NOTE: while we had the mutex unlocked, some other goroutine may have created a slice.
	if slice, ok := t.sliceMap[id]; ok && !slice.Expired() {
		return slice.Decrement() >= 0
	}

	slice := &timeSlice{expiration, id, t.maxCounter - 1, nil}
	t.addSlice(slice)

	if !t.sweepRunning {
		t.sweepRunning = true
		go t.sweepLoop()
	}

	return t.maxCounter-1 >= 0
}

func (t *TimeSliceLimiter) addSlice(slice *timeSlice) {
	if t.latestSlice != nil {
		t.latestSlice.Next = slice
		t.latestSlice = slice
	} else {
		t.earliestSlice = slice
		t.latestSlice = slice
	}
	t.sliceMap[slice.Identifier] = slice
}

func (t *TimeSliceLimiter) removeEarliestSlice() {
	delete(t.sliceMap, t.earliestSlice.Identifier)
	t.earliestSlice = t.earliestSlice.Next
	if t.earliestSlice == nil {
		t.latestSlice = nil
	}
}

// sweepRoutine removes slices which have expired to prevent memory waste.
func (t *TimeSliceLimiter) sweepLoop() {
	for {
		t.accessLock.Lock()
		for t.earliestSlice != nil && t.earliestSlice.Expired() {
			t.removeEarliestSlice()
		}
		var iterationDelay time.Duration

		if t.earliestSlice == nil {
			t.sweepRunning = false
			t.accessLock.Unlock()
			return
		} else {
			iterationDelay = t.earliestSlice.Expiration.Sub(time.Now())
		}
		t.accessLock.Unlock()

		// NOTE: a little more than we need to in order to deal with clock imprecision.
		time.Sleep(iterationDelay + time.Millisecond)
	}
}

type timeSlice struct {
	Expiration time.Time
	Identifier string
	Remaining  int64

	Next *timeSlice
}

func (t *timeSlice) Decrement() int64 {
	return atomic.AddInt64(&t.Remaining, -1)
}

func (t *timeSlice) Expired() bool {
	return time.Now().After(t.Expiration)
}

func (t *timeSlice) Get() int64 {
	return atomic.LoadInt64(&t.Remaining)
}
