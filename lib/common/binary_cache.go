package common

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"github.com/samber/lo"
	"hash"
	"io"
	"os"
	"sort"
)

const hashNumDig = 7

type BuildInfo struct {
	Version string    `json:"version"`
	Args    []string  `json:"args"`
	Files   []string  `json:"files"`
	Hash    hash.Hash `json:"-"`
	HashStr string    `json:"hash"`
}

type FileInfo struct {
	Name    string
	Hash    hash.Hash
	HashStr string
	Size    int64
}

func GetFileInfo(filePath string) (fileInfo *FileInfo, err error) {
	stat, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}
	if stat.IsDir() {
		return nil, errors.New("not a file")
	}
	hash_, err := (func() (hash.Hash, error) {
		hash_ := sha1.New()
		reader, err := os.Open(filePath)
		if err != nil {
			return nil, err
		}
		defer (func() { _ = reader.Close() })()
		_, err = io.Copy(hash_, reader)
		if err != nil {
			return nil, err
		}
		return hash_, nil
	})()
	if err != nil {
		return nil, err
	}
	return &FileInfo{
		Name:    filePath,
		Hash:    hash_,
		HashStr: hex.EncodeToString(hash_.Sum(nil))[:hashNumDig],
		Size:    stat.Size(),
	}, nil
}

func NewBuildInfo(
	version string,
	args []string,
	fileInfoList []*FileInfo,
) *BuildInfo {
	hashOut := sha1.New()
	hashOut.Write([]byte(version))
	for _, arg := range args {
		hashOut.Write([]byte(arg))
	}
	sort.Slice(fileInfoList, func(i, j int) bool {
		if fileInfoList[i].Size == fileInfoList[j].Size {
			return fileInfoList[i].HashStr < fileInfoList[j].HashStr
		}
		return fileInfoList[i].Size < fileInfoList[j].Size
	})
	for _, fileInfo := range fileInfoList {
		hashOut.Write([]byte(fileInfo.HashStr))
	}
	return &BuildInfo{
		Version: version,
		Args:    args,
		Files: lo.Map(fileInfoList, func(f *FileInfo, _ int) string {
			return f.Name + ":" + f.HashStr
		}),
		Hash:    hashOut,
		HashStr: hex.EncodeToString(hashOut.Sum(nil))[0:hashNumDig],
	}
}
