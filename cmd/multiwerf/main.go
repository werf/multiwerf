package main

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/flant/multiwerf/pkg/app"
	"github.com/flant/multiwerf/pkg/multiwerf"
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

	var versionStr string
	var channelStr string
	var outputFormat string
	var forceRemoteCheck bool
	var shell = "default"

	// multiwerf update
	updateCmd := kpApp.
		Command("update", "Perform self-update and download the actual werf binary.").
		Action(func(c *kingpin.ParseContext) error {
			// TODO add special error to exit with 1 and not print error message with kingpin
			err := multiwerf.Update(versionStr, channelStr, []string{})
			if err != nil {
				os.Exit(1)
			}
			return nil
		})
	updateCmd.Arg("MAJOR.MINOR", "Selector of a release series. Examples: 1.0, 1.3.").
		HintOptions("1.0", "1.3").
		Required().
		StringVar(&versionStr)
	updateCmd.Arg("CHANNEL", "The minimum acceptable level of stability. One of: alpha|beta|rc|ea|stable.").
		HintOptions(multiwerf.AvailableChannels...).
		Default("stable").
		EnumVar(&channelStr, multiwerf.AvailableChannels...)

	// multiwerf use
	useCmd := kpApp.
		Command("use", "Print the script that should be sourced to use the actual werf binary in the current shell session.").
		Action(func(c *kingpin.ParseContext) error {
			// TODO add special error to exit with 1 and not print error message with kingpin
			if shell == "powershell" {
				color.NoColor = true
			}

			err := multiwerf.Use(versionStr, channelStr, forceRemoteCheck, shell)
			if err != nil {
				os.Exit(1)
			}
			return nil
		})
	useCmd.Arg("MAJOR.MINOR", "Selector of a release series. Examples: 1.0, 1.3.").
		HintOptions("1.0", "1.3").
		Required().
		StringVar(&versionStr)
	useCmd.Arg("CHANNEL", "Minimum acceptable level of stability. One of: alpha|beta|rc|ea|stable.").
		HintOptions(multiwerf.AvailableChannels...).
		Default("stable").
		EnumVar(&channelStr, multiwerf.AvailableChannels...)
	useCmd.Flag("force-remote-check", "Force check of `werf' versions in a remote storage (bintray). Do not reset delay file.").
		BoolVar(&forceRemoteCheck)
	useCmd.Flag("shell", "Set to 'powershell' or use default behaviour that is compatible with any unix shell.").
		Default(shell).
		EnumVar(&shell, []string{"default", "powershell"}...)

	// multiwerf use-script-path
	getUseScriptPathCmd := kpApp.
		Command("use-script-path", "Print the script path that should be sourced to use the actual werf binary in the current shell session.").
		Action(func(c *kingpin.ParseContext) error {
			err := multiwerf.GetUseScriptPath(versionStr, channelStr, shell)
			if err != nil {
				os.Exit(1)
			}
			return nil
		})
	getUseScriptPathCmd.Arg("MAJOR.MINOR", "Selector of a release series. Examples: 1.0, 1.3.").
		HintOptions("1.0", "1.3").
		Required().
		StringVar(&versionStr)
	getUseScriptPathCmd.Arg("CHANNEL", "The minimum acceptable level of stability. One of: alpha|beta|rc|ea|stable.").
		HintOptions(multiwerf.AvailableChannels...).
		Default("stable").
		EnumVar(&channelStr, multiwerf.AvailableChannels...)
	getUseScriptPathCmd.Flag("shell", "Set to 'cmdexe', 'powershell' or use default behaviour that is compatible with any unix shell.").
		Default(shell).
		EnumVar(&shell, []string{"default", "cmdexe", "powershell"}...)

	// multiwerf werf-path
	werfPathCmd := kpApp.
		Command("werf-path", "Print the actual werf binary path (based on local werf binaries).").
		Action(func(c *kingpin.ParseContext) error {
			err := multiwerf.WerfPath(versionStr, channelStr)
			if err != nil {
				os.Exit(1)
			}
			return nil
		})
	werfPathCmd.Arg("MAJOR.MINOR", "Selector of a release series. Examples: 1.0, 1.3.").
		HintOptions("1.0", "1.3").
		Required().
		StringVar(&versionStr)
	werfPathCmd.Arg("CHANNEL", "The minimum acceptable level of stability. One of: alpha|beta|rc|ea|stable.").
		HintOptions(multiwerf.AvailableChannels...).
		Default("stable").
		EnumVar(&channelStr, multiwerf.AvailableChannels...)

	var werfArgs []string

	// multiwerf werf-exec
	werfExecCmd := kpApp.
		Command("werf-exec", "Exec the actual werf binary (based on local werf binaries).").
		Action(func(c *kingpin.ParseContext) error {
			err := multiwerf.WerfExec(versionStr, channelStr, werfArgs)
			if err != nil {
				os.Exit(1)
			}
			return nil
		})
	werfExecCmd.Arg("MAJOR.MINOR", "Selector of a release series. Examples: 1.0, 1.3.").
		HintOptions("1.0", "1.3").
		Required().
		StringVar(&versionStr)
	werfExecCmd.Arg("CHANNEL", "The minimum acceptable level of stability. One of: alpha|beta|rc|ea|stable.").
		HintOptions(multiwerf.AvailableChannels...).
		Default("stable").
		EnumVar(&channelStr, multiwerf.AvailableChannels...)
	werfExecCmd.Arg("WERF_ARGS", "Pass args to werf").
		StringsVar(&werfArgs)

	// multiwerf available-releases
	releasesCmd := kpApp.
		Command("available-releases", "Show available major.minor versions or available versions for each channel or exact version for major.minor and channel.").
		Action(func(c *kingpin.ParseContext) error {
			// TODO add special error to exit with 1 and not print error message with kingpin
			err := multiwerf.AvailableReleases(versionStr, channelStr, outputFormat)
			if err != nil {
				os.Exit(1)
			}
			return nil
		})
	releasesCmd.Arg("MAJOR.MINOR", "Selector of a release series. Examples: 1.0, 1.3.").
		HintOptions("1.0", "1.3").
		StringVar(&versionStr)
	releasesCmd.Arg("CHANNEL", "The minimum acceptable level of stability. One of: alpha|beta|rc|ea|stable.").
		HintOptions(multiwerf.AvailableChannels...).
		EnumVar(&channelStr, multiwerf.AvailableChannels...)
	releasesCmd.Flag("output", "Output format. One of: text|json.").
		Short('o').
		Default("text").
		EnumVar(&outputFormat, []string{"text", "json"}...)

	kingpin.MustParse(kpApp.Parse(os.Args[1:]))
}
