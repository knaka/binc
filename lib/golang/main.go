package golang

import (
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
)

const goExt = ".go"

func ensureExeFile(goTargetPath string, shouldRebuild bool) (exePath string, err error) {
	defer Catch(&err)
	buildArgsWoTgt := []string{"-tags", ""}
	var fileInfoList []*common.FileInfo
	var baseWithoutExt string
	if stat := V(os.Stat(goTargetPath)); stat.IsDir() {
		baseWithoutExt = filepath.Base(goTargetPath)
		files := V(filepath.Glob(filepath.Join(goTargetPath, "*"+goExt)))
		for _, file := range files {
			fileInfoList = append(fileInfoList, V(common.GetFileInfo(file)))
		}
	} else {
		goFileBase := filepath.Base(goTargetPath)
		baseWithoutExt = goFileBase[:len(goFileBase)-len(filepath.Ext(goFileBase))]
		fileInfoList = append(fileInfoList, V(common.GetFileInfo(goTargetPath)))
	}
	goModFilePath, err := findGoModFile(filepath.Dir(goTargetPath))
	if err == nil {
		fileInfoList = append(fileInfoList, V(common.GetFileInfo(goModFilePath)))
	}
	buildInfo := common.NewBuildInfo(
		V(goEnv()).Version,
		buildArgsWoTgt,
		fileInfoList,
	)
	exePath = V(common.CachedExePath(buildInfo.Hash, baseWithoutExt))
	// If the cache binary is not found, build it.
	if _, err := os.Stat(exePath); err != nil || shouldRebuild {
		prevWd := V(os.Getwd())
		V0(os.Chdir(filepath.Dir(goTargetPath)))
		defer (func() { Ignore(os.Chdir(prevWd)) })()
		buildCommand := []string{"build"}
		buildCommand = append(buildCommand, "-o", exePath)
		// Due to an inconvenient behavior of filepath.Join(), which removes the trailing dot, this approach is used instead.
		targetPath := fmt.Sprintf(".%c%s", filepath.Separator, filepath.Base(goTargetPath))
		buildArgs := append(buildArgsWoTgt, targetPath)
		buildCommand = append(buildCommand, buildArgs...)
		cmd := exec.Command(V(goCmd()), buildCommand...)
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		V0(cmd.Run())
		V0(os.Chdir(prevWd))
		buildInfoJson := V(json.Marshal(buildInfo))
		V0(os.WriteFile(V(common.InfoFilePath(buildInfo.Hash)), buildInfoJson, 0644))
		log.Println("built:", exePath)
	}
	return exePath, nil
}

// --------

type GoMainFileManager struct {
	goFilePaths []string
}

var _ common.Manager = &GoMainFileManager{}

func (m *GoMainFileManager) GetCommandBaseInfoList() (infoList []*common.CommandBaseInfo) {
	for _, goFilePath := range m.goFilePaths {
		goFileBase := filepath.Base(goFilePath)
		cmdBase := goFileBase[:len(goFileBase)-len(filepath.Ext(goFileBase))]
		infoList = append(infoList, &common.CommandBaseInfo{
			CmdBase:    cmdBase,
			SourcePath: goFilePath,
		})
	}
	return infoList
}

func (m *GoMainFileManager) Run(args []string, shouldRebuild bool) (err error) {
	defer Catch(&err)
	cmdBase := filepath.Base(args[0])
	for _, goFilePath := range m.goFilePaths {
		if filepath.Base(goFilePath) != cmdBase+goExt {
			continue
		}
		exePath := V(ensureExeFile(goFilePath, shouldRebuild))
		cmd := exec.Command(exePath, args[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	return errors.New(fmt.Sprintf("no matching go file found: %s", args[0]))
}

// CanRun checks if the command can be run by this manager.
func (m *GoMainFileManager) CanRun(cmdBase string) bool {
	for _, goFilePath := range m.goFilePaths {
		if filepath.Base(goFilePath) == cmdBase+goExt {
			return true
		}
	}
	return false
}

func newGoMainFileManager(dirPath string) common.Manager {
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
	return &GoMainFileManager{
		goFilePaths: matchedPaths,
	}
}

// --------

type GoMainPackageManager struct {
	mainDirPaths []string
}

var _ common.Manager = &GoMainPackageManager{}

func (m *GoMainPackageManager) CanRun(cmdBase string) bool {
	for _, mainDirPath := range m.mainDirPaths {
		if filepath.Base(mainDirPath) == cmdBase {
			return true
		}
	}
	return false
}

func (m *GoMainPackageManager) Run(args []string, shouldRebuild bool) (err error) {
	defer Catch(&err)
	cmdBase := filepath.Base(args[0])
	//goland:noinspection GoDeferInLoop
	for _, mainDirPath := range m.mainDirPaths {
		if filepath.Base(mainDirPath) != cmdBase {
			continue
		}
		exePath := V(ensureExeFile(mainDirPath, shouldRebuild))
		cmd := exec.Command(exePath, args[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	return errors.New(fmt.Sprintf("no matching go main directory found: %s", args[0]))
}

func (m *GoMainPackageManager) GetCommandBaseInfoList() (infoList []*common.CommandBaseInfo) {
	for _, mainDirPath := range m.mainDirPaths {
		infoList = append(infoList, &common.CommandBaseInfo{
			CmdBase:    filepath.Base(mainDirPath),
			SourcePath: mainDirPath,
		})
	}
	return infoList
}

func newGoMainPackageManager(dirPath string) common.Manager {
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
	return &GoMainPackageManager{
		mainDirPaths: goMainDirPaths,
	}
}

// --------

func init() {
	common.RegisterManagerFactory(
		"Go Main File Manager",
		newGoMainFileManager,
		100,
	)
	common.RegisterManagerFactory(
		"Go Main Package Manager",
		newGoMainPackageManager,
		100,
	)
}
