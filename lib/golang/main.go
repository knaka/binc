package golang

import (
	"encoding/json"
	"fmt"
	"github.com/knaka/binc/lib/common"
	. "github.com/knaka/go-utils"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

type GoManager struct {
	dir string
}

var _ common.Manager = &GoManager{}

func compile(dir string) (exe string, err error) {
	defer Catch(&err)
	prevDir := Ensure(os.Getwd())
	Ensure0(os.Chdir(filepath.Dir(dir)))
	defer (func() { Ensure0(os.Chdir(prevDir)) })()

	goCmd := Ensure(findGoCmd())
	goEnv := Ensure(getGoEnv(goCmd))

	base := filepath.Base(dir)
	targetDir := fmt.Sprintf(".%c%s", filepath.Separator, base)
	buildArgs := []string{"-tags", "", targetDir}
	goFileInfoList, buildArgsWoTgt, err := getGoFileInfoList(buildArgs)
	if err != nil {
		return
	}
	buildInfo := newBuildInfoWithFiles(
		goEnv.Version,
		buildArgsWoTgt,
		goFileInfoList,
	)

	exe = cacheFile(buildInfo.Hash)
	if _, err := os.Stat(exe); err != nil {
		buildCommand := []string{"build"}
		buildCommand = append(buildCommand, "-o", exe)
		buildCommand = append(buildCommand, buildArgs...)
		buildCommand = append(buildCommand, targetDir)
		log.Println("7b580ab", buildCommand)
		cmd := exec.Command(goCmd, buildCommand...)
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		Ensure0(cmd.Run())
		buildInfoJson := Ensure(json.Marshal(buildInfo))
		Ensure0(os.WriteFile(infoFile(buildInfo.Hash), buildInfoJson, 0644))
	}

	return exe, nil
}

func (m *GoManager) CanRun(cmd string) bool {
	baseName := filepath.Base(cmd)
	entries := Ensure(os.ReadDir(m.dir))
	for _, ent := range entries {
		if ent.IsDir() {
			if ent.Name() == baseName {
				return true
			}
			continue
		}
	}
	return false
}

func (m *GoManager) Run(args []string) error {
	//log.Println("args:", args)
	baseName := filepath.Base(args[0])
	entries := Ensure(os.ReadDir(m.dir))
	for _, ent := range entries {
		if ent.IsDir() {
			if ent.Name() == baseName {
				dirAbs := filepath.Join(m.dir, baseName)
				exe := Ensure(compile(dirAbs))
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

func (m *GoManager) CanManage(dir string) bool {
	var err error
	defer Catch(&err)
	// If the directory contains at least one "*.go" file,
	matches := Ensure(filepath.Glob(filepath.Join(dir, "*.go")))
	if len(matches) > 0 {
		return false
	}
	matches = Ensure(filepath.Glob(filepath.Join(dir, "*", "*.go")))
	if len(matches) > 0 {
		return true
	}
	return false
}

func newManager(dir string) common.Manager {
	return &GoManager{
		dir: dir,
	}
}

func init() {
	common.RegisterManagerFactory(newManager, 100)
}
