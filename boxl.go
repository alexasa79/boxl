// Package boxl parses and renders the box language as PNG diagrams.
package boxl

// RenderPNG renders source into a PNG suitable for graphical frontends. Empty
// source produces an empty byte slice.
func RenderPNG(source string) ([]byte, error) {
	return renderPNGSource(source)
}
