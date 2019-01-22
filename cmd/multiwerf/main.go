package main

import (
	"fmt"
	"os"

	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/flant/multiwerf/pkg/app"
	"github.com/flant/multiwerf/pkg/multiwerf"
)

func main() {
	kpApp := kingpin.New(app.AppName, app.AppDescription)

	// global defaults
	app.SetupGlobalSettings(kpApp)

	// multiwerf version
	kpApp.Command("version", "Show version").Action(func(c *kingpin.ParseContext) error {
		fmt.Printf("%s %s\n", app.AppName, app.Version)
		return nil
	})

	var versionStr string
	var channelStr string

	// multiwerf update
	updateCmd := kpApp.
		Command("update", "update binary to the latest PATCH version").
		Action(func(c *kingpin.ParseContext) error {
			return multiwerf.Update(versionStr, channelStr, []string{})
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
		Command("use", "check for latest PATCH version and return a source script").
		Action(func(c *kingpin.ParseContext) error {
		return multiwerf.Use(versionStr, channelStr, []string{})
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
