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

// cleanupOldBinaries removes old binaries from the cache directory occasionally.
func cleanupOldBinaries(dirPath string) (err error) {
	if rand.Intn(cleanupCycle) != 0 {
		return nil
	}
	dirEntries := V(os.ReadDir(dirPath))
	if err != nil {
		return err
	}
	for _, dirEntry := range dirEntries {
		if !dirEntry.IsDir() {
			continue
		}
		statInfoFile := V(os.Stat(filepath.Join(dirPath, dirEntry.Name(), common.InfoFileBase)))
		if statInfoFile.IsDir() {
			continue
		}
		if statInfoFile.ModTime().After(statInfoFile.ModTime().Add(-cleanupThresholdDays * 24 * time.Hour)) {
			continue
		}
		V0(os.RemoveAll(filepath.Join(dirPath, dirEntry.Name())))
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
		if V(os.Readlink(linkPath)) != appBase {
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
				if commandBaseInfo.CmdBase == appBase {
					continue
				}
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
				V0(os.Symlink(appBase, linkPath))
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
	//Debugger()
	shouldRebuild := os.Getenv("BUILD") != "" || os.Getenv("REBUILD") != ""
	if filepath.Base(args[0]) != appBase && // As symlink.
		// GoLand run configuration workaround
		!strings.HasSuffix(args[0], "_"+appBase) {
		commandArgs := args
		return execute(commandArgs, shouldRebuild)
	}
	if len(args[1:]) == 0 { // Without subcommand.
		return recreateLinks()
	}
	switch args[1] {
	case "exec", "execute":
		commandArgs := args[2:]
		return execute(commandArgs, shouldRebuild)
	case "install":
		return install(args[0])
	case "which":
		return which(args[2])
	}
	return errors.New(fmt.Sprintf("unknown command: %s", args[1]))
}
