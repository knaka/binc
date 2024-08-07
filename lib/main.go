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
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	// Load all the language managers.
	_ "github.com/knaka/binc/lib/golang"
	_ "github.com/knaka/binc/lib/haskell"
	_ "github.com/knaka/binc/lib/java"
	_ "github.com/knaka/binc/lib/rust"
	_ "github.com/knaka/binc/lib/scala"
)

//go:generate -command mockgen go run github.com/knaka/go-run-cache@latest go.uber.org/mock/mockgen@latest

//go:generate mockgen -package=mock -destination mock/main.go . MyDep
type MyDep interface {
	RandIntN(int) int
}

const appBase = "binc"

var copiedBase = fmt.Sprintf(".%s", appBase)

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
			if stat, err := os.Stat(dirPath); err != nil || !stat.IsDir() {
				continue
			}
			manager := factory.NewManager(dirPath)
			if manager == nil {
				continue
			}
			//V0(fn(manager))
			err = fn(manager)
			if err != nil {
				return err
			}
		}
	}
	return lastError
}

func which(cmdBase string) (err error) {
	return iterateOverManagers(func(manager common.Manager) (err error) {
		for _, commandBaseInfo := range manager.GetCommandBaseInfoList() {
			if commandBaseInfo.CmdBase == cmdBase {
				fmt.Println(commandBaseInfo.SourcePath)
				return nil
			}
		}
		return nil
	}, nil)
}

// Average number of launches between cleanups
const cleanupCycle = 100

// Number of days after which a binary is considered old
const cleanupThresholdDays = 90

type randIntNFnT func(int) int

type optionsT struct {
	randIntNFn randIntNFnT
}

type optSetterFnT func(*optionsT)

//goland:noinspection GoExportedFuncWithUnexportedType
func withRandFn(fn randIntNFnT) optSetterFnT {
	return func(opts *optionsT) {
		opts.randIntNFn = fn
	}
}

// cleanupOldBinaries removes old binaries from the cache directory occasionally.
func cleanupOldBinaries(
	cacheRootDirPath string,
	optSetterFnS ...optSetterFnT,
) (err error) {
	defer Catch(&err)
	options := optionsT{
		randIntNFn: rand.Intn,
	}
	for _, optSetterFn := range optSetterFnS {
		optSetterFn(&options)
	}
	if options.randIntNFn(cleanupCycle) != 0 {
		return nil
	}
	dirEntries := V(os.ReadDir(cacheRootDirPath))
	if err != nil {
		return err
	}
	for _, dirEntry := range dirEntries {
		if !dirEntry.IsDir() {
			continue
		}
		statInfoFile := V(os.Stat(filepath.Join(cacheRootDirPath, dirEntry.Name(), common.InfoFileBase)))
		if statInfoFile.IsDir() {
			continue
		}
		if statInfoFile.ModTime().After(time.Now().Add(-cleanupThresholdDays * 24 * time.Hour)) {
			continue
		}
		V0(os.RemoveAll(filepath.Join(cacheRootDirPath, dirEntry.Name())))
	}
	return nil
}

func recreateLinks() (err error) {
	defer Catch(&err)
	// Clean up old binaries.
	V0(cleanupOldBinaries(V(common.CacheRootDirPath())))
	linksDirPath := V(common.LinksDirPath())
	// Remove all symlinks in the “links” directory.
	for _, dirEntry := range V(os.ReadDir(linksDirPath)) {
		if dirEntry.Type() != os.ModeSymlink {
			continue
		}
		linkPath := filepath.Join(linksDirPath, dirEntry.Name())
		if V(os.Readlink(linkPath)) != copiedBase {
			continue
		}
		V0(os.Remove(linkPath))
	}
	theMap := map[string]string{}
	// Then create links.
	return iterateOverManagers(
		func(manager common.Manager) (err error) {
			defer Catch(&err)
			for _, commandBaseInfo := range manager.GetCommandBaseInfoList() {
				linkPath := filepath.Join(linksDirPath, commandBaseInfo.CmdBase)
				if _, err := os.Lstat(linkPath); err == nil {
					println("Conflicting source:", commandBaseInfo.SourcePath)
					println("Previous source that will be active:", theMap[commandBaseInfo.CmdBase])
					continue
				}
				theMap[commandBaseInfo.CmdBase] = commandBaseInfo.SourcePath
				if err != nil && !os.IsNotExist(err) {
					return err
				}
				V0(os.Symlink(copiedBase, linkPath))
			}
			return nil
		},
		nil,
	)
}

func execute(args []string, shouldRebuild bool) (err error) {
	return iterateOverManagers(
		func(manager common.Manager) (err error) {
			defer Catch(&err)
			if !manager.CanRun(filepath.Base(args[0])) {
				return nil
			}
			err = manager.Run(args, shouldRebuild)
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
func installSelfToLinksDir() (err error) {
	cmdPath := V(filepath.EvalSymlinks(V(filepath.Abs(V(os.Executable())))))
	destPath := filepath.Join(V(common.LinksDirPath()), copiedBase)
	stat := V(os.Stat(cmdPath))
	if stat.IsDir() {
		return errors.New("not a file")
	}
	fileType := (func() types.Type {
		in := V(os.Open(cmdPath))
		defer (func() { Ignore(in.Close()) })()
		return V(filetype.MatchReader(in))
	})()
	if !slices.Contains([]types.Type{
		matchers.TypeElf,
		matchers.TypeMachO,
		matchers.TypeExe,
	}, fileType) {
		return errors.New("not an executable binary")
	}
	(func() {
		out := V(os.Create(destPath))
		defer (func() { Ignore(out.Close()) })()
		in := V(os.Open(cmdPath))
		defer (func() { Ignore(in.Close()) })()
		V0(io.Copy(out, in))
	})()
	V0(os.Chmod(destPath, 0755))
	return nil
}

func Main(args []string) (err error) {
	Debugger()
	shouldRebuild := os.Getenv("BUILD") != "" || os.Getenv("REBUILD") != ""
	// Run the target command.
	if !slices.Contains([]string{appBase, copiedBase}, filepath.Base(args[0])) &&
		// GoLand “run configuration” workaround
		!strings.HasSuffix(args[0], "_"+appBase) {
		return execute(args, shouldRebuild)
	}
	// Run binc command itself.
	// No subcommand specified.
	if len(args[1:]) == 0 {
		return recreateLinks()
	}
	switch args[1] {
	case "exec", "execute":
		commandArgs := args[2:]
		return execute(commandArgs, shouldRebuild)
	case "install":
		err = installSelfToLinksDir()
		if err != nil {
			return
		}
		return recreateLinks()
	case "which":
		return which(args[2])
	}
	return errors.New(fmt.Sprintf("unknown command: %s", args[1]))
}
