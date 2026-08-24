// Package boxl parses and renders the box language as deterministic ASCII art.
package boxl

import (
	"fmt"
	"io"
)

// Render parses source and renders it with the canonical options.
func Render(source string) (string, error) {
	diagram, err := Parse(source)
	if err != nil {
		return "", err
	}
	renderer, err := NewRenderer(DefaultRenderOptions())
	if err != nil {
		return "", err
	}
	return renderer.Render(diagram)
}

// Transform reads box language source and writes its rendered form. A non-empty
// diagram is terminated by a newline for command-line friendliness.
func Transform(input io.Reader, output io.Writer) error {
	source, err := io.ReadAll(input)
	if err != nil {
		return fmt.Errorf("read box language: %w", err)
	}
	rendered, err := Render(string(source))
	if err != nil {
		return err
	}
	if rendered == "" {
		return nil
	}
	if _, err := fmt.Fprintln(output, rendered); err != nil {
		return fmt.Errorf("write rendered diagram: %w", err)
	}
	return nil
}
