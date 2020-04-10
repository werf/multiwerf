package multiwerf

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"

	"github.com/flant/shluz"

	"github.com/flant/multiwerf/pkg/app"
	"github.com/flant/multiwerf/pkg/output"
)

var (
	StorageDir string
	TmpDir     string
)

type UpdateOptions struct {
	SkipSelfUpdate          bool
	TryRemoteChannelMapping bool
	WithCache               bool
	WithGC                  bool
	OutputFile              string
}

// Update checks for the actual version for group/channel and downloads it to StorageDir if it does not already exist
//
// Arguments:
//
// - group - a major.minor version to update
// - channel - a string with channel name
// - options.SkipSelfUpdate - a boolean to perform self-update
// - options.WithCache - a boolean to try or not getting remote channel mapping
// - options.WithGC - a boolean to run GC before update
// - options.OutputFile - a string to write update output to file
func Update(group, channel string, options UpdateOptions) (err error) {
	var w io.Writer
	if options.OutputFile != "" {
		dirPath := filepath.Dir(options.OutputFile)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("mkdir '%s' failed: %s", dirPath, err)
		}

		f, err := os.OpenFile(options.OutputFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			return fmt.Errorf("open log file '%s' failed: %s", options.OutputFile, err)
		}
		defer f.Close()

		w = f
	} else {
		w = os.Stdout
	}

	printer := output.NewSimplePrint(w)

	if err := ValidateGroup(group, printer); err != nil {
		return err
	}

	if err := SetupStorageDir(printer); err != nil {
		return err
	}

	if err := PerformSelfUpdate(printer, options.SkipSelfUpdate); err != nil {
		return err
	}

	if options.WithGC {
		if err := gc(printer); err != nil {
			return err
		}
	}

	tryRemoteChannelMapping, err := processTryRemoteChannelMapping(printer, channel, options.WithCache, options.TryRemoteChannelMapping)
	if err != nil {
		return err
	}

	messages := make(chan ActionMessage, 0)

	go func() {
		UpdateChannelVersionBinary(messages, group, channel, tryRemoteChannelMapping)
		messages <- ActionMessage{action: "exit"}
	}()

	return PrintActionMessages(messages, printer)
}

func processTryRemoteChannelMapping(printer output.Printer, channel string, withCache, tryRemoteChannelMapping bool) (bool, error) {
	isLocalChannelMappingFileExist, err := isLocalChannelMappingFileExist()
	if err != nil {
		return false, err
	}

	if !isLocalChannelMappingFileExist {
		tryRemoteChannelMapping = true
	}

	if withCache {
		tryRemoteChannelMappingDelay := DelayFile{
			Filename: filepath.Join(StorageDir, "try-remote-channel-mapping.delay"),
		}

		if channel == "alpha" || channel == "beta" {
			tryRemoteChannelMappingDelay.WithDelay(app.AlphaBetaUpdateDelay)
		} else {
			tryRemoteChannelMappingDelay.WithDelay(app.UpdateDelay)
		}

		remains := tryRemoteChannelMappingDelay.TimeRemains()
		if remains != "" && isLocalChannelMappingFileExist {
			PrintActionMessage(
				ActionMessage{
					msg:     fmt.Sprintf("multiwerf channel mapping update has been delayed: %s left till next download attempt", remains),
					msgType: OkMsgType,
				},
				printer,
			)

			tryRemoteChannelMapping = false
		} else {
			if err := tryRemoteChannelMappingDelay.UpdateTimestamp(); err != nil {
				return false, err
			}
		}
	}

	return tryRemoteChannelMapping, nil
}

type UseOptions struct {
	ForceRemoteCheck        bool
	AsFile                  bool
	SkipSelfUpdate          bool
	TryRemoteChannelMapping bool
	WithGC                  bool
}

