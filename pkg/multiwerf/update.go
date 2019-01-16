package multiwerf

import (
	"fmt"
	"path/filepath"

	"github.com/flant/multiwerf/pkg/bintray"
	"github.com/flant/multiwerf/pkg/app"
)

// 1. Check for new version → print version
// 2. Check hashes  existed binaries → print 'version stays'
// 3. Download and check if no binaries are existed — print 'updated to'
func updateBinary(version string, channel string, messages chan ActionMessage) {
	// Get file with versions from repository
	// download:
	// - github.com/master/release/VERSIONS-MAJOR.MINOR
	//   or if error:
	// - werf.io/versions/VERSIONS-MAJOR.MINOR
	// curl https://api.bintray.com/packages/flant/dapp/ruby2go !!!
	// returns json with versions field

	btClient := bintray.NewBintrayClient(app.BintraySubject, app.BintrayRepo, app.BintrayPackage)

	pkgInfo, err := btClient.GetPackage()
	if err != nil {
		messages <- ActionMessage{err: err, state: "exit"}
		return
	}

	versions := bintray.GetPackageVersions(pkgInfo)
	if len(versions) == 0 {
		messages <- ActionMessage{
			msg: fmt.Sprintf("No versions found for package %s/%s/%s", app.BintraySubject, app.BintrayRepo, app.BintrayPackage),
			state: "exit"}
	} else {
		messages <- ActionMessage{msg: fmt.Sprintf("Discover %d versions of package %s", len(versions), app.BintrayPackage), debug: true}
	}

	// Calc latest version for channel
	effectiveVersion, err := ChooseLatestVersion(version, channel, versions, AvailableChannels)
	if err != nil {
		messages <- ActionMessage{err: err, state: "exit"}
		return
	}
	messages <- ActionMessage{msg: fmt.Sprintf("Detect version '%s' as latest for channel %s@%s", effectiveVersion, version, channel)}

	werfStorageDir, err := TildeExpand(app.WerfStorageDir)
	if err != nil {
		messages <- ActionMessage{err: err, state: "exit"}
		return
	}

	dstPath := filepath.Join(werfStorageDir, effectiveVersion)
	messages <- ActionMessage{msg: fmt.Sprintf("dstPath is '%s'", dstPath), debug: true}

	files := ReleaseFiles(app.BintrayPackage, effectiveVersion, app.OsArch)

	messages <- ActionMessage{msg: fmt.Sprintf("files: %+v", files ), debug: true}

	// Check if binary already in .multiwerf/<version> and verify a hash
	prgExists, _ := FileExists(dstPath, files["program"])
	messages <- ActionMessage{msg: fmt.Sprintf("prg file %s exists %v. e:%v", files["program"], prgExists, err ), debug: true}
	hashExists, _:= FileExists(dstPath, files["hash"])
	messages <- ActionMessage{msg: fmt.Sprintf("hash file %s exists %v. e:%v", files["hash"], prgExists, err ), debug: true}
	if  prgExists && hashExists {
		// check hash
		match, err := VerifyReleaseFileHash(dstPath, files)
		if err != nil {
			messages <- ActionMessage{msg: fmt.Sprintf("Error while verifying existing file: %v", err ), debug: true}
		}
		if match {
			messages <- ActionMessage{msg: fmt.Sprintf("werf %s@%s stays at %s",
				version, channel, effectiveVersion), state: "done"}
			return
		}
	}

	// If no binary or hash not verified: download
	messages <- ActionMessage{msg: "Start downloading", debug: true}
	err = btClient.DownloadRelease(effectiveVersion, dstPath, files)
	if err != nil {
		messages <- ActionMessage{err: err, state: "exit"}
		return
	}

	// Check hash of the binary
	messages <- ActionMessage{msg: "Check hash...", debug: true}

	CalculateSHA256(filepath.Join(dstPath, "dappfile-yml"))


	// Not match — ERROR and grace exit
	messages <- ActionMessage{msg: "Hash not matched: exit now", err: fmt.Errorf("not matched"), debug: true}

	// If match — ok, exit 0
	// Message "Updated werf 1.1 to 1.1.2-alpha.1" or "werf 1.1@alpha updated to 1.1.2-alpha.1 released on 10.02.2019 13:45:11 UTC"
	messages <- ActionMessage{msg: "werf 1.1@alpha updated to 1.1.2-alpha.1 released on 10.02.2019 13:45:11 UTC", state: "done"}
	return
}


