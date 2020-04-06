// +build !windows

package util

import (
	"fmt"
	"syscall"
)

func Setsid() error {
	fmt.Println("!!!")

	pid, err := syscall.Setsid()
	if pid == -1 {
		return err
	}

	fmt.Println("!!!")

	return nil
}
