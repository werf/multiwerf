package main

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/flant/multiwerf/pkg/app"
	"github.com/flant/multiwerf/pkg/multiwerf"
)

var (
	groupHelp        = "Selector of a release series. Examples: 1.0, 1.3."
	groupHintOptions = []string{"1.0", "1.3"}

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

	updateHelp = "Try to download remote channel mapping and sync channel werf version. To disable set to 'no'."

	selfUpdateHelp = "Perform multiwerf self-update. To disable set to 'no'."
)

func main() {
	kpApp := kingpin.New(app.AppName, fmt.Sprintf("%s %s: %s", app.AppName, app.Version, app.AppDescription))

	// global defaults
	app.SetupGlobalSettings(kpApp)

	//kpApp.HelpFlag

	// multiwerf version
	kpApp.Command("version", "Show version.").Action(func(c *kingpin.ParseContext) error {
		fmt.Printf("%s %s\n", app.AppName, app.Version)
		return nil
	})

	var groupStr string
	var channelStr string
	var update = "yes"
	var selfUpdate = "yes"
	var forceRemoteCheck bool
	var shell = "default"
	var withCache bool
	var asFile bool
	var setsid bool

	// multiwerf update
	updateCmd := kpApp.
		Command("update", "Perform self-update and download the actual channel werf binary.").
		Action(func(c *kingpin.ParseContext) error {
			channelStr = normalizeChannel(channelStr)

			options := multiwerf.UpdateOptions{
				SkipSelfUpdate:          selfUpdate == "no",
				WithCache:               withCache,
				TryRemoteChannelMapping: update == "yes",
				Setsid:                  setsid,
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
		Default(selfUpdate).
		StringVar(&selfUpdate)
	updateCmd.Flag("update", updateHelp).
		Envar("MULTIWERF_UPDATE").
		Default(update).
		StringVar(&update)
	updateCmd.Flag("setsid", "Enable running process in a new session, in background").
		BoolVar(&setsid)

	// multiwerf use
	useCmd := kpApp.
		Command("use", "Generate the shell script that should be sourced to use the actual channel werf binary in the current shell session based on the local channel mapping.").
		Action(func(c *kingpin.ParseContext) error {
			channelStr = normalizeChannel(channelStr)

			options := multiwerf.UseOptions{
				ForceRemoteCheck:        forceRemoteCheck,
				AsFile:                  asFile,
				SkipSelfUpdate:          selfUpdate == "no",
				TryRemoteChannelMapping: update == "yes",
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
		Default(shell).
		EnumVar(&shell, []string{"default", "cmdexe", "powershell"}...)
	useCmd.Flag("as-file", "Create the script and print the path for sourcing.").
		BoolVar(&asFile)
	useCmd.Flag("self-update", selfUpdateHelp).
		Envar("MULTIWERF_SELF_UPDATE").
		Default(selfUpdate).
		StringVar(&selfUpdate)
	useCmd.Flag("update", updateHelp).
		Envar("MULTIWERF_UPDATE").
		Default(update).
		StringVar(&update)

	// multiwerf werf-path
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

	var werfArgs []string

	// multiwerf werf-exec
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

	kingpin.MustParse(kpApp.Parse(os.Args[1:]))
}

func normalizeChannel(value string) string {
	switch value {
	case "rc", "early-access":
		return "ea"
	default:
		return value
	}
}
