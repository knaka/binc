package golang

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGoCmd(t *testing.T) {
	path, err := goCmd()
	assert.Nil(t, err)
	assert.NotEmpty(t, path)
}
