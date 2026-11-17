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
Multiple connection chains can share a line, as in `a -> b c -> a`.

Use a directive to choose the layout of a chart or a containing box:

| Directive | Arrangement |
| --- | --- |
| `#dir:row` | Left to right, the default |
| `#dir:col` or `#dir:column` | Top to bottom |
| `#dir:diag` or `#dir:diagonal` | Down and right |
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

Each diagram can have one layout directive, anywhere among its statements. It
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

Arrows leave a gap at both boxes. Forward connections use straight arrows when
the path is clear. Return connections and forward connections blocked by other
boxes curve around them. The renderer considers both sides and several bend
sizes, choosing routes together to minimize arrow crossings, overlapping paths,
and arrowhead collisions. Simpler, shorter routes break ties. The search has a
fixed computation budget and retains the best combination found. Arrowheads
follow the direction of the line or curve. Containing boxes expand to include
their arrows.

## Box appearance

Names starting with `_`, such as `_` or `_group`, hide the box border. These
containers keep their contents, padding, and layout space.

Use `#bg` to fill a box and `#fg` to color its border, text, and arrows. Place
the directives inside the box's brackets. Foreground colors carry into child
boxes unless a palette assigns a new pair at the next level. Background colors
apply to the box that declares them. At the top level, they style the chart.

```text
_[
    #bg:white #fg:black #dir:diag
    a["A" #bg:rgb(240, 240, 240)] -> b["B" #bg:#33669980]
]
```

Colors accept names such as `white`, `black`, and `red`; `rgb(R,G,B)` with
integer channels from 0 to 255; `#RGB`; `#RRGGBB`; and `#RRGGBBAA`. The last
two hexadecimal digits of `#RRGGBBAA` specify alpha, from `00` for transparent
to `FF` for opaque. The name `transparent` is also supported. Names and hex
digits are case-insensitive. Each box or chart can declare one `#bg` and one
`#fg` directive. Explicit background fills also apply to borderless containers.

## Palettes and color expressions

`#palette:pastel`, `#theme:pastel`, and `#colors:pastel` select the same palette.
The available palettes are:

| Palette | Colors |
| --- | --- |
| `pastel` | Mixed soft colors |
| `earth` | Moss, clay, ochre, and woodland colors |
| `ocean` | Blues, teals, sea greens, and indigo |

Each contains 16 pairs, with a pale background and a darker foreground for the
border and text. For example, `#theme:ocean` selects the ocean palette.
Top-level boxes start with pair 1. Siblings share the
current pair, and children start with the next pair. After pair 16, the palette
wraps to pair 1.

Use `#colors:$color3` to select both components of pair 3 for a box.
`#colorpair:$color3` is an alias. A change carries forward to later siblings
inside the same containing box:

```text
#palette:pastel
A[
    B["B"]
    C["C" #colors:$color5]
    E["E"]
]
D[F["F"]]
```

`A` and `D` use pair 1. `B` and `F` use pair 2. `C` selects pair 5, which `E`
keeps. Changes inside `A` stay in that sibling group. A palette declared inside
a box starts that box at pair 1 and applies to its descendants. Borderless
containers participate in palette depth and receive palette fills.
Switching palettes inside a box stays local to that box and its descendants.

`$color1` through `$color16` reference pairs in the active palette.
`#bg:$color3` selects pair 3's background, while `#fg:$color3` selects its
foreground. These lookups do not advance the palette. Literal `#bg` and `#fg`
overrides also carry forward to later siblings when a palette is active.

`nextcolors()` returns the next pair and advances the current sibling group's
palette position. Selecting a whole pair sets that position, so after
`#colors:$color5`, `#colors:nextcolors()` selects pair 6. Child groups advance
independently. Expressions within a box run in source order; each call advances
once. The `rgb(R,G,B)` function returns a literal color.

```text
#theme:pastel
a["A"]
b["B" #colors:nextcolors()]
c["C"]
```

`a` uses pair 1; `b` advances to pair 2; `c` keeps pair 2. Palette variables
and `nextcolors()` require an active palette. At chart level, palette selection
keeps the canvas transparent; use `#bg` to fill it.

The `pastel` pairs are:

| Variable | Border and text | Background |
| --- | --- | --- |
| `$color1` | `#326B53` | `#D8F2E4` |
| `$color2` | `#3E6B3A` | `#DEF2D8` |
| `$color3` | `#8D4F40` | `#F9DFD8` |
| `$color4` | `#805080` | `#F3E1F4` |
| `$color5` | `#66508C` | `#EBE2F7` |
| `$color6` | `#365F85` | `#DEEFFA` |
| `$color7` | `#756020` | `#FAF1C8` |
| `$color8` | `#8A4B70` | `#F9E1EE` |
| `$color9` | `#4B6670` | `#E1EAED` |
| `$color10` | `#2F6A65` | `#D9F1EE` |
| `$color11` | `#665C75` | `#EAE6EF` |
| `$color12` | `#8A3F52` | `#FBE4E8` |
| `$color13` | `#536B45` | `#E5EBDD` |
| `$color14` | `#8A512F` | `#FCE8D5` |
| `$color15` | `#72613E` | `#EFE8D5` |
| `$color16` | `#485A91` | `#E2E8FC` |

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
By default, graphical output uses a transparent
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
