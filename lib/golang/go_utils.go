package golang

import (
	"bytes"
	"encoding/json"
	"errors"
	. "github.com/knaka/go-utils"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// findGoModFile finds the go.mod file in the given directory or its parents.
func findGoModFile(initialDirPath string) (goModPath string, err error) {
	dirPath := initialDirPath
	for {
		goModPath = filepath.Join(dirPath, "go.mod")
		_, err = os.Stat(goModPath)
		if err == nil {
			return goModPath, nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		err = nil
		parentDirPath := filepath.Dir(dirPath)
		if parentDirPath == dirPath {
			return "", errors.New("go.mod not found")
		}
		dirPath = parentDirPath
	}
	// unreachable
}

// GoEnv is a struct to hold the output of `go env -json`.
type GoEnv struct {
	Version string `json:"GOVERSION"`
}

var goCmd = sync.OnceValues(func() (goPath string, err error) {
	defer Catch(&err)
	path := filepath.Join(os.Getenv("GOROOT"), "bin", "go")
	if stat, err := os.Stat(path); err == nil && !stat.IsDir() {
		return path, nil
	}
	goPath = Ensure(exec.LookPath("go"))
	if err == nil {
		return goPath, nil
	}
	path = filepath.Join(runtime.GOROOT(), "bin", "go")
	if stat, err := os.Stat(path); err == nil && !stat.IsDir() {
		return path, nil
	}
	return "", errors.New("go command not found")
})

// splitArgs splits the arguments into the arguments for `go run` and the arguments for the command.
func splitArgs(goCmd string, args []string) (runArgs []string, cmdArgs []string, err error) {
	var buf bytes.Buffer
	cmd := exec.Command(goCmd, append([]string{"run", "-n"}, args...)...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = &buf
	err = cmd.Run()
	if err != nil {
		_, _ = os.Stderr.Write(buf.Bytes())
		return
	}
	lines := strings.Split(buf.String(), "\n")
	lastLine := lines[len(lines)-2]
	fields := strings.SplitN(lastLine, " ", 2)
	runArgs = args
	if len(fields) > 1 {
		expandedCmdArgs := ""
		delim := ""
		for {
			elem := runArgs[len(runArgs)-1]
			runArgs = runArgs[:len(runArgs)-1]
			cmdArgs = append([]string{elem}, cmdArgs...)
			expandedCmdArgs = elem + delim + expandedCmdArgs
			delim = " "
			if expandedCmdArgs == fields[1] {
				break
			}
		}
	}
	return
}

var goEnv = sync.OnceValues(func() (goEnv_ GoEnv, err error) {
	defer Catch(&err)
	outStr := Ensure(exec.Command(Ensure(goCmd()), "env", "-json").Output())
	Ensure0(json.Unmarshal(outStr, &goEnv_))
	return
})
