package lib

import (
	"errors"
	"fmt"
	"github.com/h2non/filetype"
	"github.com/h2non/filetype/matchers"
	"github.com/h2non/filetype/types"
	"github.com/knaka/binc/lib/common"
	. "github.com/knaka/go-utils"
	"github.com/samber/lo"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	// Load all the language managers.
	_ "github.com/knaka/binc/lib/golang"
	_ "github.com/knaka/binc/lib/haskell"
)

const appBase = "binc"

func iterateOverManagers(
	fn func(manager common.Manager) error,
	lastError error,
) (err error) {
	defer Catch(&err)
	bincPathEnv := os.Getenv("BINCPATH")
	bincDirPaths := lo.Filter(strings.Split(bincPathEnv, ":"), func(dir string, _ int) bool {
		stat, err := os.Stat(dir)
		return err == nil && stat.IsDir()
	})
	for _, factory := range common.Factories() {
		for _, dirPath := range bincDirPaths {
			manager := factory.NewManager(dirPath)
			if manager == nil {
				continue
			}
			V0(fn(manager))
		}
	}
	return lastError
}

func recreateLinks() (err error) {
	return iterateOverManagers(
		func(manager common.Manager) error {
			return manager.CreateLinks()
		},
		nil,
	)
}

func execute(args []string) (err error) {
	return iterateOverManagers(
		func(manager common.Manager) error {
			if !manager.CanRun(filepath.Base(args[0])) {
				return nil
			}
			err := manager.Run(args)
			if err != nil {
				var exitError *exec.ExitError
				if errors.As(err, &exitError) {
					os.Exit(exitError.ExitCode())
				}
				return err
			}
			os.Exit(0)
			return nil // unreachable
		},
		errors.New(fmt.Sprintf("no matching command found: %s", args[0])),
	)
}

// install installs the given binary to the “links” directory.
func install(path string) (err error) {
	var binaryTypes = []types.Type{
		matchers.TypeElf,
		matchers.TypeMachO,
		matchers.TypeExe,
	}
	defer Catch(&err)
	absPath := V(filepath.Abs(path))
	stat := V(os.Stat(absPath))
	if stat.IsDir() {
		return errors.New("not a file")
	}
	fileType := (func() types.Type {
		in := V(os.Open(absPath))
		defer (func() { Ignore(in.Close()) })()
		return V(filetype.MatchReader(in))
	})()
	if !slices.Contains(binaryTypes, fileType) {
		return errors.New("not an executable binary")
	}
	tempDestPath := filepath.Join(V(common.LinksDirPath()), appBase+".temp")
	(func() {
		out := V(os.Create(tempDestPath))
		defer (func() { Ignore(out.Close()) })()
		in := V(os.Open(absPath))
		defer (func() { Ignore(in.Close()) })()
		_ = V(io.Copy(out, in))
	})()
	V0(os.Chmod(tempDestPath, 0755))
	destPath := filepath.Join(V(common.LinksDirPath()), appBase)
	Ignore(os.Rename(tempDestPath, destPath))
	V0(os.Rename(tempDestPath, destPath))
	return nil
}

func Main(args []string) (err error) {
	defer Catch(&err)
	Debugger()
	if filepath.Base(args[0]) != appBase &&
		// GoLand run configuration workaround
		!strings.HasSuffix(args[0], "_"+appBase) {
		commandArgs := args
		return execute(commandArgs)
	}
	if len(args[1:]) == 0 {
		return recreateLinks()
	}
	switch args[1] {
	case "exec", "execute":
		commandArgs := args[2:]
		return execute(commandArgs)
	case "install":
		return install(args[0])
	}
	return errors.New(fmt.Sprintf("unknown command: %s", args[1]))
}
