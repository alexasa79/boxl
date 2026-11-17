package boxl

import (
	"image/color"
	"reflect"
	"sort"
	"strings"
)

type palettePair struct {
	foreground, background color.NRGBA
}

type colorPalette [16]palettePair

func opaqueColor(rgb uint32) color.NRGBA {
	return color.NRGBA{R: uint8(rgb >> 16), G: uint8(rgb >> 8), B: uint8(rgb), A: 255}
}

func opaquePair(foreground, background uint32) palettePair {
	return palettePair{foreground: opaqueColor(foreground), background: opaqueColor(background)}
}

var pastelPalette = colorPalette{
	opaquePair(0x365f85, 0xdeeffa), // Sky.
	opaquePair(0x3e6b3a, 0xdef2d8), // Mint.
	opaquePair(0x8d4f40, 0xf9dfd8), // Coral.
	opaquePair(0x4b6670, 0xe1eaed), // Mist.
	opaquePair(0x8a512f, 0xfce8d5), // Peach.
	opaquePair(0x665c75, 0xeae6ef), // Gray violet.
	opaquePair(0x756020, 0xfaf1c8), // Butter.
	opaquePair(0x66508c, 0xebe2f7), // Lavender.
	opaquePair(0x72613e, 0xefe8d5), // Sand.
	opaquePair(0x805080, 0xf3e1f4), // Lilac.
	opaquePair(0x326b53, 0xd8f2e4), // Seafoam.
	opaquePair(0x8a4b70, 0xf9e1ee), // Pink.
	opaquePair(0x2f6a65, 0xd9f1ee), // Aqua.
	opaquePair(0x8a3f52, 0xfbe4e8), // Rose.
	opaquePair(0x485a91, 0xe2e8fc), // Periwinkle.
	opaquePair(0x536b45, 0xe5ebdd), // Sage.
}

var earthPalette = colorPalette{
	opaquePair(0x3e637d, 0xd3e1ed), // Dusty blue.
	opaquePair(0x456533, 0xdde9c6), // Moss.
	opaquePair(0x9a5141, 0xf3d0c8), // Terracotta.
	opaquePair(0x6e5569, 0xe9d9e9), // Heather.
	opaquePair(0x85661f, 0xf0e4ba), // Ochre.
	opaquePair(0x3d654d, 0xd4e8dc), // Pine.
	opaquePair(0x775265, 0xebd3df), // Mulberry.
	opaquePair(0x466570, 0xd5e5eb), // Slate.
	opaquePair(0x52713d, 0xd9ebc7), // Lichen.
	opaquePair(0x92563f, 0xf0d4c5), // Clay.
	opaquePair(0x4f6b5b, 0xd9e8de), // Sage.
	opaquePair(0x6d527c, 0xe8cfed), // Flint.
	opaquePair(0x7c684a, 0xf0e5c9), // Sandstone.
	opaquePair(0x496478, 0xd8e3ed), // Riverstone.
	opaquePair(0x596b36, 0xe0e8c8), // Olive.
	opaquePair(0x83513f, 0xeecabd), // Chestnut.
}

var oceanPalette = colorPalette{
	opaquePair(0x124f78, 0x7ec8f2), // Blue.
	opaquePair(0x1e542c, 0x88d4a2), // Kelp.
	opaquePair(0x722e24, 0xf29a8b), // Coral.
	opaquePair(0x453174, 0xb4a3e8), // Twilight.
	opaquePair(0x71520b, 0xe8c96a), // Sand.
	opaquePair(0x0a514e, 0x67cfc3), // Lagoon.
	opaquePair(0x742348, 0xeca0c7), // Shell.
	opaquePair(0x23366f, 0x8fa5ec), // Indigo.
	opaquePair(0x1a5238, 0x80d6b0), // Sea glass.
	opaquePair(0x6d3d14, 0xedb170), // Sunset.
	opaquePair(0x0b5864, 0x6bd1df), // Aqua.
	opaquePair(0x4f226f, 0xc897e0), // Urchin.
	opaquePair(0x5f4707, 0xd8c66a), // Dune.
	opaquePair(0x113d5c, 0x70afca), // Deep sea.
	opaquePair(0x445a19, 0xbdd47a), // Algae.
	opaquePair(0x6e2a18, 0xe99278), // Starfish.
}

var availablePalettes = map[string]*colorPalette{
	"pastel": &pastelPalette,
	"earth":  &earthPalette,
	"ocean":  &oceanPalette,
}

