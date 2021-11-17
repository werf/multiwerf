package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/werf/multiwerf/pkg/app"
	"github.com/werf/multiwerf/pkg/multiwerf"
)

var (
	groupHelp        = "Selector of a release series. Examples: 1.0, 1.1, 1.2."
	groupHintOptions = []string{"1.0", "1.1", "1.2"}

	channels = []string{
		"alpha",
		"beta",
		"ea",
		"stable",
		"rock-solid",
	}
	channelHelp = fmt.Sprintf("The minimum acceptable level of stability. One of: %s.", strings.Join(channels, "|"))
	channelEnum = []string{
		"alpha",
		"beta",
		"ea",
		"early-access",
		"rc", // legacy
		"stable",
		"rock-solid",
	}

	updateDefault = "yes"
	updateHelp    = "Try to download remote channel mapping and sync channel werf version. To disable set to 'no'."

	selfUpdateDefault = "yes"
	selfUpdateHelp    = "Perform multiwerf self-update. To disable set to 'no'."

	withGCDefault = "yes"
	withGCHelp    = "Run GC before update."

	tryTrdlDefault = "on-self-update"
	tryTrdlHelp    = "Automatically download and use trdl package manager instead of multiwerf, multiwerf is DEPRECATED, more info: https://github.com/werf/trdl. To disable set to 'no'. Try to use trdl only when self update is enabled by default ('on-self-update')."

	shellDefault = "default"
)

func main() {
	kpApp := kingpin.New(app.AppName, fmt.Sprintf("%s %s: %s", app.AppName, app.Version, app.AppDescription))

	app.SetupGlobalSettings(kpApp)

	updateCommand(kpApp)
	useCommand(kpApp)
	werfPathCommand(kpApp)
	werfExecCommand(kpApp)
	werfGCCommand(kpApp)
	versionCommand(kpApp)

	command, err := kpApp.Parse(os.Args[1:])
	if err != nil {
		kingpin.MustParse(command, err)
		os.Exit(1)
	}
}

func getTryTrdl(tryTrdl string, skipSelfUpdate bool) bool {
	var res bool
	switch tryTrdl {
	case "yes":
		res = true
	case "on-self-update":
		res = !skipSelfUpdate
	}

	return res
}

func updateCommand(kpApp *kingpin.Application) {
	var (
		groupStr           string
		channelStr         string
		update             string
		selfUpdate         string
		withCache          bool
		withGC             string
		updateInBackground bool
		updateOutputFile   string
		tryTrdl            string
	)

	// multiwerf update
	updateCmd := kpApp.
		Command("update", "Perform self-update and download the actual channel werf binary.").
		Action(func(c *kingpin.ParseContext) error {
			channelStr = normalizeChannel(channelStr)

			options := multiwerf.UpdateOptions{
				SkipSelfUpdate:          selfUpdate == "no",
				WithCache:               withCache,
				WithGC:                  withGC == "yes",
				TryRemoteChannelMapping: update == "yes",
				OutputFile:              updateOutputFile,
			}

			options.TryTrdl = getTryTrdl(tryTrdl, options.SkipSelfUpdate)

			if options.TryTrdl {
				done, err := tryExecTrdl(NewTrdlWerfUpdateCommand(groupStr, channelStr, os.Stdout, os.Stdout))
				if done {
					return err
				}
			}

			if updateInBackground {
				var args []string
				for _, arg := range os.Args[1:] {
					if arg == "--in-background" || strings.HasPrefix(arg, "--in-background=") {
						continue
					}
					args = append(args, arg)
				}

				cmd := exec.Command(os.Args[0], args...)
				if err := cmd.Start(); err != nil {
					fmt.Printf("command '%s' start failed: %s\n", strings.Join(append(os.Args[:0], args...), " "), err.Error())
					os.Exit(1)
				}

				if err := cmd.Process.Release(); err != nil {
					fmt.Printf("process release failed: %s\n", err.Error())
					return err
				}

				os.Exit(0)
			}

			// TODO add special error to exit with 1 and not print error message with kingpin
			if err := multiwerf.Update(groupStr, channelStr, options); err != nil {
				os.Exit(1)
			}

			return nil
		})
	updateCmd.Arg("MAJOR.MINOR", groupHelp).
		HintOptions(groupHintOptions...).
		Required().
		StringVar(&groupStr)
	updateCmd.Arg("CHANNEL", channelHelp).
		HintOptions(channels...).
		Default("stable").
		EnumVar(&channelStr, channelEnum...)
	updateCmd.Flag("with-cache", "Cache remote channel mapping between updates.").
		BoolVar(&withCache)
	updateCmd.Flag("self-update", selfUpdateHelp).
		Envar("MULTIWERF_SELF_UPDATE").
		Default(selfUpdateDefault).
		StringVar(&selfUpdate)
	updateCmd.Flag("try-trdl", tryTrdlHelp).
		Envar("MULTIWERF_TRY_TRDL").
		Default(tryTrdlDefault).
		StringVar(&tryTrdl)
	updateCmd.Flag("with-gc", withGCHelp).
		Envar("MULTIWERF_WITH_GC").
		Default(withGCDefault).
		StringVar(&withGC)
	updateCmd.Flag("update", updateHelp).
		Envar("MULTIWERF_UPDATE").
		Default(updateDefault).
		StringVar(&update)
	updateCmd.Flag("in-background", "Enable running process in background").
		BoolVar(&updateInBackground)
	updateCmd.Flag("output-file", "Save command output in file").
		StringVar(&updateOutputFile)
}

