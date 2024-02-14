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
			manager := newGoFileManager(tt.dir)
			if (manager != nil) != tt.found {
				t.Errorf("newGoFileManager() = %v, want %v", manager, tt.found)
			}
			if manager != nil {
				assert.Greater(t, len(manager.files), 0)
			}
		})
	}
}

func TestCanRun(t *testing.T) {
	manager := newGoFileManager(filepath.Join("testdata", "prj", "cmd"))
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
	common.SetHomeDir(homeDir)
	exe := Ensure(compileFile(filepath.Join("testdata", "prj", "cmd", "say_hello.go")))
	cmd := exec.Command(exe)
	assert.Contains(t, exe, "8a37f7eaaa05aeeff66bd6251305bac9173b2c4b")
	output := Ensure(cmd.Output())
	assert.Contains(t, string(output), "Hello, World!")
}