// Use:
// * prints a shell script or
// * generates a shell script file and prints the path
//
// The script includes two parts for defined group/version based on local channel mapping:
// * multiwerf update procedure that will be performed on background or foreground and
// * werf alias that uses path to the actual werf binary
func Use(group, channel string, shell string, options UseOptions) (err error) {
	printer := output.NewSilentPrint()
	if err := ValidateGroup(group, printer); err != nil {
		return err
	}

	if err := SetupStorageDir(printer); err != nil {
		return err
	}

	firstWerfPathLogPath := filepath.Join(StorageDir, "multiwerf_use_first_werf_path.log")
	backgroundUpdateLogPath := filepath.Join(StorageDir, "multiwerf_use_background_update.log")

	groupAndChannelArgs := []string{group, channel}
	commonUpdateArgs := groupAndChannelArgs[0:]
	if options.SkipSelfUpdate {
		commonUpdateArgs = append(commonUpdateArgs, "--self-update=no")
	}

	if !options.TryRemoteChannelMapping {
		commonUpdateArgs = append(commonUpdateArgs, "--update=no")
	}

	if !options.WithGC {
		commonUpdateArgs = append(commonUpdateArgs, "--with-gc=no")
	}

	foregroundUpdateArgs := commonUpdateArgs[0:]

	backgroundUpdateArgs := append(
		commonUpdateArgs[0:],
		"--in-background",
		fmt.Sprintf("--output-file=%s", backgroundUpdateLogPath),
	)
	if !options.ForceRemoteCheck {
		backgroundUpdateArgs = append(backgroundUpdateArgs, "--with-cache")
	}

	scriptArgs := []interface{}{
		strings.Join(groupAndChannelArgs, " "),  // %[1]s: group channel
		strings.Join(foregroundUpdateArgs, " "), // %[2]s: group channel [flag ...]
		strings.Join(backgroundUpdateArgs, " "), // %[3]s: group channel [flag ...]
		firstWerfPathLogPath,                    // %[4]s: multiwerf_use_first_werf_path.log
	}

	var filename = "werf_source"
	var filenameExt string
	var fileContent string

	switch shell {
	case "cmdexe":
		filenameExt = "bat"
		fileContent = fmt.Sprintf(`
FOR /F "tokens=*" %%%%g IN ('multiwerf werf-path %[1]s') do (SET WERF_PATH=%%%%g)
echo %%WERF_USE_SCRIPT_PATH%% > %[4]s

IF %%ERRORLEVEL%% NEQ 0 (
    multiwerf update %[2]s 
    FOR /F "tokens=*" %%%%g IN ('multiwerf werf-path %[1]s') do (SET WERF_PATH=%%%%g)
) ELSE (
    multiwerf update %[3]s
)

DOSKEY werf=%%WERF_PATH%% $*
`, scriptArgs...)
	case "powershell":
		filenameExt = "ps1"
		fileContent = fmt.Sprintf(`
if ((Invoke-Expression -Command "multiwerf werf-path %[1]s" | Out-String -OutVariable WERF_PATH) -and ($LastExitCode -eq 0)) {
    multiwerf update %[3]s 
} else {
    multiwerf update %[2]s
	Invoke-Expression -Command "multiwerf werf-path %[1]s" | Out-String -OutVariable WERF_PATH
}

function werf { & $WERF_PATH.Trim() $args }
`, scriptArgs...)
	default:
		if runtime.GOOS == "windows" {
			fileContent = fmt.Sprintf(`
if multiwerf werf-path %[1]s >%[4]s 2>&1; then
    multiwerf update %[3]s
else
    multiwerf update %[2]s
fi

WERF_PATH=$(multiwerf werf-path %[1]s | sed 's/\\/\//g')
WERF_FUNC=$(cat <<EOF
werf() 
{
    $WERF_PATH "\$@"
}
EOF
)

eval "$WERF_FUNC"
`, scriptArgs...)
		} else {
			fileContent = fmt.Sprintf(`
if multiwerf werf-path %[1]s >%[4]s 2>&1; then
    multiwerf update %[3]s
else
    multiwerf update %[2]s
fi

WERF_PATH=$(multiwerf werf-path %[1]s)
WERF_FUNC=$(cat <<EOF
werf() 
{
    $WERF_PATH "\$@"
}
EOF
)

eval "$WERF_FUNC"
`, scriptArgs...)
		}
	}

	fileContent = fmt.Sprintln(strings.TrimSpace(fileContent))

	if !options.AsFile {
		fmt.Printf(fileContent)
	} else {
		if options.ForceRemoteCheck {
			filename = strings.Join([]string{filename, "force_remote_check"}, "_with_")
		}

		withExtraArgs := !reflect.DeepEqual(commonUpdateArgs, groupAndChannelArgs)
		if withExtraArgs {
			filename = strings.Join([]string{filename, shluz.MurmurHash(strings.Join(commonUpdateArgs, " "))}, "_")
		}

		if filenameExt != "" {
			filename = strings.Join([]string{filename, filenameExt}, ".")
		}

		fileContentBytes := []byte(fileContent)
		dstPath := filepath.Join(StorageDir, "scripts", strings.Join([]string{group, channel}, "-"), filename)
		tmpDstPath := dstPath + ".tmp"

		if exist, err := FileExists(dstPath); err != nil {
			printer.Error(err)
			return err
		} else if exist {
			currentFileContentBytes, err := ioutil.ReadFile(dstPath)
			if err != nil {
				printer.Error(err)
				return err
			}

			if bytes.Equal(currentFileContentBytes, fileContentBytes) {
				fmt.Println(dstPath)
				return nil
			}
		}

		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			printer.Error(err)
			return err
		}

		if err := ioutil.WriteFile(tmpDstPath, fileContentBytes, os.ModePerm); err != nil {
			printer.Error(err)
			return err
		}

		err = os.Rename(tmpDstPath, dstPath)
		if err != nil {
			err := fmt.Errorf("rename tmp file failed: %v", err)
			printer.Error(err)
			return err
		}

		fmt.Println(dstPath)
	}

	return nil
}

