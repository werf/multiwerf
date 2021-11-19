package trdlexec

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	uuid "github.com/satori/go.uuid"
	"golang.org/x/crypto/openpgp"
)

const (
	trdlPGPSigningKey = `-----BEGIN PGP PUBLIC KEY BLOCK-----

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
wjqHlF/rgsvSHLxXpN2Ks36VjXRxUUcgYAnZKEU=
=DrxJ
-----END PGP PUBLIC KEY BLOCK-----`

	trdlLatestURL    = `https://tuf.trdl.dev/targets/releases/VERSION/OS-ARCH/bin/BIN_NAME`
	trdlLatestSigURL = `https://tuf.trdl.dev/targets/signatures/VERSION/OS-ARCH/bin/BIN_NAME.sig`
)

func makeTrdlBinLatestURL() (string, string) {
	os := runtime.GOOS
	arch := runtime.GOARCH
	version := "0.2.1"
	binName := "trdl"
	if os == "windows" {
		binName += ".exe"
	}

	url := trdlLatestURL
	url = strings.Replace(url, "VERSION", version, 1)
	url = strings.Replace(url, "OS", os, 1)
	url = strings.Replace(url, "ARCH", arch, 1)
	url = strings.Replace(url, "BIN_NAME", binName, 1)

	sigUrl := trdlLatestSigURL
	sigUrl = strings.Replace(sigUrl, "VERSION", version, 1)
	sigUrl = strings.Replace(sigUrl, "OS", os, 1)
	sigUrl = strings.Replace(sigUrl, "ARCH", arch, 1)
	sigUrl = strings.Replace(sigUrl, "BIN_NAME", binName, 1)

	return url, sigUrl
}

func TryExecTrdl(cmd TrdlCommand, autoInstallTrdl bool) (bool, error) {
	installed, err := isTrdlInstalled(cmd.GetLogWriter())
	if err != nil {
		err = cmd.ConstructCommandError(err)
		cmd.LogCommandError(err)
		return false, err
	}
	if !installed {
		if !autoInstallTrdl {
			return false, fmt.Errorf("trdl is not installed into the system")
		}

		if err := unsetFlag(fmt.Sprintf("%s.%s.trdl_enabled", cmd.GetGroup(), cmd.GetChannel())); err != nil {
			err = cmd.ConstructCommandError(err)
			cmd.LogCommandError(err)
			return false, err
		}

		if err := installTrdl(cmd.GetLogWriter()); err != nil {
			err = cmd.ConstructCommandError(err)
			cmd.LogCommandError(err)
			return false, err
		}
	}

	repoInstalled, err := isWerfRepositoryInstalled()
	if err != nil {
		err = cmd.ConstructCommandError(err)
		cmd.LogCommandError(err)
		return false, err
	}
	if !repoInstalled {
		if err := unsetFlag(fmt.Sprintf("%s.%s.trdl_enabled", cmd.GetGroup(), cmd.GetChannel())); err != nil {
			err = cmd.ConstructCommandError(err)
			cmd.LogCommandError(err)
			return false, err
		}

		if err := installWerfRepository(cmd.GetLogWriter()); err != nil {
			err = cmd.ConstructCommandError(err)
			cmd.LogCommandError(err)
			return false, err
		}
	}

	isTrdlEnabled, err := isFlagSet(fmt.Sprintf("%s.%s.trdl_enabled", cmd.GetGroup(), cmd.GetChannel()))
	if err != nil {
		err = cmd.ConstructCommandError(err)
		cmd.LogCommandError(err)
		return false, err
	}

	if err := cmd.Exec(isTrdlEnabled); err != nil {
		commandErr := err

		retErr := cmd.ConstructCommandError(commandErr)
		if isTrdlEnabled {
			fmt.Fprintf(cmd.GetLogWriter(), "Trdl command failed, will fail multiwerf process\n")
		} else {
			cmd.LogCommandError(retErr)
			fmt.Fprintf(cmd.GetLogWriter(), "Trdl command failed, will fallback on multiwerf\n")
		}

		return isTrdlEnabled, retErr
	}

	if err := setFlag(fmt.Sprintf("%s.%s.trdl_enabled", cmd.GetGroup(), cmd.GetChannel())); err != nil {
		err = cmd.ConstructCommandError(err)
		cmd.LogCommandError(err)
		return false, err
	}

	return true, nil
}

