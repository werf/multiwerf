package app

import (
	"os"
	"runtime"
	"strings"
	"time"

	"gopkg.in/alecthomas/kingpin.v2"
)

var AppName = "multiwerf"
var AppDescription = "werf version manager"

var Version = "dev"

var SelfBintraySubject = "flant"
var SelfBintrayRepo = "multiwerf"
var SelfBintrayPackage = "multiwerf"

var BintraySubject = "flant"
var BintrayRepo = "werf"
var BintrayPackage = "werf"
var OsArch = strings.Join([]string{runtime.GOOS, runtime.GOARCH}, "-")
var StorageDir = "~/.multiwerf"

var ChannelMappingUrl = "https://raw.githubusercontent.com/flant/werf/multiwerf/multiwerf.json"
var ChannelMappingPath string

var SelfUpdate = "yes"
var Update = "yes"
var DebugMessages = "no"

// An hour delay between checks for the latest version of werf
var UpdateDelay = time.Hour

// A 5 minute delay for alpha and beta releases
var AlphaBetaUpdateDelay = time.Minute * 5

// 2 hour delay between check for the latest version of multiwerf
var SelfUpdateDelay = 2 * time.Hour

// SetupGlobalSettings init global flags with default values
func SetupGlobalSettings(kpApp *kingpin.Application) {
	kpApp.Flag("channel-mapping-url", "The URL to specific remote channel mapping file.").
		Envar("MULTIWERF_CHANNEL_MAPPING_URL").
		Default(ChannelMappingUrl).
		StringVar(&ChannelMappingUrl)

	kpApp.Flag("channel-mapping-path", "The path to override default channel mapping file.").
		Envar("MULTIWERF_CHANNEL_MAPPING_PATH").
		Default(ChannelMappingPath).
		StringVar(&ChannelMappingPath)

	kpApp.Flag("bintray-subject", "The subject part for bintray api.").
		Hidden().
		Envar("MULTIWERF_BINTRAY_SUBJECT").
		Default(ChannelMappingPath).
		StringVar(&ChannelMappingPath)

	kpApp.Flag("bintray-repo", "The repository part for bintray api.").
		Hidden().
		Envar("MULTIWERF_BINTRAY_REPO").
		Default(BintrayRepo).
		StringVar(&BintrayRepo)

	kpApp.Flag("bintray-package", "The package part for bintray api.").
		Hidden().
		Envar("MULTIWERF_BINTRAY_PACKAGE").
		Default(BintrayPackage).
		StringVar(&BintrayPackage)

	// Default for os-arch is set at compile time
	kpApp.Flag("os-arch", "The pair of os and arch of binary separated by dash").
		Hidden().
		Envar("MULTIWERF_OS_ARCH").
		Default(OsArch).
		StringVar(&OsArch)

	kpApp.Flag("storage-dir", "The directory for stored binaries").
		Hidden().
		Envar("MULTIWERF_STORAGE_DIR").
		Default(StorageDir).
		StringVar(&StorageDir)

	kpApp.Flag("self-update", "Set to 'no' to disable self-update in use and update command.").
		Envar("MULTIWERF_SELF_UPDATE").
		Default(SelfUpdate).
		StringVar(&SelfUpdate)

	kpApp.Flag("update", "Set to 'no' to disable werf update in use and update command.").
		Envar("MULTIWERF_UPDATE").
		Default(Update).
		StringVar(&Update)

	kpApp.Flag("debug", "Set to 'yes' to turn on debug messages.").
		Envar("MULTIWERF_DEBUG").
		Default(DebugMessages).
		StringVar(&DebugMessages)

	// Render help for hidden flags
	kpApp.Flag("help-advanced", "Show help for advanced flags.").PreAction(func(context *kingpin.ParseContext) error {
		context, err := kpApp.ParseContext(os.Args[1:])
		if err != nil {
			return err
		}

		usageTemplate := `
{{define "FormatCommand"}}\
{{if .FlagSummary}} {{.FlagSummary}}{{end}}\
{{range .Args}} {{if not .Required}}[{{end}}<{{.Name}}>{{if .Value|IsCumulative}}...{{end}}{{if not .Required}}]{{end}}{{end}}\
{{end}}\

{{define "FormatUsage"}}\
{{template "FormatCommand" .}}{{if .Commands}} <command> [<args> ...]{{end}}
{{if .Help}}
{{.Help|Wrap 0}}\
{{end}}\

{{end}}\

usage: {{.App.Name}}{{template "FormatUsage" .App}}

Advanced flags:
{{range .Context.Flags}}\
{{if .Hidden}}\
{{if .Short}}-{{.Short|Char}}, {{end}}--{{.Name}}{{if not .IsBoolFlag}}={{.FormatPlaceHolder}}{{end}}
        {{.Help}}
{{end}}\
{{end}}\
`

		if err := kpApp.UsageForContextWithTemplate(context, 2, usageTemplate); err != nil {
			panic(err)
		}

		os.Exit(0)
		return nil
	}).Bool()
}
