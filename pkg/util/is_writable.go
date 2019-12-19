// +build !windows

package util

import (
	"fmt"
	"os"
	"syscall"
)

func PathShouldBeWritable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("cannot stat file %s: %v", path, err)
	}

	// Check if the user write bit is enabled in file permission
	if info.Mode().Perm()&(1<<(uint(7))) == 0 {
		return fmt.Errorf("write permission bit is not set on %s", path)
	}

	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("no stat_t for file %s", path)
	}

	if uint32(os.Geteuid()) != stat.Uid {
		return fmt.Errorf("user %d doesn't have permission to write %s", os.Geteuid(), path)
	}

	return nil
}
