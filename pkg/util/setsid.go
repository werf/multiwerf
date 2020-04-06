// +build !windows

package util

import (
	"fmt"
	"os"
	"syscall"
)

func Setsid() error {
	fmt.Println(syscall.Getuid())
	fmt.Println(syscall.Getpid())
	fmt.Println(syscall.Getppid())

	ret, _, errno := syscall.Syscall(syscall.SYS_FORK, 0, 0, 0)
	if errno != 0 {
		return fmt.Errorf("fork failed: errno %d", errno)
	}
	if ret > 0 {
		os.Exit(0)
	}

	pid, err := syscall.Setsid()
	if pid < 0 || err != nil {
		return err
	}

	fmt.Println(syscall.Getuid())
	fmt.Println(syscall.Getpid())
	fmt.Println(syscall.Getppid())

	return nil
}
