package lib

import (
	_ "github.com/knaka/binc/lib/golang"
	. "github.com/knaka/go-utils"
)

func Main(args []string) (err error) {
	defer Catch(&err)
	return nil
}