func isTrdlInstalled(logWriter io.Writer) (bool, error) {
	_, err := exec.LookPath("trdl")
	if _, isExecErr := err.(*exec.Error); isExecErr {
		fmt.Fprintf(logWriter, "Trdl installation not found: %s\n", err)
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func checkSignature(logWriter io.Writer, path, sig string) error {
	keyring, err := openpgp.ReadArmoredKeyRing(bytes.NewReader([]byte(trdlPGPSigningKey)))
	if err != nil {
		return fmt.Errorf("unable to parse sig file %q key: %s", sig, err)
	}

	fileReader, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("unable to open file %q: %s", path, err)
	}

	sigData, err := os.ReadFile(sig)
	if err != nil {
		return fmt.Errorf("unable to read signature file %q: %s", sig, err)
	}

	signer, err := openpgp.CheckDetachedSignature(keyring, fileReader, bytes.NewReader(sigData))
	for _, id := range signer.Identities {
		fmt.Fprintf(logWriter, "Signed by %s\n", id.Name)
	}

	return err
}

func downloadFile(url, destFile string) error {
	out, err := os.Create(destFile)
	if err != nil {
		return fmt.Errorf("unable to create destination file %q: %s", destFile, err)
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("unable to issue http get for %q: %s", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("error while downloading %q: %s", url, err)
	}

	if err := out.Close(); err != nil {
		return fmt.Errorf("error writing %q: %s", destFile, err)
	}

	return nil
}

func installTrdl(logWriter io.Writer) error {
	selfBinPath, err := exec.LookPath(os.Args[0])
	if err != nil {
		return fmt.Errorf("unable to lookup self bin path %s: %s", os.Args[0], err)
	}

	destDir := filepath.Dir(selfBinPath)
	destFile := filepath.Join(destDir, "trdl")
	if runtime.GOOS == "windows" {
		destFile += ".exe"
	}
	url, sigUrl := makeTrdlBinLatestURL()

	destFileTemp := fmt.Sprintf("%s.%s", destFile, uuid.NewV4().String())

	fmt.Fprintf(logWriter, "Downloading %s into %s\n", url, destFileTemp)
	if err := downloadFile(url, destFileTemp); err != nil {
		return fmt.Errorf("unable to download %q: %s", url, err)
	}
	defer os.RemoveAll(destFileTemp)

	destFileSig := destFileTemp + ".sig"
	fmt.Fprintf(logWriter, "Downloading %s into %s\n", sigUrl, destFileSig)
	if err := downloadFile(sigUrl, destFileSig); err != nil {
		return fmt.Errorf("unable to download %q: %s", sigUrl, err)
	}
	defer os.RemoveAll(destFileSig)

	if err := checkSignature(logWriter, destFileTemp, destFileSig); err != nil {
		return fmt.Errorf("unable to check signature %s of downloaded trdl binary %s: %s", destFileSig, destFileTemp, err)
	}

	if err := os.Chmod(destFileTemp, 0755); err != nil {
		return fmt.Errorf("unable to chmod %q: %s", destFileTemp, err)
	}

	fmt.Fprintf(logWriter, "Creating trdl binary file %s\n", destFile)

	if err := os.Rename(destFileTemp, destFile); err != nil {
		return fmt.Errorf("unable to rename %q to %q: %s", destFileTemp, destFile, err)
	}

	return nil
}

func isWerfRepositoryInstalled() (bool, error) {
	cmd := exec.Command("trdl", "list")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("unable to list installed trdl repositories: %s:\n%s", err, strings.TrimSpace(string(output)))
	}

	repos := ParseRepoTable(output)

	for _, repo := range repos {
		if repo["Name"] == "werf" {
			if repo["URL"] == "https://tuf.werf.io" {
				return true, nil
			}
			return false, fmt.Errorf("unable to use \"werf\" trdl repository with unknown url %q: expected \"https://tuf.werf.io\"", repo["Url"])
		}
	}

	return false, nil
}

func ParseRepoTable(text []byte) []map[string]string {
	scanner := bufio.NewScanner(bytes.NewReader(text))

	var keys []string
	var res []map[string]string

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "Name ") {
			wordsScannder := bufio.NewScanner(bytes.NewReader([]byte(line)))
			wordsScannder.Split(bufio.ScanWords)
			for wordsScannder.Scan() {
				keys = append(keys, wordsScannder.Text())
			}
		} else {
			wordsScannder := bufio.NewScanner(bytes.NewReader([]byte(line)))
			wordsScannder.Split(bufio.ScanWords)

			data := make(map[string]string)

			i := 0
			for wordsScannder.Scan() {
				data[keys[i]] = wordsScannder.Text()
				i++
			}

			res = append(res, data)

		}
	}

	return res
}

