package haskell

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/knaka/binc/lib/common"
	. "github.com/knaka/go-utils"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

type CabalScriptManager struct {
	cabalCmd  string
	filePaths []string
}

var _ common.Manager = &CabalScriptManager{}

var extensions = []string{
	".cabal.hs",
	".cabal.lhs",
}

func (m *CabalScriptManager) GetCommandBaseInfoList() (infoList []*common.CommandBaseInfo) {
	for _, hsFilePath := range m.filePaths {
		hsFileBase := filepath.Base(hsFilePath)
		for _, ext := range extensions {
			if strings.HasSuffix(hsFileBase, ext) {
				infoList = append(infoList, &common.CommandBaseInfo{
					CmdBase:    hsFileBase[:len(hsFileBase)-len(ext)],
					SourcePath: hsFilePath,
				})
			}
		}
	}
	return infoList
}

func (m *CabalScriptManager) CanRun(cmdBase string) bool {
	for _, hsFilePath := range m.filePaths {
		for _, ext := range extensions {
			if filepath.Base(hsFilePath) == cmdBase+ext {
				return true
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

func (m *CabalScriptManager) Run(args []string, shouldRebuild bool) (err error) {
	defer Catch(&err)
	cmdBase := filepath.Base(args[0])
	for _, hsFilePath := range m.filePaths {
		for _, ext := range extensions {
			if filepath.Base(hsFilePath) == cmdBase+ext {
				//cmd := exec.Command(m.cabalCmd, append([]string{"run", hsFilePath}, args[1:]...)...)
				exePath := V(ensureExeFile(hsFilePath, cmdBase, shouldRebuild))
				cmd := exec.Command(exePath, args[1:]...)
				cmd.Stdin = os.Stdin
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				return cmd.Run()
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
	cabalCmd_, err := cabalCmd()
	if err != nil {
		return nil
	}
	var matchedPaths []string
	for _, ext := range extensions {
		matchedPaths = append(matchedPaths, Ensure(filepath.Glob(filepath.Join(dirPath, "*"+ext)))...)
	}
	if len(matchedPaths) == 0 {
		return nil
	}
	return &CabalScriptManager{
		cabalCmd:  cabalCmd_,
		filePaths: matchedPaths,
	}
}

func init() {
	common.RegisterManagerFactory(
		"Cabal Script Manager",
		newCabalScriptManager,
		50,
	)
}