func useCommand(kpApp *kingpin.Application) {
	var (
		groupStr         string
		channelStr       string
		update           string
		selfUpdate       string
		withGC           string
		forceRemoteCheck bool
		shell            string
		asFile           bool
		tryTrdl          string
	)

	useCmd := kpApp.
		Command("use", "Generate the shell script that should be sourced to use the actual channel werf binary in the current shell session based on the local channel mapping.").
		Action(func(c *kingpin.ParseContext) error {
			channelStr = normalizeChannel(channelStr)
			options := multiwerf.UseOptions{
				ForceRemoteCheck:        forceRemoteCheck,
				AsFile:                  asFile,
				SkipSelfUpdate:          selfUpdate == "no",
				TryRemoteChannelMapping: update == "yes",
				WithGC:                  withGC == "yes",
			}

			options.TryTrdl = getTryTrdl(tryTrdl, options.SkipSelfUpdate)

			if options.TryTrdl {
				logPath := filepath.Join(os.Getenv("HOME"), ".multiwerf", "trdl", "log")
				if err := os.MkdirAll(filepath.Dir(logPath), os.ModePerm); err != nil {
					return fmt.Errorf("unable to create dir %s: %s", filepath.Dir(logPath), err)
				}

				logWriter, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
				if err != nil {
					return fmt.Errorf("unable to open file %q: %s", logPath, err)
				}
				defer logWriter.Close()

				done, err := tryExecTrdl(NewTrdlWerfUseCommand(groupStr, channelStr, os.Stdout, logWriter, asFile))
				if done {
					return err
				}
			}

			if err := multiwerf.Use(groupStr, channelStr, shell, options); err != nil {
				os.Exit(1)
			}

			return nil
		})
	useCmd.Arg("MAJOR.MINOR", groupHelp).
		HintOptions(groupHintOptions...).
		Required().
		StringVar(&groupStr)
	useCmd.Arg("CHANNEL", channelHelp).
		HintOptions(channels...).
		Default("stable").
		EnumVar(&channelStr, channelEnum...)
	useCmd.Flag("force-remote-check", "Do not use '--with-cache' option with background multiwerf update command.").
		BoolVar(&forceRemoteCheck)
	useCmd.Flag("shell", "Set to 'cmdexe', 'powershell' or use the default behaviour that is compatible with any unix shell.").
		Default(shellDefault).
		EnumVar(&shell, []string{"default", "cmdexe", "powershell"}...)
	useCmd.Flag("as-file", "Create the script and print the path for sourcing.").
		BoolVar(&asFile)
	useCmd.Flag("self-update", selfUpdateHelp).
		Envar("MULTIWERF_SELF_UPDATE").
		Default(selfUpdateDefault).
		StringVar(&selfUpdate)
	useCmd.Flag("try-trdl", tryTrdlHelp).
		Envar("MULTIWERF_TRY_TRDL").
		Default(tryTrdlDefault).
		StringVar(&tryTrdl)
	useCmd.Flag("with-gc", withGCHelp).
		Envar("MULTIWERF_WITH_GC").
		Default(withGCDefault).
		StringVar(&withGC)
	useCmd.Flag("update", updateHelp).
		Envar("MULTIWERF_UPDATE").
		Default(updateDefault).
		StringVar(&update)
}

func werfPathCommand(kpApp *kingpin.Application) {
	var (
		groupStr   string
		channelStr string
	)

	werfPathCmd := kpApp.
		Command("werf-path", "Print the actual channel werf binary path based on the local channel mapping.").
		Action(func(c *kingpin.ParseContext) error {
			channelStr = normalizeChannel(channelStr)

			err := multiwerf.WerfPath(groupStr, channelStr)
			if err != nil {
				os.Exit(1)
			}
			return nil
		})
	werfPathCmd.Arg("MAJOR.MINOR", groupHelp).
		HintOptions(groupHintOptions...).
		Required().
		StringVar(&groupStr)
	werfPathCmd.Arg("CHANNEL", channelHelp).
		HintOptions(channels...).
		Default("stable").
		EnumVar(&channelStr, channelEnum...)
}

func werfExecCommand(kpApp *kingpin.Application) {
	var groupStr string
	var channelStr string
	var werfArgs []string

	werfExecCmd := kpApp.
		Command("werf-exec", "Exec the actual channel werf binary based on the local channel mapping.").
		Action(func(c *kingpin.ParseContext) error {
			channelStr = normalizeChannel(channelStr)

			err := multiwerf.WerfExec(groupStr, channelStr, werfArgs)
			if err != nil {
				os.Exit(1)
			}
			return nil
		})
	werfExecCmd.Arg("MAJOR.MINOR", groupHelp).
		HintOptions(groupHintOptions...).
		Required().
		StringVar(&groupStr)
	werfExecCmd.Arg("CHANNEL", channelHelp).
		HintOptions(channels...).
		Default("stable").
		EnumVar(&channelStr, channelEnum...)
	werfExecCmd.Arg("WERF_ARGS", "Pass args to werf binary.").
		StringsVar(&werfArgs)
}

func werfGCCommand(kpApp *kingpin.Application) {
	kpApp.
		Command("gc", "Run garbage collection.").
		Action(func(c *kingpin.ParseContext) error {
			err := multiwerf.GC()
			if err != nil {
				_, _ = fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			return nil
		})
}

func versionCommand(kpApp *kingpin.Application) *kingpin.CmdClause {
	return kpApp.Command("version", "Show version.").Action(func(c *kingpin.ParseContext) error {
		fmt.Printf("%s %s\n", app.AppName, app.Version)
		return nil
	})
}

func normalizeChannel(value string) string {
	switch value {
	case "rc", "early-access":
		return "ea"
	default:
		return value
	}
}
