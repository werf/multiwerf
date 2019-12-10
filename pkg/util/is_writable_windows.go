package util

import (
	"fmt"
	"os"
)

func PathShouldBeWritable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("cannot stat file '%s': %v", path, err)
	}

	// Check if the user bit is enabled in file permission
	if info.Mode().Perm()&(1<<(uint(7))) == 0 {
		return fmt.Errorf("write permission bit is not set on '%s'", path)
	}

	return nil
}
