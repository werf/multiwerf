package multiwerf

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"io"
	"math"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

const fileChunkSize = int64(1024 * 1024) // 1Mb blocks

// CalculateSHA256 returns SHA256 hash of filePath content
func CalculateSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)

	if err != nil {
		return "", err
	}
	defer file.Close()

	// calculate the file size
	info, _ := file.Stat()

	filesize := info.Size()

	chunkCount := uint64(math.Ceil(float64(filesize) / float64(fileChunkSize)))

	hash := sha256.New()

	for i := uint64(0); i < chunkCount; i++ {
		blocksize := fileChunkSize
		if i == chunkCount-1 {
			blocksize = filesize - int64(i)*fileChunkSize
		}
		buf := make([]byte, blocksize)

		n, err := file.Read(buf)
		if err != nil {
			return "", err
		}
		if int64(n) != blocksize {
			return "", fmt.Errorf("cannot read %d bytes. Only %d read.", blocksize, n)
		}
		n, err = hash.Write(buf)
		if err != nil {
			return "", err
		}
		if int64(n) != blocksize {
			return "", fmt.Errorf("cannot add %d bytes to hash. Only %d written.", blocksize, n)
		}
	}

	res := hash.Sum(nil)
	return fmt.Sprintf("%x", res), nil
}

// ReleaseFiles return a map with release filenames of package pkg for particular osArch and version
func ReleaseFiles(pkg string, version string, osArch string) map[string]string {
	files := map[string]string{
		"hash": "SHA256SUMS",
		"sig":  "SHA256SUMS.sig",
	}
	fileExt := ""
	if strings.Contains(osArch, "windows") {
		fileExt = ".exe"
	}
	prgFileName := fmt.Sprintf("%s-%s-%s%s", pkg, osArch, version, fileExt)
	//prgFileName = fmt.Sprintf("dappfile-yml-linux-amd64-%s", version)
	files["program"] = prgFileName
	return files
}

func IsReleaseFilesExist(dir string, files map[string]string) (bool, error) {
	exist := true
	for fileType, fileName := range files {
		if fileType == "sig" {
			continue
		}
		fExist, err := FileExists(dir, fileName)
		if err != nil {
			return false, err
		}
		if !fExist {
			exist = false
			break
		}
	}

	return exist, nil
}

// LoadHashFile opens a file and returns hashes map
func LoadHashFile(dir string, fileName string) (hashes map[string]string) {
	filePath := filepath.Join(dir, fileName)
	file, err := os.Open(filePath)

	if err != nil {
		return
	}
	defer file.Close()

	return LoadHashMap(file)
}

// LoadHashMap returns a map filename -> hash from reader
func LoadHashMap(hashesReader io.Reader) (hashes map[string]string) {
	hashes = map[string]string{}
	scanner := bufio.NewScanner(hashesReader)
	for scanner.Scan() {
		hashLine := scanner.Text()
		parts := strings.SplitN(hashLine, " ", 2)
		if len(parts[0]) != 64 {
			continue
		}

		if len(parts[1]) > 1 {
			hashes[parts[1][1:]] = parts[0]
		}
	}

	//if err := scanner.Err(); err != nil {
	//	log.Fatal(err)
	//}

	return
}

// FileExists returns true if file `name` is existing in `dir`
func FileExists(dir string, name string) (bool, error) {
	filePath := filepath.Join(dir, name)
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if info.Mode().IsRegular() {
		return true, err
	}
	return false, nil
}

// DirExists returns true if path is an existing directory
func DirExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if info.IsDir() {
		return true, err
	}
	return false, nil
}

// TildeExpand expands tilde prefix with home directory path
func TildeExpand(path string) (string, error) {
	if len(path) == 0 || path[0] != '~' {
		return path, nil
	}

	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		usr, err := user.Current()
		if err != nil {
			return "", err
		}
		homeDir = usr.HomeDir
	}

	return filepath.Join(homeDir, path[1:]), nil
}

// VerifyReleaseFileHash verify targetFile in dir accroding to hashFile in dir
//
// There are three states:
// - err != nil if something is missing
// false, nil if files are exists, we got hash for file and hashes is not matched
// true, nil is hashes is matched
func VerifyReleaseFileHash(dir string, hashFile string, targetFile string) (bool, error) {
	hashFileExists, err := FileExists(dir, hashFile)
	if err != nil {
		return false, err
	}
	if !hashFileExists {
		return false, fmt.Errorf("%s is not exists", hashFile)
	}

	prgFileExists, err := FileExists(dir, targetFile)
	if err != nil {
		return false, err
	}
	if !prgFileExists {
		return false, fmt.Errorf("%s is not exists", targetFile)
	}

	hashes := LoadHashFile(dir, hashFile)

	if len(hashes) == 0 {
		return false, fmt.Errorf("%s is empty or is not a checksums file", hashFile)
	}

	return VerifyReleaseFileHashFromHashes(dir, hashes, targetFile)
}

// VerifyReleaseFileHashFromHashes verifies targetFile hash with matched hash from hashes map
func VerifyReleaseFileHashFromHashes(dir string, hashes map[string]string, targetFile string) (bool, error) {
	hashForFile, hasHash := hashes[targetFile]
	if !hasHash {
		return false, fmt.Errorf("there is not checksum for %s", targetFile)
	}

	hash, err := CalculateSHA256(filepath.Join(dir, targetFile))
	if err != nil {
		return false, fmt.Errorf("sha256 failed for %s: %v", targetFile, err)
	}

	if hash != hashForFile {
		return false, nil
	}

	return true, nil
}

// ExpandPath expands tilde prefix and returns an absolute path
func ExpandPath(path string) (resPath string, err error) {
	expPath, err := TildeExpand(path)
	if err != nil {
		return
	}

	resPath, err = filepath.Abs(expPath)
	if err != nil {
		return
	}

	return resPath, nil
}
