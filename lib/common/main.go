package common

import (
	"encoding/hex"
	. "github.com/knaka/go-utils"
	"hash"
	"os"
	"path/filepath"
)

type NewManagerFn func(dir string) Manager

type Factory struct {
	PriorityWeight int
	NewManager     NewManagerFn
}

var factories []*Factory

func RegisterManagerFactory(fn NewManagerFn, priorityWeight int) {
	factories = append(factories, &Factory{
		PriorityWeight: priorityWeight,
		NewManager:     fn,
	})
}

type Manager interface {
	//Link() ([]string, error)
	CanRun(cmd string) bool
	Run(args []string) error
}

func CacheRootDir() string {
	cacheRootDir := filepath.Join(linksDir(), ".cache")
	Ensure0(os.MkdirAll(cacheRootDir, 0755))
	return cacheRootDir
}

func CacheDir(h hash.Hash) (dir string) {
	dir = filepath.Join(
		CacheRootDir(),
		hex.EncodeToString(h.Sum(nil)),
	)
	Ensure0(os.MkdirAll(dir, 0755))
	return dir
}

func CacheFile(h hash.Hash) string {
	return filepath.Join(
		CacheDir(h),
		"exe",
	)
}

func InfoFile(h hash.Hash) string {
	return filepath.Join(
		CacheDir(h),
		"info.json",
	)
}

var homeDir = Ensure(os.UserHomeDir())

// SetHomeDir should be available only for testing?
func SetHomeDir(dir string) {
	homeDir = dir
}

func linksDir() string {
	path := filepath.Join(homeDir, ".binc")
	Ensure0(os.MkdirAll(path, 0755))
	return path
}
