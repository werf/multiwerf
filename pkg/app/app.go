package app

import "gopkg.in/alecthomas/kingpin.v2"

var AppName = "multiwerf"
var AppDescription = "version manager and updater for werf.io project"

var Version = "dev"

var BintraySubject = "flant"
var BintrayRepo = "werf"
var BintrayPackage = "werf"
var OsArch = "linux-amd64"
var StorageDir = "~/.multiwerf"

var DebugMessages = "no"


func SetupGlobalSettings(kpApp *kingpin.Application) {
	kpApp.Flag("debug", "turn on debug messages").
		Envar("MULTIWERF_DEBUG").
		Default(DebugMessages).
		StringVar(&DebugMessages)

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
}