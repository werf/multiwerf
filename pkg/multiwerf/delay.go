package multiwerf

import (
	"os"
	"time"
)

type Delayable interface {
	SetDelay(d time.Duration)
	IsDelayPassed() bool
	UpdateTimestamp()
}

type UpdateDelay struct {
	Filename string
	Delay    time.Duration
}

func (u *UpdateDelay) SetDelay(d time.Duration) {
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

// UpdateTimestamp sets delay timestamp as now()
func (u *UpdateDelay) UpdateTimestamp() {
	os.Remove(u.Filename)
	f, _ := os.Create(u.Filename)
	f.Close()
}
