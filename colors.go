package boxl

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"

	"golang.org/x/image/colornames"
)

func parseColor(value string) (color.NRGBA, error) {
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
	if strings.HasPrefix(value, "rgb(") && strings.HasSuffix(value, ")") {
		parts := strings.Split(value[4:len(value)-1], ",")
		if len(parts) != 3 {
			return color.NRGBA{}, fmt.Errorf("rgb requires three integers between 0 and 255")
		}
		var channels [3]uint8
		for i, part := range parts {
			channel, err := strconv.ParseUint(strings.TrimSpace(part), 10, 8)
			if err != nil {
				return color.NRGBA{}, fmt.Errorf("rgb requires three integers between 0 and 255")
			}
			channels[i] = uint8(channel)
		}
		return color.NRGBA{R: channels[0], G: channels[1], B: channels[2], A: 255}, nil
	}
	return color.NRGBA{}, fmt.Errorf("unknown color %q; expected a name, rgb(R,G,B), #RGB, #RRGGBB, or #RRGGBBAA", value)
}

func (p *Parser) parseDirective(diagram *Diagram) error {
	name, value, _ := strings.Cut(p.current.Value, ":")
	switch name {
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
		parsed, err := parseColor(value)
		if err != nil {
			return p.errorf(p.current, "invalid #%s color: %v", name, err)
		}
		*target = &ColorDirective{Color: parsed, Position: p.current.Position}
		return p.advance()
	default:
		return p.errorf(p.current, "unknown directive %q", name)
	}
}
