package common

import (
	"encoding/hex"
	. "github.com/knaka/go-utils"
	"hash"
	"os"
	"path/filepath"
	"sort"
)

type NewManagerFn func(dirPath string) Manager

type Factory struct {
	PriorityWeight int
	NewManager     NewManagerFn
}

var factories []*Factory

// Factories returns a list of factories in descending order of priority weight.
func Factories() []*Factory {
	sort.Slice(factories, func(i, j int) bool {
		return factories[i].PriorityWeight > factories[j].PriorityWeight
	})
	return factories
}

func RegisterManagerFactory(fn NewManagerFn, priorityWeight int) {
	factories = append(factories, &Factory{
		PriorityWeight: priorityWeight,
		NewManager:     fn,
	})
}

type Manager interface {
	GetLinkBases() []string
	CanRun(cmdBase string) bool
	Run(args []string) error
}

func CacheRootDirPath() (cacheRootDirPath string, err error) {
	defer Catch(&err)
	cacheRootDirPath = filepath.Join(Ensure(LinksDirPath()), ".cache")
	Ensure0(os.MkdirAll(cacheRootDirPath, 0755))
	return cacheRootDirPath, nil
}

func CacheDirPath(h hash.Hash) (dir string, err error) {
	defer Catch(&err)
	dir = filepath.Join(
		Ensure(CacheRootDirPath()),
		hex.EncodeToString(h.Sum(nil)),
	)
	Ensure0(os.MkdirAll(dir, 0755))
	return dir, nil
}

func CachedExePath(h hash.Hash, base string) (cachedExePath string, err error) {
	cachedExePath = filepath.Join(
		Ensure(CacheDirPath(h)),
		base,
	)
	return
}

func InfoFilePath(h hash.Hash) (infoFile string, err error) {
	defer Catch(&err)
	infoFile = filepath.Join(
		Ensure(CacheDirPath(h)),
		"info.json",
	)
	return infoFile, err
}

var homeDirPath = Ensure(os.UserHomeDir())

// SetHomeDirPath should be available only for testing?
func SetHomeDirPath(dirPath string) {
	homeDirPath = dirPath
}

func LinksDirPath() (path string, err error) {
	defer Catch(&err)
	path = filepath.Join(homeDirPath, ".binc")
	Ensure0(os.MkdirAll(path, 0755))
	return
}
