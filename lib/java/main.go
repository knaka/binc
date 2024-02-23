package java

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

type JavaClassManager struct {
	javacCmd  string
	javaCmd   string
	filePaths []string
}

var _ common.Manager = &JavaClassManager{}

var extensions = []string{
	".java",
}

func (m *JavaClassManager) GetCommandBaseInfoList() (infoList []*common.CommandBaseInfo) {
	for _, javaFilePath := range m.filePaths {
		javaFileBase := filepath.Base(javaFilePath)
		for _, ext := range extensions {
			if strings.HasSuffix(javaFileBase, ext) {
				infoList = append(infoList, &common.CommandBaseInfo{
					CmdBase:    common.Camel2Kebab(javaFileBase[:len(javaFileBase)-len(ext)]),
					SourcePath: javaFilePath,
				})
			}
		}
	}
	return infoList
}

func (m *JavaClassManager) CanRun(cmdBase string) bool {
	for _, hsFilePath := range m.filePaths {
		for _, ext := range extensions {
			if filepath.Base(hsFilePath) == common.Kebab2Camel(cmdBase)+ext {
				return true
			}
		}
	}
	return false
}

func ensureClassFile(javaFilePath string, cmdBase string, shouldRebuild bool) (classFilePath string, err error) {
	var fileInfoList []*common.FileInfo
	fileInfoList = append(fileInfoList, V(common.GetFileInfo(javaFilePath)))
	buildInfo := common.NewBuildInfo(
		"",  // TODO: Decide if the version of Javac or Java should be recorded
		nil, // Any arguments?
		fileInfoList,
	)
	classFilePath = V(common.CachedExePath(buildInfo.Hash, common.Kebab2Camel(cmdBase)+".class"))
	if _, err = os.Stat(classFilePath); err != nil || shouldRebuild {
		V0(os.MkdirAll(filepath.Dir(classFilePath), 0755))
		cmd := exec.Command(V(javacCommand()), "-d", filepath.Dir(classFilePath), javaFilePath)
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		V0(cmd.Run())
		buildInfoJson := V(json.Marshal(buildInfo))
		V0(os.WriteFile(V(common.InfoFilePath(buildInfo.Hash)), buildInfoJson, 0644))
		log.Println("built:", classFilePath)
	}
	return classFilePath, nil
}

func (m *JavaClassManager) Run(args []string, shouldRebuild bool) (err error) {
	defer Catch(&err)
	cmdBase := filepath.Base(args[0])
	for _, javaFilePath := range m.filePaths {
		for _, ext := range extensions {
			if filepath.Base(javaFilePath) == common.Kebab2Camel(cmdBase)+ext {
				classFilePath := V(ensureClassFile(javaFilePath, cmdBase, shouldRebuild))
				classDir := filepath.Dir(classFilePath)
				classBase := common.Kebab2Camel(cmdBase)
				cmd := exec.Command(m.javaCmd, append([]string{"-cp", classDir, classBase}, args[1:]...)...)
				cmd.Stdin = os.Stdin
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				return cmd.Run()
			}
		}
	}
	return errors.New(fmt.Sprintf("no matching java file found: %s", args[0]))
}

var javacCommand = sync.OnceValues(func() (cabalPath string, err error) {
	defer Catch(&err)
	return V(exec.LookPath("javac")), nil
})

var javaCommand = sync.OnceValues(func() (cabalPath string, err error) {
	defer Catch(&err)
	return V(exec.LookPath("java")), nil
})

func newJavaClassManager(dirPath string) common.Manager {
	javacCmd, err := javacCommand()
	if err != nil {
		return nil
	}
	javaCmd, err := javaCommand()
	if err != nil {
		return nil
	}
	var matchedPaths []string
	for _, ext := range extensions {
		matchedPaths = append(matchedPaths, V(filepath.Glob(filepath.Join(dirPath, "*"+ext)))...)
	}
	if len(matchedPaths) == 0 {
		return nil
	}
	return &JavaClassManager{
		javacCmd:  javacCmd,
		javaCmd:   javaCmd,
		filePaths: matchedPaths,
	}
}

func init() {
	common.RegisterManagerFactory(
		"Java Class Manager",
		newJavaClassManager,
		50,
	)
}
