package common

import (
	"encoding/hex"
	. "github.com/knaka/go-utils"
	"hash"
	"os"
	"path/filepath"
	"sort"
)

type NewManagerFn func(dir string) Manager

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
	CreateLinks() (err error)
	CanRun(cmd string) bool
	Run(args []string) error
}

func CacheRootDir() (cacheRootDir string, err error) {
	defer Catch(&err)
	cacheRootDir = filepath.Join(Ensure(LinksDir()), ".cache")
	Ensure0(os.MkdirAll(cacheRootDir, 0755))
	return cacheRootDir, nil
}

func CacheDir(h hash.Hash) (dir string, err error) {
	defer Catch(&err)
	dir = filepath.Join(
		Ensure(CacheRootDir()),
		hex.EncodeToString(h.Sum(nil)),
	)
	Ensure0(os.MkdirAll(dir, 0755))
	return dir, nil
}

func CacheFile(h hash.Hash, base string) (cacheFile string, err error) {
	cacheFile = filepath.Join(
		Ensure(CacheDir(h)),
		base,
	)
	return
}

func InfoFile(h hash.Hash) (infoFile string, err error) {
	defer Catch(&err)
	infoFile = filepath.Join(
		Ensure(CacheDir(h)),
		"info.json",
	)
	return infoFile, err
}

var homeDir = Ensure(os.UserHomeDir())

// SetHomeDir should be available only for testing?
func SetHomeDir(dir string) {
	homeDir = dir
}

func LinksDir() (path string, err error) {
	defer Catch(&err)
	path = filepath.Join(homeDir, ".binc")
	Ensure0(os.MkdirAll(path, 0755))
	return
}
