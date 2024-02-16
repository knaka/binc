package golang

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/knaka/binc/lib/common"
	. "github.com/knaka/go-utils"
	"github.com/samber/lo"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const goExt = ".go"

func compile(goPath string) (exePath string, err error) {
	defer Catch(&err)
	// Due to an inconvenient behavior of filepath.Join(), which removes the trailing dot, this approach is used instead.
	targetPath := fmt.Sprintf(".%c%s", filepath.Separator, filepath.Base(goPath))
	buildArgs := []string{"-tags", "", targetPath}
	var fileInfoList []*FileInfo
	var baseWithoutExt string
	if stat := V(os.Stat(goPath)); stat.IsDir() {
		baseWithoutExt = filepath.Base(goPath)
		files := V(filepath.Glob(filepath.Join(goPath, "*.go")))
		for _, file := range files {
			fileInfoList = append(fileInfoList, V(getGoFileInfo(file)))
		}
		sort.Slice(fileInfoList, func(i, j int) bool {
			if fileInfoList[i].Size == fileInfoList[j].Size {
				return fileInfoList[i].Hash < fileInfoList[j].Hash
			}
			return fileInfoList[i].Size < fileInfoList[j].Size
		})
	} else {
		goFileBase := filepath.Base(goPath)
		baseWithoutExt = goFileBase[:len(goFileBase)-len(filepath.Ext(goFileBase))]
		fileInfoList = append(fileInfoList, V(getGoFileInfo(goPath)))
	}
	buildInfo := newBuildInfoWithFiles(
		V(goEnv()).Version,
		buildArgs,
		fileInfoList,
	)
	exePath = V(common.CachedExePath(buildInfo.Hash, baseWithoutExt))
	// If the cache binary is not found, build it.
	if _, err := os.Stat(exePath); err != nil {
		prevWd := V(os.Getwd())
		V0(os.Chdir(filepath.Dir(goPath)))
		defer (func() { Ignore(os.Chdir(prevWd)) })()
		buildCommand := []string{"build"}
		buildCommand = append(buildCommand, "-o", exePath)
		buildCommand = append(buildCommand, buildArgs...)
		cmd := exec.Command(V(goCmd()), buildCommand...)
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		V0(cmd.Run())
		V0(os.Chdir(prevWd))
		buildInfoJson := V(json.Marshal(buildInfo))
		V0(os.WriteFile(V(common.InfoFilePath(buildInfo.Hash)), buildInfoJson, 0644))
	}
	return exePath, nil
}

// --------

type GoFileManager struct {
	goFilePaths []string
}

var _ common.Manager = &GoFileManager{}

func (m *GoFileManager) GetLinkBases() (linkPaths []string) {
	linkPaths = make([]string, len(m.goFilePaths))
	for i, goFilePath := range m.goFilePaths {
		goFileBase := filepath.Base(goFilePath)
		linkPaths[i] = goFileBase[:len(goFileBase)-len(filepath.Ext(goFileBase))]
	}
	return linkPaths
}

func (m *GoFileManager) Run(args []string) (err error) {
	defer Catch(&err)
	cmdBase := filepath.Base(args[0])
	for _, goFilePath := range m.goFilePaths {
		if filepath.Base(goFilePath) != cmdBase+goExt {
			continue
		}
		exePath := V(compile(goFilePath))
		cmd := exec.Command(exePath, args[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	return errors.New(fmt.Sprintf("no matching go file found: %s", args[0]))
}

// CanRun checks if the command can be run by this manager.
func (m *GoFileManager) CanRun(cmdBase string) bool {
	for _, goFilePath := range m.goFilePaths {
		if filepath.Base(goFilePath) == cmdBase+goExt {
			return true
		}
	}
	return false
}

func newGoFileManager(dirPath string) common.Manager {
	if _, err := goCmd(); err != nil {
		return nil
	}
	matchedPaths := V(filepath.Glob(filepath.Join(dirPath, "*"+goExt)))
	matchedPaths = lo.Filter(matchedPaths, func(goFilePath string, _ int) bool {
		return !strings.HasPrefix(filepath.Base(goFilePath), "_") &&
			!strings.HasPrefix(filepath.Base(goFilePath), ".")
	})
	if len(matchedPaths) == 0 {
		return nil
	}
	return &GoFileManager{
		goFilePaths: matchedPaths,
	}
}

// --------

type GoDirManager struct {
	mainDirPaths []string
}

var _ common.Manager = &GoDirManager{}

func (m *GoDirManager) CanRun(cmdBase string) bool {
	for _, mainDirPath := range m.mainDirPaths {
		if filepath.Base(mainDirPath) == cmdBase {
			return true
		}
	}
	return false
}

func (m *GoDirManager) Run(args []string) (err error) {
	defer Catch(&err)
	cmdBase := filepath.Base(args[0])
	//goland:noinspection GoDeferInLoop
	for _, mainDirPath := range m.mainDirPaths {
		if filepath.Base(mainDirPath) != cmdBase {
			continue
		}
		exePath := V(compile(mainDirPath))
		cmd := exec.Command(exePath, args[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	return errors.New(fmt.Sprintf("no matching go main directory found: %s", args[0]))
}

func (m *GoDirManager) GetLinkBases() (linkPaths []string) {
	linkPaths = make([]string, len(m.mainDirPaths))
	for i, mainDirPath := range m.mainDirPaths {
		linkPaths[i] = filepath.Base(mainDirPath)
	}
	return linkPaths
}

func newGoDirManager(dirPath string) common.Manager {
	if _, err := goCmd(); err != nil {
		return nil
	}
	var goMainDirPaths []string
	for _, dirEntry := range V(os.ReadDir(dirPath)) {
		if !dirEntry.IsDir() {
			continue
		}
		mainDirPath := filepath.Join(dirPath, dirEntry.Name())
		if strings.HasPrefix(filepath.Base(mainDirPath), "_") ||
			strings.HasPrefix(filepath.Base(mainDirPath), ".") {
			continue
		}
		matched := V(filepath.Glob(filepath.Join(mainDirPath, "*"+goExt)))
		if len(matched) == 0 {
			continue
		}
		goMainDirPaths = append(goMainDirPaths, mainDirPath)
	}
	return &GoDirManager{
		mainDirPaths: goMainDirPaths,
	}
}

// --------

func init() {
	common.RegisterManagerFactory(
		newGoFileManager,
		100,
	)
	common.RegisterManagerFactory(
		newGoDirManager,
		100,
	)
}
