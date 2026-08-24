package main

import "testing"

func TestParseKittyCellSize(t *testing.T) {
	size, ok := parseKittyCellSize([]byte("ignored\x1b[6;21;10ttrailing"))
	if !ok {
		t.Fatal("did not parse Kitty cell-size response")
	}
	if size.Width != 10 || size.Height != 21 {
		t.Fatalf("cell size = %+v", size)
	}
}

func TestParseKittyCellSizeRejectsInvalidResponse(t *testing.T) {
	if _, ok := parseKittyCellSize([]byte("\x1b[4;800;1200t")); ok {
		t.Fatal("accepted a window-size response as a cell-size response")
	}
}

func TestCellSizeFromWindow(t *testing.T) {
	size, ok := cellSizeFromWindow(40, 120, 1200, 800)
	if !ok {
		t.Fatal("did not calculate terminal cell size")
	}
	if size.Width != 10 || size.Height != 20 {
		t.Fatalf("cell size = %+v", size)
	}
}

func TestCellSizeFromWindowRejectsMissingPixels(t *testing.T) {
	if _, ok := cellSizeFromWindow(40, 120, 0, 0); ok {
		t.Fatal("accepted missing terminal pixel dimensions")
	}
}
