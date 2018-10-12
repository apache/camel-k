package blush

import (
	"fmt"
	"strconv"
	"strings"
)

// BgLevel is the colour value of R, G, or B when the colour is shown in the
// background.
const BgLevel = 70

// These are colour settings. NoRGB results in no colouring in the terminal.
var (
	NoRGB     = RGB{-1, -1, -1}
	FgRed     = RGB{255, 0, 0}
	FgBlue    = RGB{0, 0, 255}
	FgGreen   = RGB{0, 255, 0}
	FgBlack   = RGB{0, 0, 0}
	FgWhite   = RGB{255, 255, 255}
	FgCyan    = RGB{0, 255, 255}
	FgMagenta = RGB{255, 0, 255}
	FgYellow  = RGB{255, 255, 0}
	BgRed     = RGB{BgLevel, 0, 0}
	BgBlue    = RGB{0, 0, BgLevel}
	BgGreen   = RGB{0, BgLevel, 0}
	BgBlack   = RGB{0, 0, 0}
	BgWhite   = RGB{BgLevel, BgLevel, BgLevel}
	BgCyan    = RGB{0, BgLevel, BgLevel}
	BgMagenta = RGB{BgLevel, 0, BgLevel}
	BgYellow  = RGB{BgLevel, BgLevel, 0}
)

// Some stock colours. There will be no colouring when NoColour is used.
var (
	NoColour = Colour{NoRGB, NoRGB}
	Red      = Colour{FgRed, NoRGB}
	Blue     = Colour{FgBlue, NoRGB}
	Green    = Colour{FgGreen, NoRGB}
	Black    = Colour{FgBlack, NoRGB}
	White    = Colour{FgWhite, NoRGB}
	Cyan     = Colour{FgCyan, NoRGB}
	Magenta  = Colour{FgMagenta, NoRGB}
	Yellow   = Colour{FgYellow, NoRGB}
)

//DefaultColour is the default colour if no colour is set via arguments.
var DefaultColour = Blue

// RGB represents colours that can be printed in terminals. R, G and B should be
// between 0 and 255.
type RGB struct {
	R, G, B int
}

// Colour is a pair of RGB colours for foreground and background.
type Colour struct {
	Foreground RGB
	Background RGB
}

// Colourise wraps the input between colours.
func Colourise(input string, c Colour) string {
	if c.Background == NoRGB && c.Foreground == NoRGB {
		return input
	}

	var fg, bg string
	if c.Foreground != NoRGB {
		fg = foreground(c.Foreground)
	}
	if c.Background != NoRGB {
		bg = background(c.Background)
	}
	return fg + bg + input + unformat()
}

func foreground(c RGB) string {
	return fmt.Sprintf("\033[38;5;%dm", colour(c.R, c.G, c.B))
}

func background(c RGB) string {
	return fmt.Sprintf("\033[48;5;%dm", colour(c.R, c.G, c.B))
}

func unformat() string {
	return "\033[0m"
}

func colour(red, green, blue int) int {
	return 16 + baseColor(red, 36) + baseColor(green, 6) + baseColor(blue, 1)
}

func baseColor(value int, factor int) int {
	return int(6*float64(value)/256) * factor
}

func colorFromArg(colour string) Colour {
	if strings.HasPrefix(colour, "#") {
		return hexColour(colour)
	}
	if grouping.MatchString(colour) {
		if c := colourGroup(colour); c != NoColour {
			return c
		}
	}
	return stockColour(colour)
}

func colourGroup(colour string) Colour {
	g := grouping.FindStringSubmatch(colour)
	group, err := strconv.Atoi(g[2])
	if err != nil {
		return NoColour
	}
	c := stockColour(g[1])
	switch group % 8 {
	case 0:
		c.Background = BgRed
	case 1:
		c.Background = BgBlue
	case 2:
		c.Background = BgGreen
	case 3:
		c.Background = BgBlack
	case 4:
		c.Background = BgWhite
	case 5:
		c.Background = BgCyan
	case 6:
		c.Background = BgMagenta
	case 7:
		c.Background = BgYellow
	}
	return c
}

func stockColour(colour string) Colour {
	c := DefaultColour
	switch colour {
	case "r", "red":
		c = Red
	case "b", "blue":
		c = Blue
	case "g", "green":
		c = Green
	case "bl", "black":
		c = Black
	case "w", "white":
		c = White
	case "cy", "cyan":
		c = Cyan
	case "mg", "magenta":
		c = Magenta
	case "yl", "yellow":
		c = Yellow
	case "no-colour", "no-color":
		c = NoColour
	}
	return c
}

func hexColour(colour string) Colour {
	var r, g, b int
	colour = strings.TrimPrefix(colour, "#")
	switch len(colour) {
	case 3:
		c := strings.Split(colour, "")
		r = getInt(c[0] + c[0])
		g = getInt(c[1] + c[1])
		b = getInt(c[2] + c[2])
	case 6:
		c := strings.Split(colour, "")
		r = getInt(c[0] + c[1])
		g = getInt(c[2] + c[3])
		b = getInt(c[4] + c[5])
	default:
		return DefaultColour
	}
	for _, n := range []int{r, g, b} {
		if n < 0 {
			return DefaultColour
		}
	}
	return Colour{RGB{R: r, G: g, B: b}, NoRGB}
}

// getInt returns a number between 0-255 from a hex code. If the hex is not
// between 00 and ff, it returns -1.
func getInt(hex string) int {
	d, err := strconv.ParseInt("0x"+hex, 0, 64)
	if err != nil || d > 255 || d < 0 {
		return -99
	}
	return int(d)
}
