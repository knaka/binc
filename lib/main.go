package lib

import (
	"errors"
	"github.com/h2non/filetype"
	"github.com/h2non/filetype/matchers"
	"github.com/h2non/filetype/types"
	"github.com/knaka/binc/lib/common"
	_ "github.com/knaka/binc/lib/golang"
	. "github.com/knaka/go-utils"
	"io"
	"os"
	"path/filepath"
	"slices"
)

func createLinks() (err error) {
	defer Catch(&err)

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
	input := R(os.Open(file))
	defer (func() { Ignore(input.Close()) })()
	R0(input.Read(header))
	kind := R(filetype.Match(header))
	if !slices.Contains(binaryTypes, kind) {
		return errors.New("not an executable binary")
	}
	binFile := filepath.Join(R(common.LinksDir()), "binc")
	binOut := R(os.Create(binFile))
	defer (func() { Ignore(binOut.Close()) })()
	binIn := R(os.Open(file))
	defer (func() { Ignore(binIn.Close()) })()
	R(io.Copy(binOut, binIn))
	R0(binOut.Close())
	R0(os.Chmod(binFile, 0755))
	return nil
}

func Main(args []string) (err error) {
	defer Catch(&err)
	WaitForDebugger()
	if filepath.Base(args[0]) == "binc" || filepath.Base(args[0]) == "main" {
		cmdArgs := args[1:]
		if len(cmdArgs) == 0 {
			return createLinks()
		}
		switch cmdArgs[0] {
		case "install":
			return ensureInstalled(args[0])
		}
		return errors.New("unknown command")
	}
	return nil
}
