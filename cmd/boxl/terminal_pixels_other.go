//go:build !unix

package main

import (
	"boxl"
	"os"
)

func ioctlKittyCellSize(tty *os.File) (boxl.KittyCellSize, bool) {
	return boxl.KittyCellSize{}, false
}
