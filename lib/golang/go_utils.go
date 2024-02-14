package golang

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// findGoModFile finds the go.mod file in the given directory or its parents.
func findGoModFile(modDir string) (string, error) {
	var err error
	for {
		_, err = os.Stat(filepath.Join(modDir, "go.mod"))
		if err == nil {
			break
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		err = nil
		parent := filepath.Dir(modDir)
		if parent == modDir {
			return "", errors.New("go.mod not found")
		}
		modDir = parent
	}
	return filepath.Join(modDir, "go.mod"), nil
}

// GoEnv is a struct to hold the output of `go env -json`.
type GoEnv struct {
	Version string `json:"GOVERSION"`
}

// findGoCmd finds the go command.
func findGoCmd() (string, error) {
	p := filepath.Join(os.Getenv("GOROOT"), "bin", "go")
	if stat, err := os.Stat(p); err == nil && !stat.IsDir() {
		return p, nil
	}
	goPath, err := exec.LookPath("go")
	if err == nil {
		return goPath, nil
	}
	p = filepath.Join(runtime.GOROOT(), "bin", "go")
	if stat, err := os.Stat(p); err == nil && !stat.IsDir() {
		return p, nil
	}
	return "", errors.New("go command not found")
}

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
	//if len(cmdArgs) > 0 && cmdArgs[0] == "--" {
	//	cmdArgs = cmdArgs[1:]
	//}
	return
}

func getGoEnv(goCmd string) (env GoEnv, err error) {
	cmd := exec.Command(goCmd, "env", "-json")
	out, err := cmd.Output()
	if err != nil {
		return
	}
	err = json.Unmarshal(out, &env)
	return
}
