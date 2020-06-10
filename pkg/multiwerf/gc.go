package multiwerf

import (
	"fmt"
	"os"
	"sort"

	"github.com/werf/lockgate"

	"github.com/werf/multiwerf/pkg/locker"

	"github.com/werf/multiwerf/pkg/output"
)

const GCLockName = "gc"

func gc(printer output.Printer) error {
	messages := make(chan ActionMessage, 0)
	go func() {
		isAcquired, lockHandle, err := locker.Locker.Acquire(GCLockName, lockgate.AcquireOptions{NonBlocking: true})
		if err != nil {
			messages <- ActionMessage{
				msg:     fmt.Sprintf("GC: Cannot acquire a lock: %v", err),
				msgType: WarnMsgType,
			}

			messages <- ActionMessage{action: "exit"}

			return
		} else if !isAcquired {
			messages <- ActionMessage{
				msg:     "GC: Skipped due to performing the operation by another process",
				msgType: WarnMsgType,
			}

			messages <- ActionMessage{action: "exit"}

			return
		}

		defer func() { _ = locker.Locker.Release(lockHandle) }()

		var actualVersions []string
		for _, channelMappingFilePath := range []string{localChannelMappingPath(), localOldChannelMappingPath()} {
			channelMapping, err := newLocalChannelMapping(channelMappingFilePath)
			if err != nil {
				switch err.(type) {
				case LocalChannelMappingNotFoundError:
					continue
				default:
					messages <- ActionMessage{err: err}
				}
			}

			if channelMapping == nil {
				messages <- ActionMessage{
					msg:     fmt.Sprintf("GC: Channel mapping invalid: %s", channelMappingFilePath),
					msgType: WarnMsgType,
					stage:   "gc",
				}

				continue
			}

		channelMappingVersionsLoop:
			for _, cVersion := range channelMapping.AllVersions() {
				for _, version := range actualVersions {
					if cVersion == version {
						continue channelMappingVersionsLoop
					}
				}

				actualVersions = append(actualVersions, cVersion)
			}
		}

		sort.Strings(actualVersions)

		messages <- ActionMessage{
			msg:     fmt.Sprintf("GC: Actual versions: %v", actualVersions),
			msgType: OkMsgType,
			stage:   "gc",
		}

		localVersions, err := localVersions()
		if err != nil {
			messages <- ActionMessage{err: err}
		}

		sort.Strings(localVersions)

		messages <- ActionMessage{
			stage:   "gc",
			msg:     fmt.Sprintf("GC: Local versions:  %v", localVersions),
			msgType: OkMsgType,
		}

		var versionsToRemove []string
	localVersionsLoop:
		for _, localVersion := range localVersions {
			for _, version := range actualVersions {
				if version == localVersion {
					continue localVersionsLoop
				}
			}

			versionsToRemove = append(versionsToRemove, localVersion)
		}

		if len(versionsToRemove) == 0 {
			messages <- ActionMessage{
				stage:   "gc",
				msg:     "GC: Nothing to clean",
				msgType: OkMsgType,
			}
		}

		for _, version := range versionsToRemove {
			messages <- ActionMessage{
				msg:     fmt.Sprintf("GC: Removing version %v ...", version),
				msgType: OkMsgType,
				stage:   "gc",
			}

			if err := os.RemoveAll(localVersionDirPath(version)); err != nil {
				messages <- ActionMessage{err: err}
			}
		}

		messages <- ActionMessage{action: "exit"}
	}()

	return PrintActionMessages(messages, printer)
}
