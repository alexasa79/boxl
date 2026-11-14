# boxl

`boxl` is a small Go library and command-line renderer for the box language.
It tokenizes and parses source into an AST, collects global box declarations in
a first rendering pass, then performs constraint layout and rendering in a
second pass.

```text
a["request"] -> b["response"]
```

renders a diagram to `out.png` by default.

## Layout constraints

`->` only declares connections. Whitespace describes relative placement:

```text
a["input"] -> b["process"]

a b
```

Boxes on one line are arranged from left to right. Consecutive layout lines
form rows, with boxes in matching columns arranged from top to bottom. Thus
`a b` is shorthand for `a@then b`, while `a` followed by `b` on the next line
is shorthand for `a@above b`. A blank line or the end of the containing diagram
ends a whitespace layout description. Arrow statements are not layout rows and
separate adjacent layout descriptions.

```text
a b
c d
```

The example above forms a two-row, two-column layout.

The explicit `@` form remains available. Constraints are read relative to the
box before `@`:

- `then d` and `left-of d` place `d` to its right.
- `right-of d` places it to the right of `d`.
- `above d` and `below d` place it vertically relative to `d`.

Horizontal constraints align box centers vertically; vertical constraints
align them horizontally. Spacing and all unconstrained placement remain
automatic. Multiple declarations of a box accumulate constraints, and
constraints may only refer to boxes in the same container.

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
  text with nested content, multiple containers for one box, boundary-crossing
  connections or constraints, and contradictory layout constraints are
  reported as errors.
- Diagrams use compact 40x40 pixel boxes when they have no text; labeled leaves
  share the same dimensions. Container boxes expand recursively around their contents.