func installWerfRepository(logWriter io.Writer) error {
	fmt.Fprintf(logWriter, "Adding werf repository into trdl ...\n")

	cmd := exec.Command("trdl", "add", "werf", "https://tuf.werf.io", "1", "b7ff6bcbe598e072a86d595a3621924c8612c7e6dc6a82e919abe89707d7e3f468e616b5635630680dd1e98fc362ae5051728406700e6274c5ed1ad92bea52a2")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("trdl add command failed: %s\n%s", err, strings.TrimSpace(string(output)))
	}

	return nil
}

func getFlagPath(name string) string {
	return filepath.Join(os.Getenv("HOME"), ".multiwerf", "trdl", name)
}

func isFlagSet(name string) (bool, error) {
	flagPath := getFlagPath(name)

	_, err := os.Stat(flagPath)
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("error accessing %q: %s", flagPath, err)
	}

	return true, nil
}

func setFlag(name string) error {
	flagPath := getFlagPath(name)

	if err := os.MkdirAll(filepath.Dir(flagPath), os.ModePerm); err != nil {
		return fmt.Errorf("unable to create dir %q: %s", filepath.Dir(flagPath))
	}

	if err := os.WriteFile(flagPath, []byte{}, os.ModePerm); err != nil {
		return fmt.Errorf("unable to write file %q: %s", flagPath, err)
	}

	return nil
}

func unsetFlag(name string) error {
	flagPath := getFlagPath(name)

	if err := os.RemoveAll(flagPath); err != nil {
		return fmt.Errorf("unable to remove file %q: %s", flagPath, err)
	}

	return nil
}

type TrdlCommand interface {
	Exec(isTrdlEnabled bool) error
	GetLogWriter() io.Writer
	ConstructCommandError(error) error
	LogCommandError(error)
	GetGroup() string
	GetChannel() string
}

type TrdlCommandCommonParams struct {
	Group     string
	Channel   string
	Stdout    io.Writer
	LogWriter io.Writer
}

func (command *TrdlCommandCommonParams) GetGroup() string {
	return command.Group
}

func (command *TrdlCommandCommonParams) GetChannel() string {
	return command.Channel
}

type TrdlWerfUpdateCommand struct {
	TrdlCommandCommonParams
}

func NewTrdlWerfUpdateCommand(group, channel string, stdout, logWriter io.Writer) *TrdlWerfUpdateCommand {
	return &TrdlWerfUpdateCommand{
		TrdlCommandCommonParams{
			Group:     group,
			Channel:   channel,
			Stdout:    stdout,
			LogWriter: logWriter,
		},
	}
}

func (command *TrdlWerfUpdateCommand) Write(p []byte) (int, error) {
	return command.LogWriter.Write(p)
}

func (command *TrdlWerfUpdateCommand) LogCommandError(err error) {
	fmt.Fprintf(command.LogWriter, "Trdl update werf command failed:\n%s\n", err)
}

func (command *TrdlWerfUpdateCommand) ConstructCommandError(err error) error {
	return err
}

func (command *TrdlWerfUpdateCommand) GetLogWriter() io.Writer {
	return command
}

func (command *TrdlWerfUpdateCommand) Exec(isTrdlEnabled bool) error {
	fmt.Fprintf(command.GetLogWriter(), "Running trdl update command ...\n")
	cmd := exec.Command("trdl", "update", "werf", command.Group, command.Channel)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("trdl update command failed: %s\n%s", err, strings.TrimSpace(string(output)))
	}

	fmt.Fprintf(command.Stdout, "%s", output)

	return nil
}

type TrdlWerfUseCommand struct {
	TrdlCommandCommonParams
	AsFile bool

	logBuf bytes.Buffer
}

func NewTrdlWerfUseCommand(group, channel string, stdout, logWriter io.Writer, asFile bool) *TrdlWerfUseCommand {
	return &TrdlWerfUseCommand{
		TrdlCommandCommonParams: TrdlCommandCommonParams{
			Group:     group,
			Channel:   channel,
			Stdout:    stdout,
			LogWriter: logWriter,
		},
		AsFile: asFile,
	}
}

func (command *TrdlWerfUseCommand) Write(p []byte) (int, error) {
	return command.logBuf.Write(p)
}

func (command *TrdlWerfUseCommand) GetLogWriter() io.Writer {
	return command
}

func (command *TrdlWerfUseCommand) LogCommandError(err error) {
	fmt.Fprintf(command.LogWriter, "Trdl use werf command logs:\n%s\n", err)
}

func (command *TrdlWerfUseCommand) ConstructCommandError(err error) error {
	return fmt.Errorf("%s\n%s", command.logBuf.String(), err)
}

