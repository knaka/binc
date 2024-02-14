package golang

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type FileInfo struct {
	Name string `json:"name"`
	Hash string `json:"hash"`
	Size int64  `json:"-"`
}

func getGoFileInfoList(args []string) ([]*FileInfo, []string, error) {
	if len(args) == 0 {
		return nil, args, nil
	}
	var files []*FileInfo
	buildArgsWoTgt := args
	name := buildArgsWoTgt[len(buildArgsWoTgt)-1]
	buildArgsWoTgt = buildArgsWoTgt[:len(buildArgsWoTgt)-1]
	if stat, err := os.Stat(name); err == nil && stat.IsDir() {
		_, err := findGoModFile(filepath.Dir(name))
		if err == nil {
			return nil, args, nil
		}
		goFiles, err := os.ReadDir(name)
		if err != nil {
			return nil, args, nil
		}
		for _, goFile := range goFiles {
			if strings.HasSuffix(goFile.Name(), ".go") {
				p := filepath.Join(name, goFile.Name())
				fileInfo, err := getGoFileInfo(p)
				if err != nil {
					return nil, args, err
				}
				files = append(files, fileInfo)
			}
		}
	} else {
	outer:
		for {
			if !strings.HasSuffix(name, ".go") {
				break outer
			}
			fileInfo, err := getGoFileInfo(name)
			if err != nil {
				return nil, args, err
			}
			files = append(files, fileInfo)
			if len(buildArgsWoTgt) == 0 {
				break outer
			}
			name = buildArgsWoTgt[len(buildArgsWoTgt)-1]
			buildArgsWoTgt = buildArgsWoTgt[:len(buildArgsWoTgt)-1]
		}
	}
	sort.Slice(files, func(i, j int) bool {
		if files[i].Size == files[j].Size {
			return files[i].Hash < files[j].Hash
		}
		return files[i].Size < files[j].Size
	})
	return files, buildArgsWoTgt, nil
}

func getGoFileInfo(name string) (*FileInfo, error) {
	stat, err := os.Stat(name)
	if err != nil {
		return nil, err
	}
	if stat.IsDir() {
		return nil, errors.New("not a file")
	}
	hashStr, err := (func() (string, error) {
		hash_ := sha1.New()
		reader, err := os.Open(name)
		if err != nil {
			return "", err
		}
		defer (func() { _ = reader.Close() })()
		_, err = io.Copy(hash_, reader)
		if err != nil {
			return "", err
		}
		return hex.EncodeToString(hash_.Sum(nil)), nil
	})()
	if err != nil {
		return nil, err
	}
	return &FileInfo{
		Name: name,
		Hash: hashStr,
		Size: stat.Size(),
	}, nil
}
