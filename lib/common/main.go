package common

import (
	. "github.com/knaka/go-utils"
	"hash"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

type NewManagerFn func(dirPath string) Manager

type Factory struct {
	Name           string
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

func RegisterManagerFactory(name string, fn NewManagerFn, priorityWeight int) {
	factories = append(factories, &Factory{
		Name:           name,
		PriorityWeight: priorityWeight,
		NewManager:     fn,
	})
}

type CommandBaseInfo struct {
	CmdBase    string
	SourcePath string
}

type Manager interface {
	GetCommandBaseInfoList() []*CommandBaseInfo
	CanRun(cmdBase string) bool
	Run(args []string, shouldRebuild bool) error
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
		hashStr(h),
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
		".info.json",
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

var reEachCamel = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`([A-Z][a-z0-9]*)`)
})

func Camel2Kebab(sIn string) (s string) {
	s = sIn
	s = reEachCamel().ReplaceAllStringFunc(s, func(s string) string {
		return "-" + strings.ToLower(s)
	})
	s = strings.TrimPrefix(s, "-")
	return s
}

var reEachKebab = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`-([a-z0-9])`)
})

func Kebab2Camel(sIn string) (s string) {
	s = "-" + sIn
	s = reEachKebab().ReplaceAllStringFunc(s, func(s string) string {
		return strings.ToUpper(s[1:2]) + s[2:]
	})
	s = strings.TrimPrefix(s, "-")
	return s

}
