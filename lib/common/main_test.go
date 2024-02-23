package common

import (
	. "github.com/knaka/go-utils"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

func TestLinkDir(t *testing.T) {
	assert.Equal(t, V(LinksDirPath()), filepath.Join(V(os.UserHomeDir()), ".binc"))
	homeDir := filepath.Join(t.TempDir(), "myhome")
	SetHomeDirPath(homeDir)
	assert.Equal(t, V(LinksDirPath()), filepath.Join(homeDir, ".binc"))
}

func TestCamel2Kebab(t *testing.T) {
	type args struct {
		sIn string
	}
	tests := []struct {
		name  string
		args  args
		wantS string
	}{
		{"0", args{"Camel"}, "camel"},
		{"1", args{"CamelCase"}, "camel-case"},
		{"2", args{"camel"}, "camel"},
		{"3", args{"camel-case"}, "camel-case"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotS := Camel2Kebab(tt.args.sIn); gotS != tt.wantS {
				t.Errorf("Camel2Kebab() = %v, want %v", gotS, tt.wantS)
			}
		})
	}
}

func TestKebab2Camel(t *testing.T) {
	type args struct {
		sIn string
	}
	tests := []struct {
		name  string
		args  args
		wantS string
	}{
		{"0", args{"camel"}, "Camel"},
		{"1", args{"camel-case"}, "CamelCase"},
		{"2", args{"Camel"}, "Camel"},
		{"3", args{"CamelCase"}, "CamelCase"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotS := Kebab2Camel(tt.args.sIn); gotS != tt.wantS {
				t.Errorf("Kebab2Camel() = %v, want %v", gotS, tt.wantS)
			}
		})
	}
}
