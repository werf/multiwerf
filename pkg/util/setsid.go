// +build !windows

package util

import (
	"fmt"
	"os"
	"syscall"
)

func Setsid() error {
	fmt.Println("!!!")
	fmt.Println(os.Getpid())
	fmt.Println(os.Getppid())
	pid, err := syscall.Setsid()
	if pid == -1 || err != nil {
		return err
	}

	fmt.Println("!!!")
	fmt.Println(os.Getpid())
	fmt.Println(os.Getppid())

	return nil
}
