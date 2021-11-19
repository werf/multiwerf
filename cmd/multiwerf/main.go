package main

import (
	"fmt"
	"os"
	"os/exec"
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

	tryTrdlDefault = "yes"
	tryTrdlHelp    = "Try to use system trdl package manager instad of multiwerf. Multiwerf is DEPRECATED, more info about trdl: https://github.com/werf/trdl. To disable trdl set to 'no'."

	autoInstallTrdlDefault = "on-self-update"
	autoInstallTrdlHelp    = "Automatically download trdl package manager and install into the system. Multiwerf is DEPRECATED, more info about trdl: https://github.com/werf/trdl. To disable auto download set to 'no'. Multiwerf will auto-download trdl by default unless self-updates is disabled by the --self-update='no' flag. It is possible to enable auto-download of trdl even if self-updates are disabled by setting option to 'yes' explicitly."

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

func getAutoInstallTrdlOption(rawInput string, tryTrdlOption, skipSelfUpdateOption bool) (bool, error) {
	switch rawInput {
	case "yes":
		return tryTrdlOption, nil
	case "on-self-update":
		return tryTrdlOption && !skipSelfUpdateOption, nil
	case "no":
		return false, nil
	default:
		return false, fmt.Errorf("bad --auto-install-trdl=%s option given, expected 'yes', 'no' or 'on-self-update'", rawInput)
	}
}

func getTryTrdlOption(rawInput string) (bool, error) {
	switch rawInput {
	case "yes":
		return true, nil
	case "no":
		return false, nil
	default:
		return false, fmt.Errorf("bad --try-trdl=%s option given, expected 'yes' or 'no'", rawInput)
	}
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
		autoInstallTrdl    string
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

			if value, err := getTryTrdlOption(tryTrdl); err != nil {
				return err
			} else {
				options.TryTrdl = value
			}

			if value, err := getAutoInstallTrdlOption(autoInstallTrdl, options.TryTrdl, options.SkipSelfUpdate); err != nil {
				return err
			} else {
				options.AutoInstallTrdl = value
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
	updateCmd.Flag("auto-install-trdl", autoInstallTrdlHelp).
		Envar("MULTIWERF_AUTO_INSTALL_TRDL").
		Default(autoInstallTrdlDefault).
		StringVar(&autoInstallTrdl)
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
		autoInstallTrdl  string
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

			if value, err := getTryTrdlOption(tryTrdl); err != nil {
				return err
			} else {
				options.TryTrdl = value
			}

			if value, err := getAutoInstallTrdlOption(autoInstallTrdl, options.TryTrdl, options.SkipSelfUpdate); err != nil {
				return err
			} else {
				options.AutoInstallTrdl = value
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
	useCmd.Flag("auto-install-trdl", autoInstallTrdlHelp).
		Envar("MULTIWERF_AUTO_INSTALL_TRDL").
		Default(autoInstallTrdlDefault).
		StringVar(&autoInstallTrdl)
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
		tryTrdl    string
	)

	werfPathCmd := kpApp.
		Command("werf-path", "Print the actual channel werf binary path based on the local channel mapping.").
		Action(func(c *kingpin.ParseContext) error {
			channelStr = normalizeChannel(channelStr)

			tryTrdlOption, err := getTryTrdlOption(tryTrdl)
			if err != nil {
				return err
			}

			if err := multiwerf.WerfPath(groupStr, channelStr, tryTrdlOption); err != nil {
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
	werfPathCmd.Flag("try-trdl", tryTrdlHelp).
		Envar("MULTIWERF_TRY_TRDL").
		Default(tryTrdlDefault).
		StringVar(&tryTrdl)
}

func werfExecCommand(kpApp *kingpin.Application) {
	var (
		groupStr   string
		channelStr string
		werfArgs   []string
		tryTrdl    string
	)

	werfExecCmd := kpApp.
		Command("werf-exec", "Exec the actual channel werf binary based on the local channel mapping.").
		Action(func(c *kingpin.ParseContext) error {
			channelStr = normalizeChannel(channelStr)

			tryTrdlOption, err := getTryTrdlOption(tryTrdl)
			if err != nil {
				return err
			}

			if err := multiwerf.WerfExec(groupStr, channelStr, werfArgs, tryTrdlOption); err != nil {
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
	werfExecCmd.Flag("try-trdl", tryTrdlHelp).
		Envar("MULTIWERF_TRY_TRDL").
		Default(tryTrdlDefault).
		StringVar(&tryTrdl)
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
