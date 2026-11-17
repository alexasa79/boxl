package boxl

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"

	"golang.org/x/image/colornames"
)

func parseColorLiteral(value string) (color.NRGBA, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "transparent" {
		return color.NRGBA{}, nil
	}
	if named, ok := colornames.Map[value]; ok {
		return color.NRGBA{R: named.R, G: named.G, B: named.B, A: named.A}, nil
	}
	if strings.HasPrefix(value, "#") {
		hex := value[1:]
		if len(hex) == 3 {
			hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
		}
		if len(hex) == 6 {
			hex += "ff"
		}
		if len(hex) != 8 {
			return color.NRGBA{}, fmt.Errorf("expected #RGB, #RRGGBB, or #RRGGBBAA")
		}
		packed, err := strconv.ParseUint(hex, 16, 32)
		if err != nil {
			return color.NRGBA{}, fmt.Errorf("invalid hexadecimal color %q", value)
		}
		return color.NRGBA{R: uint8(packed >> 24), G: uint8(packed >> 16), B: uint8(packed >> 8), A: uint8(packed)}, nil
	}
	return color.NRGBA{}, fmt.Errorf("unknown color %q; expected a name, rgb(R,G,B), #RGB, #RRGGBB, or #RRGGBBAA", value)
}

func (p *Parser) parseDirective(diagram *Diagram) error {
	name, value, _ := strings.Cut(p.current.Value, ":")
	switch name {
	case "palette", "theme", "colors", "colorpair":
		value = strings.TrimSpace(value)
		if strings.HasPrefix(value, "$") || strings.Contains(value, "(") {
			if diagram.Style.Pair != nil {
				return p.errorf(p.current, "only one palette pair directive is allowed per box or chart")
			}
			directive, err := parseColorDirective(value, p.current.Position, true)
			if err != nil {
				return p.errorf(p.current, "invalid #%s pair: %v", name, err)
			}
			diagram.Style.Pair = directive
		} else {
			if lookupPalette(value) == nil {
				return p.errorf(p.current, "unknown palette %q; expected %s", value, strings.Join(paletteNames(), ", "))
			}
			if diagram.Style.Palette != nil {
				return p.errorf(p.current, "only one palette directive is allowed per box or chart")
			}
			diagram.Style.Palette = &PaletteDirective{Name: strings.ToLower(value), Position: p.current.Position}
		}
		return p.advance()
	case "dir", "layout", "direction":
		if diagram.Layout != nil {
			return p.errorf(p.current, "only one layout directive is allowed per diagram")
		}
		layout, err := p.parseLayoutDirective()
		if err != nil {
			return err
		}
		diagram.Layout = layout
		return nil
	case "bg", "fg":
		var target **ColorDirective
		if name == "bg" {
			target = &diagram.Style.Background
		} else {
			target = &diagram.Style.Foreground
		}
		if *target != nil {
			return p.errorf(p.current, "only one #%s directive is allowed per box or chart", name)
		}
		directive, err := parseColorDirective(value, p.current.Position, false)
		if err != nil {
			return p.errorf(p.current, "invalid #%s color: %v", name, err)
		}
		*target = directive
		return p.advance()
	default:
		return p.errorf(p.current, "unknown directive %q", name)
	}
}
