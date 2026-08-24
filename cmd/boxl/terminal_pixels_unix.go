//go:build unix

package main

import (
	"boxl"
	"os"

	"golang.org/x/sys/unix"
)

func ioctlKittyCellSize(tty *os.File) (boxl.KittyCellSize, bool) {
	size, err := unix.IoctlGetWinsize(int(tty.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return boxl.KittyCellSize{}, false
	}
	return cellSizeFromWindow(
		int(size.Row), int(size.Col), int(size.Xpixel), int(size.Ypixel),
	)
}
