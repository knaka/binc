package haskell

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/knaka/binc/lib/common"
	. "github.com/knaka/go-utils"
	"github.com/samber/lo"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

type CabalScriptManager struct {
	exeLikePaths []string
}

var _ common.Manager = &CabalScriptManager{}

// Sort in descending order of length.
var extensions = []string{
	".cabal.lhs",
	".cabal.hs",
	".lhs",
	".hs",
}

func (m *CabalScriptManager) GetCommandBaseInfoList() (infoList []*common.CommandBaseInfo) {
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
						CmdBase:    common.Camel2Kebab(exeLikeBase[:len(exeLikeBase)-len(ext)]),
						SourcePath: exeLikePath,
					})
					continue outer
				}
			}
		}
	}
	return infoList
}

func (m *CabalScriptManager) CanRun(cmdBase string) bool {
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
						common.Camel2Kebab(exeLikeBase[:len(exeLikeBase)-len(ext)]) == cmdBase {
						return true
					}
				}
			}
		}
	}
	return false
}

func ensureExeFile(hsFilePath string, cmdBase string, shouldRebuild bool) (exePath string, err error) {
	var fileInfoList []*common.FileInfo
	fileInfoList = append(fileInfoList, V(common.GetFileInfo(hsFilePath)))
	buildInfo := common.NewBuildInfo(
		"",  // TODO: Decide if the version of GHC or Cabal should be recorded
		nil, // Any arguments? Do all necessary information is in the Cabal Script file?
		fileInfoList,
	)
	exePath = V(common.CachedExePath(buildInfo.Hash, cmdBase))
	if _, err = os.Stat(exePath); err != nil || shouldRebuild {
		cmd := exec.Command(V(cabalCmd()), "build", hsFilePath)
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		Ensure0(cmd.Run())
		cmd = exec.Command(V(cabalCmd()), "list-bin", hsFilePath)
		bufExePath := V(cmd.Output())
		builtExePath := strings.TrimSpace(string(bufExePath))
		V0(os.MkdirAll(filepath.Dir(exePath), 0755))
		Ensure0(os.Rename(builtExePath, exePath))
		cmd = exec.Command(V(cabalCmd()), "clean", hsFilePath)
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		Ensure0(cmd.Run())
		buildInfoJson := V(json.Marshal(buildInfo))
		V0(os.WriteFile(V(common.InfoFilePath(buildInfo.Hash)), buildInfoJson, 0644))
		log.Println("built:", exePath)
	}
	return exePath, nil
}

func containsCabalScriptBlockStartMarker(hsFilePath string) bool {
	in := V(os.Open(hsFilePath))
	defer func() { V0(in.Close()) }()
	buf := make([]byte, 1024)
	n := V(in.Read(buf))
	return strings.Contains(string(buf[:n]), "{- cabal:")
}

func findCabalFile(exeLikePath string) (cabalFile string, err error) {
	defer Catch(&err)
	dirPathPrev := exeLikePath
	dirPath := filepath.Dir(exeLikePath)
	for dirPath != dirPathPrev {
		cabalFiles := Ensure(filepath.Glob(filepath.Join(dirPath, "*.cabal")))
		if len(cabalFiles) > 0 {
			return cabalFiles[0], nil
		}
		dirPathPrev = dirPath
		dirPath = filepath.Dir(dirPath)
	}
	return "", errors.New("no cabal file found")
}

func build(cabalFilePath string, cmdBase string) (exePath string, err error) {
	defer Catch(&err)
	cabalFileDir := filepath.Dir(cabalFilePath)
	wd := V(os.Getwd())
	defer (func() { Ignore(os.Chdir(wd)) })()
	V0(os.Chdir(cabalFileDir))
	cmd := exec.Command(V(cabalCmd()), "build", cmdBase)
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
	cmd = exec.Command(V(cabalCmd()), "list-bin", cmdBase)
	bufExePath := V(cmd.Output())
	builtExePath := strings.TrimSpace(string(bufExePath))
	V0(os.Chdir(wd))
	return builtExePath, nil
}

//goland:noinspection GoDeferInLoop
func (m *CabalScriptManager) Run(args []string, shouldRebuild bool) (err error) {
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
				if cabalFilePath, err := findCabalFile(exeLikePath); err == nil {
					builtExtPath := V(build(cabalFilePath, cmdBase))
					cmd := exec.Command(builtExtPath, args[1:]...)
					cmd.Stdin = os.Stdin
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					return cmd.Run()
				} else {
					return errors.New("no matching cabal target found")
				}
			}
		} else {
			for _, ext := range extensions {
				if exeLikeBase == cmdBase+ext ||
					exeLikeBase == common.Kebab2Camel(cmdBase)+ext {
					if containsCabalScriptBlockStartMarker(exeLikePath) {
						cmd := exec.Command(V(cabalCmd()), append([]string{"run", exeLikePath}, args[1:]...)...)
						cmd.Stdin = os.Stdin
						cmd.Stdout = os.Stdout
						cmd.Stderr = os.Stderr
						return cmd.Run()
					} else if cabalFilePath, err := findCabalFile(exeLikePath); err == nil {
						builtExePath := V(build(cabalFilePath, cmdBase))
						cmd := exec.Command(builtExePath, args[1:]...)
						cmd.Stdin = os.Stdin
						cmd.Stdout = os.Stdout
						cmd.Stderr = os.Stderr
						return cmd.Run()
					} else {
						return errors.New("no matching cabal target found")
					}
				}
			}
		}
	}
	return errors.New(fmt.Sprintf("no matching hs file found: %s", args[0]))
}

var cabalCmd = sync.OnceValues(func() (cabalPath string, err error) {
	defer Catch(&err)
	return V(exec.LookPath("cabal")), nil
})

func newCabalScriptManager(dirPath string) common.Manager {
	if E(cabalCmd()) != nil {
		return nil
	}
	var exeLikePaths []string
	for _, dirEntry := range V(os.ReadDir(dirPath)) {
		if !dirEntry.IsDir() {
			continue
		}
		hsFiles := V(filepath.Glob(filepath.Join(dirPath, dirEntry.Name(), "*.hs")))
		if len(hsFiles) > 0 {
			exeLikePaths = append(exeLikePaths, filepath.Join(dirPath, dirEntry.Name()))
		}
	}
	for _, ext := range extensions {
		exeLikePaths = append(exeLikePaths, V(filepath.Glob(filepath.Join(dirPath, "*"+ext)))...)
	}
	exeLikePaths = lo.FindUniques(exeLikePaths)
	exeLikePaths = lo.Filter(exeLikePaths, func(hsFilePath string, _ int) bool {
		return !strings.HasPrefix(filepath.Base(hsFilePath), ".") &&
			!strings.HasPrefix(filepath.Base(hsFilePath), "_")
	})
	if len(exeLikePaths) == 0 {
		return nil
	}
	return &CabalScriptManager{
		exeLikePaths: exeLikePaths,
	}
}

func init() {
	common.RegisterManagerFactory(
		"Cabal Executable-likes Manager",
		newCabalScriptManager,
		50,
	)
}
