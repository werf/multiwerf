// +build !windows

package util

import (
	"syscall"
)

func Setsid() error {
	pid, err := syscall.Setsid()
	if pid == -1 {
		return err
	}

	return nil
}
