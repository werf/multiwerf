package app

import (
	"os"
	"runtime"
	"strings"
	"time"

	"gopkg.in/alecthomas/kingpin.v2"
)

var AppName = "multiwerf"
var AppDescription = "version manager and updater for werf.io project"

var Version = "dev"

var SelfBintraySubject = "flant"
var SelfBintrayRepo = "multiwerf"
var SelfBintrayPackage = "multiwerf"

var BintraySubject = "flant"
var BintrayRepo = "werf"
var BintrayPackage = "werf"
var OsArch = strings.Join([]string{runtime.GOOS, runtime.GOARCH}, "-")
var StorageDir = "~/.multiwerf"
var Shell = "default"

var SelfUpdate = "yes"
var Update = "yes"
var DebugMessages = "no"

// An hour delay between checks for latest version of werf
var UpdateDelay = time.Duration(time.Hour)

// A 5 minute delay for alpha and beta releases
var AlphaBetaUpdateDelay = time.Duration(time.Minute * 5)

// 24 hour delay between check for latest version of multiwerf
var SelfUpdateDelay = time.Duration(24 * time.Hour)

// SetupGlobalSettings init global flags with default values
func SetupGlobalSettings(kpApp *kingpin.Application) {
	kpApp.Flag("bintray-subject", "subject part for bintray api").
		Hidden().
		Envar("MULTIWERF_BINTRAY_SUBJECT").
		Default(BintraySubject).
		StringVar(&BintraySubject)

	kpApp.Flag("bintray-repo", "repository part for bintray api").
		Hidden().
		Envar("MULTIWERF_BINTRAY_REPO").
		Default(BintrayRepo).
		StringVar(&BintrayRepo)

	kpApp.Flag("bintray-package", "package part for bintray api").
		Hidden().
		Envar("MULTIWERF_BINTRAY_PACKAGE").
		Default(BintrayPackage).
		StringVar(&BintrayPackage)

	// Default for os-arch is set at compile time
	kpApp.Flag("os-arch", "os and arch of binary (linux-amd64)").
		Hidden().
		Envar("MULTIWERF_OS_ARCH").
		Default(OsArch).
		StringVar(&OsArch)

	kpApp.Flag("shell", "set to 'powershell' or use default behaviour that is compatible with sh, bash and zsh").
		Envar("MULTIWERF_SHELL").
		Default(Shell).
		StringVar(&Shell)

	kpApp.Flag("storage-dir", "directory for store binaries (~/.multiwerf)").
		Hidden().
		Envar("MULTIWERF_STORAGE_DIR").
		Default(StorageDir).
		StringVar(&StorageDir)

	kpApp.Flag("self-update", "set to `no' to disable self update in use and update command").
		Envar("MULTIWERF_SELF_UPDATE").
		Default(SelfUpdate).
		StringVar(&SelfUpdate)

	kpApp.Flag("update", "set to `no' to disable werf update in use and update command").
		Envar("MULTIWERF_UPDATE").
		Default(Update).
		StringVar(&Update)

	kpApp.Flag("debug", "set to yes to turn on debug messages").
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
