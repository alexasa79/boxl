# boxl

`boxl` is a small Go library and command-line renderer for the box language.
It tokenizes and parses source into an AST, collects global box declarations in
a first rendering pass, then performs horizontal layout and ASCII rendering in a
second pass.

```text
a["request"] -> b["response"]
```

renders as:

```text
+----------+     +----------+
| request  | --> | response |
+----------+     +----------+
```

## Command line

```sh
make
echo 'a -> b' | ./boxl
./boxl diagram.boxl
./boxl -e 'a["hello"] -> b'
./boxl -kitty -e 'a["hello"] -> b'
./boxl -o diagram.txt diagram.boxl
```

`-kitty` draws geometric boxes, arrows, nesting, and labels into a PNG, then
transmits it directly with the Kitty graphics protocol. It does not rasterize
the ASCII output. The switch works with stdin, a source file, and `-e`; the
terminal must support the protocol.

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
text, err := boxl.Render(source)
png, err := boxl.RenderPNG(source)
err = boxl.Transform(reader, writer)
err = boxl.TransformKitty(reader, writer)
err = boxl.TransformKittyAtCellSize(reader, writer, boxl.KittyCellSize{Width: 10, Height: 20})
```

Use `NewRenderer` with `RenderOptions` when custom spacing is needed.

## Current language decisions

- Identifiers start with a Unicode letter or `_`; later characters may also be
  digits, `-`, or `.`.
- Single- and double-quoted text may span physical lines. Line breaks and
  indentation are preserved, and Go-style escapes remain supported.
- Repeating the same text declaration is allowed. Conflicting text, combining
  text with nested content, multiple containers for one box, boundary-crossing
  connections, and connection cycles are reported as errors.
- Text output gives all leaf boxes the same dimensions. Graphical output uses
  compact 40x40 pixel boxes when they have no text; labeled leaves share the
  same dimensions. Container boxes expand recursively around their contents.
