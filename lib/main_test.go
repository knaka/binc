package lib

import (
	"github.com/knaka/binc/lib/mock"
	testfsutils "github.com/knaka/go-testutils/fs"
	. "github.com/knaka/go-utils"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func AlwaysZero(_ int) int {
	return 0
}

var _ randIntNFnT = AlwaysZero

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
	assert.Len(t, V(os.ReadDir(cacheRootDirPath)), 2)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	myDep := mock.NewMockMyDep(ctrl)
	myDep.EXPECT().RandIntN(gomock.Eq(cleanupCycle)).
		Times(99).Return(1)
	myDep.EXPECT().RandIntN(gomock.Eq(cleanupCycle)).
		Times(1).Return(0)
	for i := 0; i < cleanupCycle; i++ {
		V0(cleanupOldBinaries(cacheRootDirPath, withRandFn(myDep.RandIntN)))
	}
	dirEntries := V(os.ReadDir(cacheRootDirPath))
	assert.Len(t, dirEntries, 1)
	assert.Equal(t, "9b19d37", dirEntries[0].Name())
}
