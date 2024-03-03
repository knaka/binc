package lib

import (
	testfsutils "github.com/knaka/go-testutils/fs"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/knaka/go-utils"
)

func AlwaysZero(_ int) int {
	return 0
}

var _ intRandFnT = AlwaysZero

// One of two cache entries is older than the threshold, so it should be removed.
func TestCleanupOldBinaries(t *testing.T) {
	cacheRootDirPath := filepath.Join(t.TempDir(), "cache")
	V0(testfsutils.CopyDir(
		cacheRootDirPath,
		filepath.Join("testdata", "cache"),
	))
	oldCacheDirPath := filepath.Join(cacheRootDirPath, "3f7e097")
	V0(os.Chtimes(
		filepath.Join(oldCacheDirPath, ".info.json"),
		time.Time{},
		time.Now().AddDate(0, 0, -cleanupThresholdDays-1),
	))
	newCacheDirPath := filepath.Join(cacheRootDirPath, "9b19d37")
	V0(os.Chtimes(
		filepath.Join(newCacheDirPath, ".info.json"),
		time.Time{},
		time.Now().AddDate(0, 0, -cleanupThresholdDays+1),
	))
	// Pass AlwaysZero to make the test deterministic.
	V0(cleanupOldBinaries(cacheRootDirPath, randFn(AlwaysZero)))
	dirEntries := V(os.ReadDir(cacheRootDirPath))
	assert.Len(t, dirEntries, 1)
	assert.Equal(t, "9b19d37", dirEntries[0].Name())
}