func lookupPalette(name string) *colorPalette {
	return availablePalettes[strings.ToLower(name)]
}

func paletteNames() []string {
	names := make([]string, 0, len(availablePalettes))
	for name := range availablePalettes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

type paletteState struct {
	palette *colorPalette
	index   int
}

type resolvedBoxColors struct {
	background, foreground color.NRGBA
}

func (p palettePair) colors() resolvedBoxColors {
	return resolvedBoxColors{background: p.background, foreground: p.foreground}
}

func directiveExpression(directive *ColorDirective) Expression {
	if directive.Expression != nil {
		return directive.Expression
	}
	return ColorLiteral{Value: directive.Color}
}

func sameColorDirective(a, b *ColorDirective) bool {
	first, firstErr := evaluateColorExpression(directiveExpression(a), nil)
	second, secondErr := evaluateColorExpression(directiveExpression(b), nil)
	if firstErr == nil && secondErr == nil {
		return first == second
	}
	return reflect.DeepEqual(directiveExpression(a), directiveExpression(b))
}

func applyColorStyle(style BoxStyle, colors resolvedBoxColors, state *paletteState) (resolvedBoxColors, error) {
	type assignment struct {
		kind      string
		directive *ColorDirective
	}
	var assignments []assignment
	for _, item := range []assignment{{"bg", style.Background}, {"fg", style.Foreground}, {"pair", style.Pair}} {
		if item.directive != nil {
			assignments = append(assignments, item)
		}
	}
	sort.SliceStable(assignments, func(i, j int) bool {
		return assignments[i].directive.Position.Offset < assignments[j].directive.Position.Offset
	})
	for _, item := range assignments {
		value, err := evaluateColorExpression(directiveExpression(item.directive), state)
		if err != nil {
			return colors, modelError(item.directive.Position, "%v", err)
		}
		switch item.kind {
		case "pair":
			if value.kind != palettePairValue {
				return colors, modelError(item.directive.Position, "expected a palette pair")
			}
			colors = value.pair.colors()
			state.index = value.index
		case "bg", "fg":
			var selected color.NRGBA
			switch value.kind {
			case palettePairValue:
				selected = value.pair.background
				if item.kind == "fg" {
					selected = value.pair.foreground
				}
			case literalColorValue:
				selected = value.color
			default:
				return colors, modelError(item.directive.Position, "expected a color")
			}
			if item.kind == "bg" {
				colors.background = selected
			} else {
				colors.foreground = selected
			}
		}
	}
	return colors, nil
}

func (m *model) resolveColorStyles() error {
	m.resolvedStyles = make(map[*declaredBox]resolvedBoxColors)
	root := resolvedBoxColors{foreground: opaqueColor(0xffffff)}
	state := paletteState{index: -1}
	if directive := m.styles[nil].Palette; directive != nil {
		state.palette = lookupPalette(directive.Name)
		if state.palette == nil {
			return modelError(directive.Position, "unknown palette %q", directive.Name)
		}
		root.foreground = state.palette[0].foreground
	}
	var err error
	root, err = applyColorStyle(m.styles[nil], root, &state)
	if err != nil {
		return err
	}
	m.resolvedStyles[nil] = root
	return m.resolveChildColors(nil, state, root)
}

func (m *model) resolveChildColors(scope *declaredBox, parent paletteState, parentColors resolvedBoxColors) error {
	level := parent
	current := resolvedBoxColors{foreground: parentColors.foreground}
	if level.palette != nil {
		level.index = (level.index + 1) % len(level.palette)
		current = level.palette[level.index].colors()
	}
	for _, box := range m.children(scope) {
		state, colors := level, current
		style := m.styles[box]
		if directive := style.Palette; directive != nil {
			state = paletteState{palette: lookupPalette(directive.Name), index: 0}
			if state.palette == nil {
				return modelError(directive.Position, "unknown palette %q", directive.Name)
			}
			colors = state.palette[0].colors()
		}
		var err error
		colors, err = applyColorStyle(style, colors, &state)
		if err != nil {
			return err
		}
		m.resolvedStyles[box] = colors
		if level.palette != nil && state.palette == level.palette {
			// Child evaluation receives a copy, so deeper levels cannot advance this one.
			current, level.index = colors, state.index
		}
		if err := m.resolveChildColors(box, state, colors); err != nil {
			return err
		}
	}
	return nil
}
