package boxl

import (
	"encoding/base64"
	"fmt"
	"io"
)

const kittyChunkSize = 4096

// KittyCellSize is the terminal cell size in physical pixels.
type KittyCellSize struct {
	Width  int
	Height int
}

func (s KittyCellSize) valid() bool {
	return s.Width > 0 && s.Height > 0
}

// RenderPNG renders source into a PNG suitable for graphical frontends. Empty
// source produces an empty byte slice.
func RenderPNG(source string) ([]byte, error) {
	return renderPNGSource(source)
}

// TransformKitty reads box language source and writes a Kitty graphics
// protocol sequence that transmits and displays the rendered diagram.
func TransformKitty(input io.Reader, output io.Writer) error {
	return transformKitty(input, output, KittyCellSize{})
}

// TransformKittyAtCellSize renders at the terminal's physical cell resolution.
// The resulting PNG exactly matches the pixel dimensions of its Kitty placement.
func TransformKittyAtCellSize(input io.Reader, output io.Writer, cellSize KittyCellSize) error {
	if !cellSize.valid() {
		return fmt.Errorf("Kitty cell width and height must be positive")
	}
	return transformKitty(input, output, cellSize)
}

func transformKitty(input io.Reader, output io.Writer, cellSize KittyCellSize) error {
	source, err := io.ReadAll(input)
	if err != nil {
		return fmt.Errorf("read box language: %w", err)
	}
	pngData, columns, rows, err := renderKittyPNGSource(string(source), cellSize)
	if err != nil {
		return err
	}
	if len(pngData) == 0 {
		return nil
	}
	if err := writeKittyImage(output, pngData, columns, rows); err != nil {
		return fmt.Errorf("write Kitty image: %w", err)
	}
	return nil
}

func writeKittyImage(output io.Writer, pngData []byte, columns, rows int) error {
	encoded := base64.StdEncoding.EncodeToString(pngData)
	for offset := 0; offset < len(encoded); offset += kittyChunkSize {
		end := min(offset+kittyChunkSize, len(encoded))
		more := 0
		if end < len(encoded) {
			more = 1
		}
		var control string
		if offset == 0 {
			control = fmt.Sprintf("a=T,f=100,q=2,m=%d", more)
			if columns > 0 && rows > 0 {
				control = fmt.Sprintf("a=T,f=100,q=2,c=%d,r=%d,m=%d", columns, rows, more)
			}
		} else {
			control = fmt.Sprintf("q=2,m=%d", more)
		}
		if _, err := fmt.Fprintf(output, "\x1b_G%s;%s\x1b\\", control, encoded[offset:end]); err != nil {
			return err
		}
	}
	return nil
}
