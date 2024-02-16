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

func getGoFileInfoList(buildArgs []string) ([]*FileInfo, []string, error) {
	if len(buildArgs) == 0 {
		return nil, buildArgs, nil
	}
	var fileInfoList []*FileInfo
	buildArgsWoTgt := buildArgs
	tgt := buildArgsWoTgt[len(buildArgsWoTgt)-1]
	buildArgsWoTgt = buildArgsWoTgt[:len(buildArgsWoTgt)-1]
	if stat, err := os.Stat(tgt); err == nil && stat.IsDir() {
		_, err := findGoModFile(filepath.Dir(tgt))
		if err == nil {
			return nil, buildArgs, nil
		}
		goFiles, err := os.ReadDir(tgt)
		if err != nil {
			return nil, buildArgs, nil
		}
		for _, goFile := range goFiles {
			if strings.HasSuffix(goFile.Name(), ".go") {
				p := filepath.Join(tgt, goFile.Name())
				fileInfo, err := getGoFileInfo(p)
				if err != nil {
					return nil, buildArgs, err
				}
				fileInfoList = append(fileInfoList, fileInfo)
			}
		}
	} else {
	outer:
		for {
			if !strings.HasSuffix(tgt, ".go") {
				break outer
			}
			fileInfo, err := getGoFileInfo(tgt)
			if err != nil {
				return nil, buildArgs, err
			}
			fileInfoList = append(fileInfoList, fileInfo)
			if len(buildArgsWoTgt) == 0 {
				break outer
			}
			tgt = buildArgsWoTgt[len(buildArgsWoTgt)-1]
			buildArgsWoTgt = buildArgsWoTgt[:len(buildArgsWoTgt)-1]
		}
	}
	sort.Slice(fileInfoList, func(i, j int) bool {
		if fileInfoList[i].Size == fileInfoList[j].Size {
			return fileInfoList[i].Hash < fileInfoList[j].Hash
		}
		return fileInfoList[i].Size < fileInfoList[j].Size
	})
	return fileInfoList, buildArgsWoTgt, nil
}

func getGoFileInfo(goFilePath string) (*FileInfo, error) {
	stat, err := os.Stat(goFilePath)
	if err != nil {
		return nil, err
	}
	if stat.IsDir() {
		return nil, errors.New("not a file")
	}
	hashStr, err := (func() (string, error) {
		hash_ := sha1.New()
		reader, err := os.Open(goFilePath)
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
		Name: goFilePath,
		Hash: hashStr,
		Size: stat.Size(),
	}, nil
}
