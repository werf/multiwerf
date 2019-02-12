package main

import (
	"fmt"
	"os"

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
	updateCmd.Arg("version", "Desired MAJOR.MINOR parts of a version").
		HintOptions("1.0", "1.1").
		Required().
		StringVar(&versionStr)
	updateCmd.Arg("channel", "Channel is one of alpha|beta|rc|stable").
		HintOptions(multiwerf.AvailableChannels...).
		Default("stable").
		EnumVar(&channelStr, multiwerf.AvailableChannels...)

	// multiwerf use
	useCmd := kpApp.
		Command("use", "Check for latest PATCH version available for channel and return a source script with alias function.").
		Action(func(c *kingpin.ParseContext) error {
			// TODO add special error to exit with 1 and not print error message with kingpin
			err := multiwerf.Use(versionStr, channelStr, []string{})
			if err != nil {
				os.Exit(1)
			}
			return nil
		})
	useCmd.Arg("version", "Desired MAJOR.MINOR parts of a version").
		HintOptions("1.0", "1.1").
		Required().
		StringVar(&versionStr)
	useCmd.Arg("channel", "Channel is one of alpha|beta|rc|stable").
		HintOptions(multiwerf.AvailableChannels...).
		Default("stable").
		EnumVar(&channelStr, multiwerf.AvailableChannels...)

	kingpin.MustParse(kpApp.Parse(os.Args[1:]))
}
