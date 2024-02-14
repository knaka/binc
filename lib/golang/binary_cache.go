package golang

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"github.com/samber/lo"
	"hash"
	"log"
	"os"
	"path/filepath"
)

type BuildInfo struct {
	Version string    `json:"version"`
	Args    []string  `json:"build_args"`
	HashStr string    `json:"hash"`
	Hash    hash.Hash `json:"-"`
}

func newBuildInfoWithPkg(goVersion string, buildArgsWoTgt []string, pkg *string) BuildInfo {
	return newBuildInfo(goVersion, buildArgsWoTgt, pkg, nil)
}

func newBuildInfoWithFiles(goVersion string, buildArgsWoTgt []string, files []*FileInfo) BuildInfo {
	return newBuildInfo(goVersion, buildArgsWoTgt, nil, files)
}

func newBuildInfo(goVersion string, buildArgsWoTgt []string, pkg *string, files []*FileInfo) BuildInfo {
	hashAccu := sha1.New()
	hashAccu.Write([]byte(goVersion))
	for _, arg := range buildArgsWoTgt {
		hashAccu.Write([]byte(arg))
	}
	if pkg != nil {
		hashAccu.Write([]byte(*pkg))
	}
	for _, f := range files {
		hashAccu.Write([]byte(f.Hash))
	}
	// Deep copy the array
	log.Println("9c0c811", buildArgsWoTgt)
	args := make([]string, len(buildArgsWoTgt))
	copy(args, buildArgsWoTgt)
	if pkg != nil {
		args = append(args, *pkg)
	}
	args = append(args, lo.Map(files, func(f *FileInfo, _ int) string {
		return f.Name + ":" + f.Hash
	})...)
	return BuildInfo{
		Version: goVersion,
		Args:    args,
		Hash:    hashAccu,
		HashStr: hex.EncodeToString(hashAccu.Sum(nil)),
	}
}

func putBuildInfo(exeCacheDir string, buildInfo_ *BuildInfo) error {
	var err error
	infoPath := filepath.Join(exeCacheDir, "build_info.json")
	buildInfoJson, err := json.Marshal(buildInfo_)
	if err != nil {
		return err
	}
	err = os.WriteFile(infoPath, buildInfoJson, 0644)
	if err != nil {
		return err
	}
	return nil
}

//func cleanupOldBinaries(dir string) error {
//	subdirs, err := os.ReadDir(dir)
//	if err != nil {
//		return err
//	}
//	for _, subdir := range subdirs {
//		if !subdir.IsDir() {
//			continue
//		}
//		if stat, err := os.Stat(filepath.Join(dir, subdir.Name(), "main")); err == nil && !stat.IsDir() {
//			if stat.ModTime().Before(stat.ModTime().Add(-cleanupThresholdDays * 24 * time.Hour)) {
//				err = os.RemoveAll(filepath.Join(dir, subdir.Name()))
//				if err != nil {
//					return err
//				}
//			}
//		}
//	}
//	return nil
//}
