package haskell

import (
	"errors"
	"fmt"
	"github.com/knaka/binc/lib/common"
	. "github.com/knaka/go-utils"
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

func (m *CabalScriptManager) GetLinkBases() (linkPaths []string) {
	var linkBases []string
	for _, hsFilePath := range m.filePaths {
		hsFileBase := filepath.Base(hsFilePath)
		for _, ext := range extensions {
			if strings.HasSuffix(hsFileBase, ext) {
				linkBases = append(linkBases, hsFileBase[:len(hsFileBase)-len(ext)])
			}
		}
	}
	return linkBases
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

func (m *CabalScriptManager) Run(args []string) (err error) {
	defer Catch(&err)
	cmdBase := filepath.Base(args[0])
	for _, hsFilePath := range m.filePaths {
		for _, ext := range extensions {
			if filepath.Base(hsFilePath) == cmdBase+ext {
				cmd := exec.Command("cabal", "run", "--", hsFilePath)
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
		newCabalScriptManager,
		50,
	)
}
