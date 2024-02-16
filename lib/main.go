package lib

import (
	"errors"
	"github.com/h2non/filetype"
	"github.com/h2non/filetype/matchers"
	"github.com/h2non/filetype/types"
	"github.com/knaka/binc/lib/common"
	_ "github.com/knaka/binc/lib/golang"
	_ "github.com/knaka/binc/lib/haskell"
	. "github.com/knaka/go-utils"
	"github.com/samber/lo"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func createLinks() (err error) {
	defer Catch(&err)

	bincPath := os.Getenv("BINCPATH")
	bincDirs := lo.Filter(strings.Split(bincPath, ":"), func(dir string, _ int) bool {
		stat, err := os.Stat(dir)
		return err == nil && stat.IsDir()
	})

	for _, factory := range common.Factories() {
		for _, dir := range bincDirs {
			manager := factory.NewManager(dir)
			if manager == nil {
				continue
			}
			V0(manager.CreateLinks())
		}
	}

	return nil
}

var binaryTypes = []types.Type{
	matchers.TypeElf,
	matchers.TypeMachO,
	matchers.TypeExe,
}

func ensureInstalled(file string) (err error) {
	defer Catch(&err)
	stat, err := os.Stat(file)
	if err != nil {
		return err
	}
	if stat.IsDir() {
		return errors.New("not a file")
	}
	header := make([]byte, 128)
	input := V(os.Open(file))
	defer (func() { Ignore(input.Close()) })()
	V0(input.Read(header))
	kind := V(filetype.Match(header))
	if !slices.Contains(binaryTypes, kind) {
		return errors.New("not an executable binary")
	}
	binFile := filepath.Join(V(common.LinksDir()), "binc")
	binOut := V(os.Create(binFile))
	defer (func() { Ignore(binOut.Close()) })()
	binIn := V(os.Open(file))
	defer (func() { Ignore(binIn.Close()) })()
	_ = V(io.Copy(binOut, binIn))
	V0(binOut.Close())
	V0(os.Chmod(binFile, 0755))
	return nil
}

func exec(args []string) (err error) {
	defer Catch(&err)
	bincPath := os.Getenv("BINCPATH")
	bincDirs := strings.Split(bincPath, ":")
	for _, factory := range common.Factories() {
		for _, dir := range bincDirs {
			manager := factory.NewManager(dir)
			if manager == nil {
				continue
			}
			if manager.CanRun(filepath.Base(args[0])) {
				V0(manager.Run(args))
				return nil
			}
		}
	}
	return errors.New("Cannot run " + args[0])
}

const appName = "binc"

func Main(args []string) (err error) {
	defer Catch(&err)
	Debugger()
	if filepath.Base(args[0]) != appName &&
		// GoLand run configuration workaround
		!strings.HasSuffix(args[0], "_"+appName) {
		return exec(args)
	}
	if len(args[1:]) == 0 {
		return createLinks()
	}
	switch args[1] {
	case "exec":
		return exec(args[2:])
	case "install":
		return ensureInstalled(args[0])
	}
	return errors.New("unknown command")
}
