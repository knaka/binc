package golang

import (
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

func TestNewManager(t *testing.T) {
	type args struct {
		dir string
	}
	tests := []struct {
		name  string
		dir   string
		found bool
	}{
		{
			"dir not exists",
			filepath.Join("testdata", "prjfoo", "cmd"),
			false,
		},
		{
			"dir exists",
			filepath.Join("testdata", "prj", "cmd"),
			true,
		},
		{
			"empty project",
			filepath.Join("testdata", "prjempty", "cmd"),
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := newFileManager(tt.dir)
			if (manager != nil) != tt.found {
				t.Errorf("newFileManager() = %v, want %v", manager, tt.found)
			}
			if manager != nil {
				goManager := manager.(*GoFileManager)
				assert.Greater(t, len(goManager.files), 0)
			}
		})
	}
}
