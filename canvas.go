package boxl

import "strings"

type canvas struct {
	width  int
	height int
	cells  [][]rune
}

func newCanvas(width, height int) *canvas {
	c := &canvas{width: width, height: height, cells: make([][]rune, height)}
	for y := range c.cells {
		c.cells[y] = make([]rune, width)
		for x := range c.cells[y] {
			c.cells[y][x] = ' '
		}
	}
	return c
}

func (c *canvas) set(x, y int, value rune) {
	if x >= 0 && x < c.width && y >= 0 && y < c.height {
		c.cells[y][x] = value
	}
}

func (c *canvas) connector(x, y int, value rune) {
	if x < 0 || x >= c.width || y < 0 || y >= c.height {
		return
	}
	existing := c.cells[y][x]
	if strings.ContainsRune("<>^v", value) {
		c.cells[y][x] = value
		return
	}
	if existing == ' ' || existing == value {
		c.cells[y][x] = value
		return
	}
	if strings.ContainsRune("<>^v", existing) {
		return
	}
	c.cells[y][x] = '+'
}

func (c *canvas) overlay(other *canvas, left, top int) {
	for y := 0; y < other.height; y++ {
		for x := 0; x < other.width; x++ {
			c.set(left+x, top+y, other.cells[y][x])
		}
	}
}

func (c *canvas) String() string {
	lines := make([]string, c.height)
	for y := range c.cells {
		lines[y] = strings.TrimRight(string(c.cells[y]), " ")
	}
	return strings.Join(lines, "\n")
}

func drawBorder(c *canvas) {
	if c.width < 2 || c.height < 2 {
		return
	}
	for x := 1; x < c.width-1; x++ {
		c.set(x, 0, '-')
		c.set(x, c.height-1, '-')
	}
	for y := 1; y < c.height-1; y++ {
		c.set(0, y, '|')
		c.set(c.width-1, y, '|')
	}
	c.set(0, 0, '+')
	c.set(c.width-1, 0, '+')
	c.set(0, c.height-1, '+')
	c.set(c.width-1, c.height-1, '+')
}
