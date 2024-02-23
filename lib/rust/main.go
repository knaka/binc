package rust

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/knaka/binc/lib/common"
	. "github.com/knaka/go-utils"
	"github.com/samber/lo"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

type CargoScriptManager struct {
	exeLikePaths []string
}

var _ common.Manager = &CargoScriptManager{}

// Sort in descending order of length.
var extensions = []string{
	".cargo.rs",
	".rs",
}

func (m *CargoScriptManager) GetCommandBaseInfoList() (infoList []*common.CommandBaseInfo) {
outer:
	for _, exeLikePath := range m.exeLikePaths {
		exeLikeBase := filepath.Base(exeLikePath)
		stat, err := os.Stat(exeLikePath)
		if err != nil {
			continue
		}
		if stat.IsDir() {
			infoList = append(infoList, &common.CommandBaseInfo{
				CmdBase:    exeLikeBase,
				SourcePath: exeLikePath,
			})
		} else {
			for _, ext := range extensions {
				if strings.HasSuffix(exeLikeBase, ext) {
					infoList = append(infoList, &common.CommandBaseInfo{
						CmdBase:    exeLikeBase[:len(exeLikeBase)-len(ext)],
						SourcePath: exeLikePath,
					})
					continue outer
				}
			}
		}
	}
	return infoList
}

func (m *CargoScriptManager) CanRun(cmdBase string) bool {
	for _, exeLikePath := range m.exeLikePaths {
		exeLikeBase := filepath.Base(exeLikePath)
		stat, err := os.Stat(exeLikePath)
		if err != nil {
			continue
		}
		if stat.IsDir() {
			if filepath.Base(exeLikePath) == cmdBase {
				return true
			}
		} else {
			for _, ext := range extensions {
				if strings.HasSuffix(exeLikeBase, ext) {
					if exeLikeBase == cmdBase+ext ||
						exeLikeBase[:len(exeLikeBase)-len(ext)] == cmdBase {
						return true
					}
				}
			}
		}
	}
	return false
}

func containsCargoScriptBlockStartMarker(rsFilePath string) bool {
	in := V(os.Open(rsFilePath))
	defer func() { V0(in.Close()) }()
	buf := make([]byte, 1024)
	n := V(in.Read(buf))
	return strings.Contains(string(buf[:n]), "{- cargo:")
}

func findCargoFile(exeLikePath string) (cargoFilePath string, err error) {
	defer Catch(&err)
	dirPathPrev := exeLikePath
	dirPath := filepath.Dir(exeLikePath)
	for dirPath != dirPathPrev {
		cargoFilePaths := V(filepath.Glob(filepath.Join(dirPath, "Cargo.toml")))
		if len(cargoFilePaths) > 0 {
			return cargoFilePaths[0], nil
		}
		dirPathPrev = dirPath
		dirPath = filepath.Dir(dirPath)
	}
	return "", errors.New("no cargo file found")
}

func build(cargoFilePath string, cmdBase string) (exePath string, err error) {
	defer Catch(&err)
	cargoFileDir := filepath.Dir(cargoFilePath)
	wd := V(os.Getwd())
	defer (func() { Ignore(os.Chdir(wd)) })()
	V0(os.Chdir(cargoFileDir))
	cmd := exec.Command(V(cargoCmd()), "build", cmdBase)
	readCloser := V(cmd.StdoutPipe())
	defer (func() { Ignore(readCloser.Close()) })()
	go (func() {
		scanner := bufio.NewScanner(readCloser)
		for {
			if !scanner.Scan() {
				break
			}
			txt := scanner.Text()
			if !strings.Contains(txt, "Up to date") {
				Ignore(os.Stderr.Write([]byte(txt + "\n")))
			}
		}
	})()
	cmd.Stderr = os.Stderr
	V0(cmd.Run())
	cmd = exec.Command(V(cargoCmd()), "list-bin", cmdBase)
	bufExePath := V(cmd.Output())
	builtExePath := strings.TrimSpace(string(bufExePath))
	V0(os.Chdir(wd))
	return builtExePath, nil
}

func (m *CargoScriptManager) Run(args []string, shouldRebuild bool) (err error) {
	defer Catch(&err)
	cmdBase := filepath.Base(args[0])
	for _, exeLikePath := range m.exeLikePaths {
		exeLikeBase := filepath.Base(exeLikePath)
		stat, err := os.Stat(exeLikePath)
		if err != nil {
			continue
		}
		if stat.IsDir() {
			if filepath.Base(exeLikePath) == cmdBase {
				if cargoFilePath, err := findCargoFile(exeLikePath); err == nil {
					cmd := exec.Command(V(cargoCmd()), append([]string{
						"-Z", "unstable-options",
						"-C", filepath.Dir(cargoFilePath),
						"run", "--quiet", "--bin", cmdBase,
					}, args[1:]...)...)
					cmd.Stdin = os.Stdin
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					return cmd.Run()
				} else {
					return errors.New("no cargo file found")
				}
			}
		} else {
			for _, ext := range extensions {
				if exeLikeBase == cmdBase+ext {
					if cargoFilePath, err := findCargoFile(exeLikePath); err == nil {
						cmd := exec.Command(V(cargoCmd()), append([]string{
							"-Z", "unstable-options",
							"-C", filepath.Dir(cargoFilePath),
							"run", "--quiet", "--bin", cmdBase,
						}, args[1:]...)...)
						cmd.Stdin = os.Stdin
						cmd.Stdout = os.Stdout
						cmd.Stderr = os.Stderr
						return cmd.Run()
					} else {
						// cargo +nightly -q -Zscript
						cmd := exec.Command(V(cargoCmd()), append([]string{
							"+nightly",
							"-Z", "script",
							"--quiet",
							exeLikePath,
						}, args[1:]...)...)
						cmd.Stdin = os.Stdin
						cmd.Stdout = os.Stdout
						cmd.Stderr = os.Stderr
						return cmd.Run()
						//} else {
						//	return errors.New("no matching cargo target found")
					}
				}
			}
		}
	}
	return errors.New(fmt.Sprintf("no matching rs file found: %s", args[0]))
}

var cargoCmd = sync.OnceValues(func() (cargoPath string, err error) {
	defer Catch(&err)
	return V(exec.LookPath("cargo")), nil
})

func newCargoScriptManager(dirPath string) common.Manager {
	if E(cargoCmd()) != nil {
		return nil
	}
	var exeLikePaths []string
	for _, dirEntry := range V(os.ReadDir(dirPath)) {
		if !dirEntry.IsDir() {
			continue
		}
		rsFiles := V(filepath.Glob(filepath.Join(dirPath, dirEntry.Name(), "*.rs")))
		if len(rsFiles) > 0 {
			exeLikePaths = append(exeLikePaths, filepath.Join(dirPath, dirEntry.Name()))
		}
	}
	for _, ext := range extensions {
		exeLikePaths = append(exeLikePaths, V(filepath.Glob(filepath.Join(dirPath, "*"+ext)))...)
	}
	exeLikePaths = lo.FindUniques(exeLikePaths)
	exeLikePaths = lo.Filter(exeLikePaths, func(rsFilePath string, _ int) bool {
		return !strings.HasPrefix(filepath.Base(rsFilePath), ".") &&
			!strings.HasPrefix(filepath.Base(rsFilePath), "_")
	})
	if len(exeLikePaths) == 0 {
		return nil
	}
	return &CargoScriptManager{
		exeLikePaths: exeLikePaths,
	}
}

func init() {
	common.RegisterManagerFactory(
		"Cargo Project Manager",
		newCargoScriptManager,
		50,
	)
}