func (command *TrdlWerfUseCommand) Exec(isTrdlEnabled bool) error {
	if !isTrdlEnabled {
		fmt.Fprintf(command.GetLogWriter(), "Running trdl update command ...\n")
		cmd := exec.Command("trdl", "update", "werf", command.Group, command.Channel)

		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("trdl update command failed: %s\n%s", err, strings.TrimSpace(string(output)))
		}

		fmt.Fprintf(command.GetLogWriter(), "%s", output)
	}

	fmt.Fprintf(command.GetLogWriter(), "Running trdl use command ...\n")
	cmd := exec.Command("trdl", "use", "werf", command.Group, command.Channel)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("trdl use command failed: %s\n%s", err, strings.TrimSpace(string(output)))
	}

	if command.AsFile {
		fmt.Fprintf(command.Stdout, "%s", output)
	} else {
		outfile := strings.TrimSpace(string(output))
		data, err := os.ReadFile(outfile)
		if err != nil {
			return fmt.Errorf("unable to read trdl use output file %q: %s", outfile, err)
		}
		fmt.Fprintf(command.Stdout, "%s", data)
	}

	return nil
}

type TrdlWerfBinPathCommand struct {
	TrdlCommandCommonParams

	logBuf bytes.Buffer
}

func NewTrdlWerfBinPathCommand(group, channel string, stdout, logWriter io.Writer) *TrdlWerfBinPathCommand {
	return &TrdlWerfBinPathCommand{
		TrdlCommandCommonParams: TrdlCommandCommonParams{
			Group:     group,
			Channel:   channel,
			Stdout:    stdout,
			LogWriter: logWriter,
		},
	}
}

func (command *TrdlWerfBinPathCommand) Write(p []byte) (int, error) {
	return command.logBuf.Write(p)
}

func (command *TrdlWerfBinPathCommand) GetLogWriter() io.Writer {
	return command
}

func (command *TrdlWerfBinPathCommand) LogCommandError(err error) {
	fmt.Fprintf(command.LogWriter, "Trdl bin-path werf command logs:\n%s\n", err)
}

func (command *TrdlWerfBinPathCommand) ConstructCommandError(err error) error {
	return fmt.Errorf("%s\n%s", command.logBuf.String(), err)
}

func (command *TrdlWerfBinPathCommand) Exec(isTrdlEnabled bool) error {
	fmt.Fprintf(command.GetLogWriter(), "Running trdl bin-path command ...\n")
	cmd := exec.Command("trdl", "bin-path", "werf", command.Group, command.Channel)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("trdl bin-path command failed: %s\n%s", err, strings.TrimSpace(string(output)))
	}
	path := strings.TrimSpace(string(output))

	fmt.Fprintf(command.Stdout, "%s\n", filepath.Join(path, "werf"))

	return nil
}

type TrdlWerfExecCommand struct {
	TrdlCommandCommonParams

	WerfArgs []string

	logBuf bytes.Buffer
}

func NewTrdlWerfExecCommand(group, channel string, werfArgs []string, stdout, logWriter io.Writer) *TrdlWerfExecCommand {
	return &TrdlWerfExecCommand{
		TrdlCommandCommonParams: TrdlCommandCommonParams{
			Group:     group,
			Channel:   channel,
			Stdout:    stdout,
			LogWriter: logWriter,
		},
		WerfArgs: werfArgs,
	}
}

func (command *TrdlWerfExecCommand) Write(p []byte) (int, error) {
	return command.logBuf.Write(p)
}

func (command *TrdlWerfExecCommand) GetLogWriter() io.Writer {
	return command
}

func (command *TrdlWerfExecCommand) LogCommandError(err error) {
	fmt.Fprintf(command.LogWriter, "Trdl exec werf command logs:\n%s\n", err)
}

func (command *TrdlWerfExecCommand) ConstructCommandError(err error) error {
	return fmt.Errorf("%s\n%s", command.logBuf.String(), err)
}

func (command *TrdlWerfExecCommand) Exec(isTrdlEnabled bool) error {
	fmt.Fprintf(command.GetLogWriter(), "Running trdl exec command ...\n")

	args := []string{"exec", "werf", command.Group, command.Channel}

	if len(command.WerfArgs) > 0 {
		args = append(args, "--")
		args = append(args, command.WerfArgs...)
	}

	cmd := exec.Command("trdl", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("trdl exec command failed: %s\n%s", err, strings.TrimSpace(string(output)))
	}

	fmt.Fprintf(command.Stdout, "%s", output)

	return nil
}
