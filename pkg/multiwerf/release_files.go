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
		//"sig":     "SHA256SUMS.sig", // TODO implement goreleaser lifecycle and verify gpg signing
		"program": ReleaseProgramFilename(pkg, version, osArch),
	}

	return files
}

func ReleaseProgramFilename(pkg, version, osArch string) string {
	fileExt := ""
	if strings.Contains(osArch, "windows") {
		fileExt = ".exe"
	}

	return fmt.Sprintf("%s-%s-%s%s", pkg, osArch, version, fileExt)
}

func IsReleaseFilesExist(dir string, files map[string]string) (bool, error) {
	exist := true
	for _, fileName := range files {
		// TODO implement goreleaser lifecycle and verify gpg signing
		//if fileType == "sig" {
		//	continue
		//}

		fExist, err := FileExists(filepath.Join(dir, fileName))
		if err != nil {
			return false, err
		} else if !fExist {
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

// FileExists returns true if path exists
func FileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if isNotExistError(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func DirExists(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		if isNotExistError(err) {
			return false, nil
		}

		return false, err
	}

	return fileInfo.IsDir(), nil
}

func isNotExistError(err error) bool {
	return os.IsNotExist(err) || IsNotADirectoryError(err)
}

func IsNotADirectoryError(err error) bool {
	return strings.HasSuffix(err.Error(), "not a directory")
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

func VerifyReleaseFileHash(messages chan ActionMessage, dir string, hashFile string, targetFile string) (bool, error) {
	if hashFileExists, err := FileExists(filepath.Join(dir, hashFile)); err != nil {
		return false, err
	} else if !hashFileExists {
		messages <- ActionMessage{
			msg:     fmt.Sprintf("The file %s does not exist", hashFile),
			msgType: WarnMsgType,
		}

		return false, nil
	}

	if prgFileExists, err := FileExists(filepath.Join(dir, targetFile)); err != nil {
		return false, err
	} else if !prgFileExists {
		messages <- ActionMessage{
			msg:     fmt.Sprintf("The file %s does not exist", targetFile),
			msgType: WarnMsgType,
		}

		return false, nil
	}

	hashes := LoadHashFile(dir, hashFile)
	if len(hashes) == 0 {
		messages <- ActionMessage{
			msg:     fmt.Sprintf("The file %s is empty or is not a checksum file", hashFile),
			msgType: WarnMsgType,
		}

		return false, nil
	}

	return VerifyReleaseFileHashFromHashes(messages, dir, hashes, targetFile)
}

func VerifyReleaseFileHashFromHashes(messages chan ActionMessage, dir string, hashes map[string]string, targetFile string) (bool, error) {
	hashForFile, hasHash := hashes[targetFile]
	if !hasHash {
		messages <- ActionMessage{
			msg:     fmt.Sprintf("There is not checksum for %s", targetFile),
			msgType: WarnMsgType,
		}

		return false, nil
	}

	hash, err := CalculateSHA256(filepath.Join(dir, targetFile))
	if err != nil {
		messages <- ActionMessage{
			msg:     fmt.Sprintf("sha256 failed for %s: %v", targetFile, err),
			msgType: WarnMsgType,
		}

		return false, nil
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
