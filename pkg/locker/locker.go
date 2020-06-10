package locker

import (
	"github.com/werf/lockgate"
	"github.com/werf/lockgate/pkg/file_lock"
)

var (
	Locker lockgate.Locker
)

func Init(locksDir string) error {
	file_lock.LegacyHashFunction = true
	if locker, err := lockgate.NewFileLocker(locksDir); err != nil {
		return err
	} else {
		Locker = locker
	}

	return nil
}
