package boxl

import (
	"errors"
	"fmt"
	"image/color"
	"strconv"
	"strings"
	"unicode"
)

type colorExpressionParser struct {
	source string
	offset int
}

func parseColorExpression(source string) (Expression, error) {
	p := colorExpressionParser{source: strings.TrimSpace(source)}
	expression, err := p.parse()
	if err != nil {
		return nil, err
	}
	p.space()
	if p.offset != len(p.source) {
		return nil, fmt.Errorf("unexpected text in color expression %q", p.source[p.offset:])
	}
	return expression, nil
}

func (p *colorExpressionParser) space() {
	for p.offset < len(p.source) && unicode.IsSpace(rune(p.source[p.offset])) {
		p.offset++
	}
}

func (p *colorExpressionParser) parse() (Expression, error) {
	p.space()
	start := p.offset
	for p.offset < len(p.source) && !strings.ContainsRune("(), \t\r\n", rune(p.source[p.offset])) {
		p.offset++
	}
	word := p.source[start:p.offset]
	if word == "" {
		return nil, fmt.Errorf("expected a color, variable, or function")
	}
	p.space()
	if p.offset < len(p.source) && p.source[p.offset] == '(' {
		call := CallExpression{Name: strings.ToLower(word)}
		p.offset++
		p.space()
		if p.offset < len(p.source) && p.source[p.offset] == ')' {
			p.offset++
			return call, nil
		}
		for {
			argument, err := p.parse()
			if err != nil {
				return nil, err
			}
			call.Arguments = append(call.Arguments, argument)
			p.space()
			if p.offset >= len(p.source) {
				return nil, fmt.Errorf("expected ) after %s arguments", call.Name)
			}
			separator := p.source[p.offset]
			p.offset++
			if separator == ')' {
				return call, nil
			}
			if separator != ',' {
				return nil, fmt.Errorf("expected , or ) in %s", call.Name)
			}
		}
	}
	if strings.HasPrefix(word, "$") {
		return VariableExpression{Name: word[1:]}, nil
	}
	if number, err := strconv.Atoi(word); err == nil {
		return IntegerLiteral{Value: number}, nil
	}
	literal, err := parseColorLiteral(word)
	if err != nil {
		return nil, err
	}
	return ColorLiteral{Value: literal}, nil
}

type colorValue struct {
	kind   int
	number int
	color  color.NRGBA
	pair   palettePair
	index  int
}

const (
	integerValue = iota
	literalColorValue
	palettePairValue
)

var errPaletteRequired = errors.New("an active palette is required for palette variables and nextcolors()")

func evaluateColorExpression(expression Expression, state *paletteState) (colorValue, error) {
	switch expression := expression.(type) {
	case ColorLiteral:
		return colorValue{kind: literalColorValue, color: expression.Value}, nil
	case IntegerLiteral:
		return colorValue{kind: integerValue, number: expression.Value}, nil
	case VariableExpression:
		number, err := strconv.Atoi(strings.TrimPrefix(expression.Name, "color"))
		if !strings.HasPrefix(expression.Name, "color") || err != nil || number < 1 || number > 16 || expression.Name != "color"+strconv.Itoa(number) {
			return colorValue{}, fmt.Errorf("unknown variable $%s; expected $color1 through $color16", expression.Name)
		}
		if state == nil || state.palette == nil {
			return colorValue{}, errPaletteRequired
		}
		return colorValue{kind: palettePairValue, pair: state.palette[number-1], index: number - 1}, nil
	case CallExpression:
		switch expression.Name {
		case "nextcolors":
			if len(expression.Arguments) != 0 {
				return colorValue{}, fmt.Errorf("nextcolors() takes no arguments")
			}
			if state == nil || state.palette == nil {
				return colorValue{}, errPaletteRequired
			}
			state.index = (state.index + 1) % len(state.palette)
			return colorValue{kind: palettePairValue, pair: state.palette[state.index], index: state.index}, nil
		case "rgb":
			if len(expression.Arguments) != 3 {
				return colorValue{}, fmt.Errorf("rgb requires three integers between 0 and 255")
			}
			var channels [3]uint8
			for i, argument := range expression.Arguments {
				value, err := evaluateColorExpression(argument, state)
				if err != nil {
					return colorValue{}, err
				}
				if value.kind != integerValue || value.number < 0 || value.number > 255 {
					return colorValue{}, fmt.Errorf("rgb requires three integers between 0 and 255")
				}
				channels[i] = uint8(value.number)
			}
			return colorValue{kind: literalColorValue, color: color.NRGBA{R: channels[0], G: channels[1], B: channels[2], A: 255}}, nil
		default:
			return colorValue{}, fmt.Errorf("unknown function %q", expression.Name)
		}
	default:
		return colorValue{}, fmt.Errorf("unsupported color expression")
	}
}

func parseColor(value string) (color.NRGBA, error) {
	expression, err := parseColorExpression(value)
	if err != nil {
		return color.NRGBA{}, err
	}
	result, err := evaluateColorExpression(expression, nil)
	if err != nil {
		return color.NRGBA{}, err
	}
	if result.kind != literalColorValue {
		return color.NRGBA{}, fmt.Errorf("expected a color")
	}
	return result.color, nil
}

func parseColorDirective(value string, position Position, pair bool) (*ColorDirective, error) {
	expression, err := parseColorExpression(value)
	if err != nil {
		return nil, err
	}
	directive := &ColorDirective{Expression: expression, Position: position}
	result, err := evaluateColorExpression(expression, nil)
	if errors.Is(err, errPaletteRequired) {
		return directive, nil
	}
	if err != nil {
		return nil, err
	}
	if pair {
		return nil, fmt.Errorf("expected a palette pair, such as $color3 or nextcolors()")
	}
	if result.kind != literalColorValue {
		return nil, fmt.Errorf("expected a color")
	}
	directive.Color = result.color
	return directive, nil
}
