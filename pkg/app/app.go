package app

import "gopkg.in/alecthomas/kingpin.v2"

var AppName = "multiwerf"
var AppDescription = "version manager and updater for werf.io project"

var Version = "dev"

var SelfBintraySubject = "flant"
var SelfBintrayRepo = "multiwerf"
var SelfBintrayPackage = "multiwerf"

var BintraySubject = "flant"
var BintrayRepo = "werf"
var BintrayPackage = "werf"
var OsArch = "linux-amd64"
var StorageDir = "~/.multiwerf"

var SelfUpdate = "yes"
var Update = "yes"
var DebugMessages = "no"

// SetupGlobalSettings init global flags with default values
func SetupGlobalSettings(kpApp *kingpin.Application) {
	kpApp.Flag("bintray-subject", "subject part for bintray api").
		Envar("MULTIWERF_BINTRAY_SUBJECT").
		Default(BintraySubject).
		StringVar(&BintraySubject)

	kpApp.Flag("bintray-repo", "repository part for bintray api").
		Envar("MULTIWERF_BINTRAY_REPO").
		Default(BintrayRepo).
		StringVar(&BintrayRepo)

	kpApp.Flag("bintray-package", "package part for bintray api").
		Envar("MULTIWERF_BINTRAY_PACKAGE").
		Default(BintrayPackage).
		StringVar(&BintrayPackage)

	// Default for os-arch is set at compile time
	kpApp.Flag("os-arch", "os and arch of binary (linux-amd64)").
		Envar("MULTIWERF_OS_ARCH").
		Default(OsArch).
		StringVar(&OsArch)

	kpApp.Flag("storage-dir", "directory for store binaries (~/.multiwerf)").
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
}
