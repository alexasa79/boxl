package main

import (
	"boxl"
	"io"
	"os"
	"regexp"
	"strconv"
	"time"

	"golang.org/x/term"
)

const kittyCellSizeQuery = "\x1b[16t"

var kittyCellSizeResponse = regexp.MustCompile("\\x1b\\[6;([0-9]+);([0-9]+)t")

func detectKittyCellSize(output io.Writer) (boxl.KittyCellSize, bool) {
	outputFile, ok := output.(*os.File)
	if !ok || !term.IsTerminal(int(outputFile.Fd())) {
		return boxl.KittyCellSize{}, false
	}
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return boxl.KittyCellSize{}, false
	}
	defer tty.Close()
	if size, ok := ioctlKittyCellSize(tty); ok {
		return size, true
	}
	return queryKittyCellSize(tty)
}

func queryKittyCellSize(tty *os.File) (boxl.KittyCellSize, bool) {
	state, err := term.MakeRaw(int(tty.Fd()))
	if err != nil {
		return boxl.KittyCellSize{}, false
	}
	defer term.Restore(int(tty.Fd()), state)
	if err := tty.SetReadDeadline(time.Now().Add(250 * time.Millisecond)); err != nil {
		return boxl.KittyCellSize{}, false
	}
	defer tty.SetReadDeadline(time.Time{})
	if _, err := io.WriteString(tty, kittyCellSizeQuery); err != nil {
		return boxl.KittyCellSize{}, false
	}

	response := make([]byte, 0, 64)
	buffer := make([]byte, 32)
	for len(response) < 256 {
		count, err := tty.Read(buffer)
		response = append(response, buffer[:count]...)
		if size, ok := parseKittyCellSize(response); ok {
			return size, true
		}
		if err != nil {
			return boxl.KittyCellSize{}, false
		}
	}
	return boxl.KittyCellSize{}, false
}

func cellSizeFromWindow(rows, columns, pixelWidth, pixelHeight int) (boxl.KittyCellSize, bool) {
	if rows <= 0 || columns <= 0 || pixelWidth <= 0 || pixelHeight <= 0 {
		return boxl.KittyCellSize{}, false
	}
	width := (pixelWidth + columns/2) / columns
	height := (pixelHeight + rows/2) / rows
	size := boxl.KittyCellSize{Width: width, Height: height}
	return size, width > 0 && height > 0
}

func parseKittyCellSize(response []byte) (boxl.KittyCellSize, bool) {
	match := kittyCellSizeResponse.FindSubmatch(response)
	if len(match) != 3 {
		return boxl.KittyCellSize{}, false
	}
	height, heightErr := strconv.Atoi(string(match[1]))
	width, widthErr := strconv.Atoi(string(match[2]))
	size := boxl.KittyCellSize{Width: width, Height: height}
	return size, heightErr == nil && widthErr == nil && width > 0 && height > 0
}
