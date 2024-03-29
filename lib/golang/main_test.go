package golang

import (
	"github.com/knaka/binc/lib/common"
	. "github.com/knaka/go-utils"
	"github.com/stretchr/testify/assert"
	"os/exec"
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
			manager := newGoMainFileManager(tt.dir)
			if (manager != nil) != tt.found {
				t.Errorf("newGoMainFileManager() = %v, want %v", manager, tt.found)
			}
			if manager != nil {
				goManager := manager.(*GoMainFileManager)
				assert.Greater(t, len(goManager.goFilePaths), 0)
			}
		})
	}
}

func TestCanRun(t *testing.T) {
	manager := newGoMainFileManager(filepath.Join("testdata", "prj", "cmd"))
	type args struct {
		cmd string
	}
	tests := []struct {
		name string
		cmd  string
		want bool
	}{
		{
			"can run",
			"say_hello",
			true,
		},
		{
			"cannot run",
			"saygoodbye",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if manager != nil {
				if got := manager.CanRun(tt.cmd); got != tt.want {
					t.Errorf("CanRun() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestCompile(t *testing.T) {
	homeDir := filepath.Join(t.TempDir(), "home")
	common.SetHomeDirPath(homeDir)
	exe := V(ensureExeFile(filepath.Join("testdata", "prj", "cmd", "say_hello.go"), false))
	cmd := exec.Command(exe)
	assert.Contains(t, exe, "f8391da")
	output := V(cmd.Output())
	assert.Contains(t, string(output), "Hello, World!")
}

func TestRun(t *testing.T) {
	homeDir := filepath.Join(t.TempDir(), "home")
	common.SetHomeDirPath(homeDir)
	manager := newGoMainFileManager(filepath.Join("testdata", "prj", "cmd"))
	V0(manager.Run([]string{filepath.Join("foo", "say_hello"), "foo", "bar"}, false))
}
