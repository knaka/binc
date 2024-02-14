package common

import (
	. "github.com/knaka/go-utils"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

func TestLinkDir(t *testing.T) {
	assert.Equal(t, linksDir(), filepath.Join(Ensure(os.UserHomeDir()), ".binc"))
	homeDir := filepath.Join(t.TempDir(), "myhome")
	SetHomeDir(homeDir)
	assert.Equal(t, linksDir(), filepath.Join(homeDir, ".binc"))
}