// WerfPath prints path to the actual version available for the group/channel based on local channel mapping
func WerfPath(group string, channel string) (err error) {
	printer := output.NewSilentPrint()

	if err := ValidateGroup(group, printer); err != nil {
		return err
	}

	if err := SetupStorageDir(printer); err != nil {
		return err
	}

	var binaryInfo *BinaryInfo
	messages := make(chan ActionMessage, 0)

	go func() {
		binaryInfo = UseChannelVersionBinary(messages, group, channel)
		messages <- ActionMessage{action: "exit"}
	}()

	if err = PrintActionMessages(messages, printer); err != nil {
		return err
	}

	fmt.Println(binaryInfo.BinaryPath)

	return nil
}

// WerfExec launches the latest binary version available for the group/channel based on local channel mapping
func WerfExec(group, channel string, args []string) (err error) {
	printer := output.NewSilentPrint()

	if err := ValidateGroup(group, printer); err != nil {
		return err
	}

	if err := SetupStorageDir(printer); err != nil {
		return err
	}

	var binaryInfo *BinaryInfo
	messages := make(chan ActionMessage, 0)

	go func() {
		binaryInfo = UseChannelVersionBinary(messages, group, channel)
		messages <- ActionMessage{action: "exit"}
	}()

	if err = PrintActionMessages(messages, printer); err != nil {
		return err
	}

	cmd := exec.Command(binaryInfo.BinaryPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

func ValidateGroup(group string, printer output.Printer) error {
	messages := make(chan ActionMessage, 0)

	go func() {
		err := CheckMajorMinor(group)
		if err != nil {
			messages <- ActionMessage{err: err}
		}

		messages <- ActionMessage{
			msg:   fmt.Sprintf("The group %s is the valid major.minor version", group),
			debug: true,
		}

		messages <- ActionMessage{action: "exit"}
	}()

	return PrintActionMessages(messages, printer)
}

func SetupStorageDir(printer output.Printer) error {
	messages := make(chan ActionMessage, 0)

	go func() {
		var err error
		StorageDir, err = ExpandPath(app.StorageDir)
		if err != nil {
			messages <- ActionMessage{
				err: fmt.Errorf("invalid storage dir %s: %s", StorageDir, err),
			}
		}

		messages <- ActionMessage{
			msg:   fmt.Sprintf("storage dir is %s", StorageDir),
			debug: true,
		}

		TmpDir = filepath.Join(StorageDir, "tmp")
		if err := os.MkdirAll(TmpDir, 0755); err != nil {
			messages <- ActionMessage{
				err: fmt.Errorf("mkdir all failed %s: %s", TmpDir, err),
			}
		}

		messages <- ActionMessage{
			msg:   fmt.Sprintf("tmp dir is %s", TmpDir),
			debug: true,
		}

		if err := shluz.Init(filepath.Join(StorageDir, "locks")); err != nil {
			messages <- ActionMessage{
				err: fmt.Errorf("init shluz failed: %s", err),
			}
		}

		messages <- ActionMessage{action: "exit"}
	}()

	return PrintActionMessages(messages, printer)
}

func GC() error {
	printer := output.NewSimplePrint(os.Stdout)

	if err := SetupStorageDir(printer); err != nil {
		return err
	}

	return gc(printer)
}

func gc(printer output.Printer) error {
	messages := make(chan ActionMessage, 0)
	go func() {
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
