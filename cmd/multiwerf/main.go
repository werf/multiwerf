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

	// multiwerf update
	updateCmd := kpApp.
		Command("update", "Update werf to the latest PATCH version available for channel.").
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
	updateCmd.Arg("CHANNEL", "Minimum acceptable level of stability. One of: alpha|beta|rc|ea|stable.").
		HintOptions(multiwerf.AvailableChannels...).
		Default("stable").
		EnumVar(&channelStr, multiwerf.AvailableChannels...)

	// multiwerf use
	useCmd := kpApp.
		Command("use", "Check for latest PATCH version available for channel and return a source script with alias function.").
		Action(func(c *kingpin.ParseContext) error {
			// TODO add special error to exit with 1 and not print error message with kingpin
			if app.Shell == "powershell" {
				color.NoColor = true
			}

			err := multiwerf.Use(versionStr, channelStr, forceRemoteCheck, []string{})
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
	releasesCmd.Arg("CHANNEL", "Minimum acceptable level of stability. One of: alpha|beta|rc|ea|stable.").
		HintOptions(multiwerf.AvailableChannels...).
		EnumVar(&channelStr, multiwerf.AvailableChannels...)
	releasesCmd.Flag("output", "Output format. One of: text|json.").
		Short('o').
		Default("text").
		EnumVar(&outputFormat, []string{"text", "json"}...)

	kingpin.MustParse(kpApp.Parse(os.Args[1:]))
}
