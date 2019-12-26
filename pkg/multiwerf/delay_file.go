package multiwerf

import (
	"fmt"
	"os"
	"time"
)

type DelayFile struct {
	Filename string
	Delay    time.Duration
}

func (u *DelayFile) WithDelay(d time.Duration) {
	u.Delay = d
}

// IsDelayPassed returns true if delay has passed since
// last UpdateTimestamp
func (u *DelayFile) IsDelayPassed() bool {
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
func (u *DelayFile) TimeRemains() string {
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
func (u *DelayFile) UpdateTimestamp() error {
	if exist, err := FileExists(u.Filename); err != nil {
		return fmt.Errorf("file exists failed: %s", err)
	} else if exist {
		if err := os.Remove(u.Filename); err != nil {
			return fmt.Errorf("remove file failed: %s", err)
		}
	}

	if f, err := os.Create(u.Filename); err != nil {
		return fmt.Errorf("create file failed: %s", err)
	} else if err := f.Close(); err != nil {
		return fmt.Errorf("close file failed: %s", err)
	}

	return nil
}
