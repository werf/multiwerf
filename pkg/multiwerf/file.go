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

func LoadHashFile(dir string, fileName string) (hashes map[string]string) {
	filePath := filepath.Join(dir, fileName)
	file, err := os.Open(filePath)

	if err != nil {
		return
	}
	defer file.Close()

	return LoadHashMap(file)
}

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

func DirExists(dir string) (bool, error) {
	info, err := os.Stat(dir)
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

func TildeExpand(path string) (string, error) {
	if len(path) == 0 || path[0] != '~' {
		return path, nil
	}

	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	return filepath.Join(usr.HomeDir, path[1:]), nil
}

// TODO transform to VerifyFileHash — return 4 states: no hash file, no target file, not match, match
func VerifyReleaseFileHash(dir string, hashFile string, targetFile string) (bool, error) {
	hashFileExists, err := FileExists(dir, hashFile)
	if err != nil {
		return false, err
	}
	if !hashFileExists {
		return false, nil
	}

	prgFileExists, err := FileExists(dir, targetFile)
	if err != nil {
		return false, err
	}
	if !prgFileExists {
		return false, nil
	}

	hashes := LoadHashFile(dir, hashFile)

	return VerifyReleaseFileHashFromHashes(dir, hashes, targetFile)
}

func VerifyReleaseFileHashFromHashes(dir string, hashes map[string]string, targetFile string) (bool, error) {
	//fmt.Printf("hashes: %+v", hashes)

	hashForFile, hasHash := hashes[targetFile]
	if !hasHash {
		return false, nil
	}

	//fmt.Printf("hashForFile: %+v", hashForFile)

	hash, err := CalculateSHA256(filepath.Join(dir, targetFile))
	if err != nil {
		return false, err
	}

	//fmt.Printf("hash calc: %+v", hash)

	if hash != hashForFile {
		return false, nil
	}

	return true, nil
}

func ExpandAndVerifyDirectoryPath(path string) (resPath string, err error) {
	resPath, err = TildeExpand(path)
	if err != nil {
		return
	}

	return resPath, nil
}
