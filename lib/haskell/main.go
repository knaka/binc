package haskell

import (
	"errors"
	"github.com/knaka/binc/lib/common"
	. "github.com/knaka/go-utils"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

type CabalScriptManager struct {
	files []string
}

var _ common.Manager = &CabalScriptManager{}

var extensions = []string{
	".cabal.hs",
	".cabal.lhs",
}

func (m *CabalScriptManager) CreateLinks() (err error) {
	defer Catch(&err)
	linksDir := V(common.LinksDir())
	for _, hsFile := range m.files {
		baseName := filepath.Base(hsFile)
		for _, ext := range extensions {
			if strings.HasSuffix(baseName, ext) {
				nameWithoutExt := baseName[:len(baseName)-len(ext)]
				link := filepath.Join(linksDir, nameWithoutExt)
				err = os.Remove(link)
				if err != nil && !os.IsNotExist(err) {
					return err
				}
				V0(os.Symlink("binc", link))
				break
			}
		}
	}
	return nil
}

func (m *CabalScriptManager) CanRun(cmd string) bool {
	baseName := filepath.Base(cmd)
	for _, hsFile := range m.files {
		for _, ext := range extensions {
			if filepath.Base(hsFile) == baseName+ext {
				return true
			}
		}
	}
	return false
}

func (m *CabalScriptManager) Run(args []string) (err error) {
	defer Catch(&err)
	baseName := filepath.Base(args[0])
	for _, hsFile := range m.files {
		for _, ext := range extensions {
			if filepath.Base(hsFile) == baseName+ext {
				cmd := exec.Command("cabal", "run", "--", hsFile)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				err := cmd.Run()
				if err != nil {
					return err
				}
				os.Exit(cmd.ProcessState.ExitCode())
			}
		}
	}
	return errors.New("file not found")
}

var cabalCmd = sync.OnceValues(func() (cabalPath string, err error) {
	defer Catch(&err)
	cabalPath = Ensure(exec.LookPath("cabal"))
	if err == nil {
		return cabalPath, nil
	}
	return "", errors.New("cabal command not found")
})

func newCabalScriptManager(dir string) common.Manager {
	if _, err := cabalCmd(); err != nil {
		return nil
	}
	if stat, err := os.Stat(dir); err != nil || !stat.IsDir() {
		return nil
	}
	var matches []string
	for _, ext := range extensions {
		matches = append(matches, Ensure(filepath.Glob(filepath.Join(dir, "*"+ext)))...)
	}
	if len(matches) == 0 {
		return nil
	}
	return &CabalScriptManager{
		files: matches,
	}
}

func init() {
	common.RegisterManagerFactory(
		newCabalScriptManager,
		50,
	)
}
