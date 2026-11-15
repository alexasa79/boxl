# boxl

`boxl` is a small Go library and command-line renderer for the box language.
It tokenizes and parses source into an AST, collects global box declarations in
a first rendering pass, then performs layout and rendering in a
second pass.

```text
a["request"] -> b["response"]
```

renders a diagram to `out.png` by default.

## Layout

Boxes are arranged from left to right by default, equivalent to `#dir:row`.
Boxes are ordered by their first appearance in the source, and repeated
references occupy one cell. Use `->` to connect boxes.

Use a directive to choose the layout of a chart or a containing box:

| Directive | Arrangement |
| --- | --- |
| `#dir:row` | Left to right, the default |
| `#dir:col` or `#dir:column` | Top to bottom |
| `#dir:grid:2x3` | Two rows and three columns, filled left to right, then top to bottom |

`#layout` and `#direction` are aliases for `#dir`. Grid dimensions must be
positive integers. Unused cells are allowed; exceeding the grid's capacity is
an error. Rows and columns grow to fit their contents, with boxes centered
within their cells.

```text
#dir:grid:2x2
a b
c d
```

This arranges `a` and `b` in the first row, with `c` and `d` in the second.

Each diagram can have one directive, anywhere among its statements. A directive
applies to that entire diagram, without being inherited by nested boxes:

```text
#dir:col
upper[
    #dir:grid:2x2
    a b c d
]
lower[e f]
```

`upper` is above `lower`. Inside `upper`, the four boxes form a grid; inside
`lower`, `e` and `f` use the default row layout.

## Command line

```sh
make
echo 'a -> b' | ./boxl
./boxl diagram.boxl
./boxl -e 'a["hello"] -> b'
./boxl -kitty -e 'a["hello"] -> b'
./boxl -o diagram.png diagram.boxl
```

Without `-kitty`, boxl writes the diagram as a PNG to `out.png` in the current
directory. Use `-o` to choose a different output file.

`-kitty` draws geometric boxes, arrows, nesting, and labels into a PNG, then
transmits it directly with the Kitty graphics protocol. The switch works with
stdin, a source file, and `-e`; the terminal must support the protocol.
Graphical output uses a transparent
background with white boxes, arrows, and text for dark terminal backgrounds.

When writing directly to a terminal, boxl obtains its physical cell dimensions
from `TIOCGWINSZ`, with a `CSI 16 t` query as fallback, and renders the PNG at
the exact pixel size of the requested Kitty placement. If both methods are
unavailable, boxl omits cell sizing and displays the PNG at its native
resolution instead of asking the terminal to rescale it.

## Library

The module path is `boxl`, matching Veda's local-module convention. A sibling
project can use a `replace` directive while developing locally:

```go
replace boxl => ../boxl
```

The main entry points are:

```go
tokens, err := boxl.Tokenize(source)
diagram, err := boxl.Parse(source)
png, err := boxl.RenderPNG(source)
err = boxl.TransformKitty(reader, writer)
err = boxl.TransformKittyAtCellSize(reader, writer, boxl.KittyCellSize{Width: 10, Height: 20})
```

## Current language decisions

- Identifiers start with a Unicode letter or `_`; later characters may also be
  digits, `-`, or `.`.
- Single- and double-quoted text may span physical lines. Line breaks and
  indentation are preserved, and Go-style escapes remain supported.
- Repeating the same text declaration is allowed. Conflicting text, combining
  text with nested content, multiple containers for one box, connections that
  cross container boundaries, and conflicting layout directives are reported
  as errors.
- Diagrams use compact 40x40 pixel boxes when they have no text; labeled leaves
  share the same dimensions. Container boxes expand recursively around their contents.
