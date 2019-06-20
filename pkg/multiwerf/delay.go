package multiwerf

import (
	"os"
	"time"
)

type Delayable interface {
	WithDelay(d time.Duration)
	IsDelayPassed() bool
	TimeRemains() string
	UpdateTimestamp()
}

type UpdateDelay struct {
	Filename string
	Delay    time.Duration
}

func (u *UpdateDelay) WithDelay(d time.Duration) {
	u.Delay = d
}

// IsDelayPassed returns true if delay has passed since
// last UpdateTimestamp
func (u *UpdateDelay) IsDelayPassed() bool {
	// TODO implement locking
	info, err := os.Stat(u.Filename)
	// File is not exists — delay is passed
	// TODO clean things up here
	if err != nil {
		return true
	}

	fTime := info.ModTime()

	if fTime.Add(u.Delay).Before(time.Now()) {
		return true
	}

	return false
}

// TimeRemains returns a string representation of time until delay is passed.
// Empty string is returned if delay is passed
func (u *UpdateDelay) TimeRemains() string {
	info, err := os.Stat(u.Filename)
	// File is not exists — delay is passed
	if err != nil {
		return ""
	}

	now := time.Now()
	delayed := info.ModTime().Add(u.Delay)

	if delayed.After(now) {
		diff := time.Second * time.Duration(delayed.Unix()-now.Unix())
		return diff.String()
	}

	return ""
}

// UpdateTimestamp sets delay timestamp as now() be recreating a delay file
func (u *UpdateDelay) UpdateTimestamp() {
	_ = os.Remove(u.Filename)
	f, _ := os.Create(u.Filename)
	_ = f.Close()
}
