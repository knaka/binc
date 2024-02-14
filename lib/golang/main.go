package golang

import (
	"encoding/json"
	"fmt"
	"github.com/knaka/binc/lib/common"
	. "github.com/knaka/go-utils"
	"os"
	"os/exec"
	"path/filepath"
)

type GoFileManager struct {
	dir   string
	files []string
}

var _ common.Manager = &GoFileManager{}

func compileFile(goFile string) (exe string, err error) {
	defer Catch(&err)
	// filepath.Join() removes the trailing dot.
	targetFile := fmt.Sprintf(".%c%s", filepath.Separator, filepath.Base(goFile))
	buildArgs := []string{"-tags", "", targetFile}
	if err != nil {
		return
	}
	buildInfo := newBuildInfoWithFiles(
		Ensure(goEnv()).Version,
		buildArgs,
		[]*FileInfo{
			Ensure(getGoFileInfo(goFile)),
		},
	)
	exe = common.CacheFile(buildInfo.Hash)
	// If the cache binary is not found, build it.
	if _, err := os.Stat(exe); err != nil {
		prevDir := Ensure(os.Getwd())
		Ensure0(os.Chdir(filepath.Dir(goFile)))
		defer (func() { Ignore(os.Chdir(prevDir)) })()
		buildCommand := []string{"build"}
		buildCommand = append(buildCommand, "-o", exe)
		buildCommand = append(buildCommand, buildArgs...)
		cmd := exec.Command(Ensure(goCmd()), buildCommand...)
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		Ensure0(cmd.Run())
		Ensure0(os.Chdir(prevDir))
		buildInfoJson := Ensure(json.Marshal(buildInfo))
		Ensure0(os.WriteFile(common.InfoFile(buildInfo.Hash), buildInfoJson, 0644))
	}
	return exe, nil
}

func (m *GoFileManager) Run(args []string) error {
	baseName := filepath.Base(args[0])
	entries := Ensure(os.ReadDir(m.dir))
	for _, ent := range entries {
		if ent.IsDir() {
			if ent.Name() == baseName {
				dirAbs := filepath.Join(m.dir, baseName)
				exe := Ensure(compileFile(dirAbs))
				cmd := exec.Command(exe, args[1:]...)
				cmd.Args[0] = baseName
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				err := cmd.Run()
				if err != nil {
					return err
				}
				break
			}
			continue
		}
		if ent.Name() == baseName+".go" {
		}
	}
	//log.Println("dirAbs:", dirAbs)
	//log.Println("fileAbs:", fileAbs)
	return nil
}

// CanRun checks if the command can be run by this manager.
func (m *GoFileManager) CanRun(cmd string) bool {
	baseName := filepath.Base(cmd)
	for _, goFile := range m.files {
		if filepath.Base(goFile) == baseName+".go" {
			return true
		}
	}
	return false
}

func newGoFileManager(dir string) *GoFileManager {
	if _, err := goCmd(); err != nil {
		return nil
	}
	if stat, err := os.Stat(dir); err != nil || !stat.IsDir() {
		return nil
	}
	matches := Ensure(filepath.Glob(filepath.Join(dir, "*.go")))
	if len(matches) == 0 {
		return nil
	}
	return &GoFileManager{
		dir:   dir,
		files: matches,
	}
}

func init() {
	common.RegisterManagerFactory(
		func(dir string) common.Manager { return newGoFileManager(dir) },
		100,
	)
}
