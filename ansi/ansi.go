package ansi

import (
	"fmt"
)

const ESC byte = 0x1B

const (
	BLACK = iota
	RED
	GREEN
	YELLOW
	BLUE
	MAGENTA
	CYAN
	WHITE
)

var fg, bg = 0, 7

type pair struct {
	a, b int
}

var state map[pair]pair = make(map[pair]pair)

func PutRune(r rune, x, y int) {
	if (state[pair{x, y}] != pair{fg, bg}) {
		fmt.Printf("%c[%d;%dH", ESC, y + 1, x + 1)
		fmt.Print(colorize(r))
		state[pair{x, y}] = pair{fg, bg}
	}
}

func Print(a ...interface{}) {
	fmt.Printf("%c[H", ESC)
	fmt.Print(a)
}

func Printf(format string, a ...interface{}) {
	s := fmt.Sprintf(format, a)
	fmt.Printf("%c[H", ESC)
	fmt.Print(s)
}

func ClearScreen() {
	fmt.Printf("%c[2J", ESC)
	fmt.Printf("%c[H", ESC)
}

func HideCursor() {
	fmt.Printf("%c[?25l", ESC)
}

func ShowCursor() {
	fmt.Printf("%c[?25h", ESC)
}

func DefineColor(color int, rgb uint32) {
	color &= 0xF
	rgb &= 0xFFFFFF

	fmt.Printf("%c]P%X%06X%c\\", ESC, color, rgb, ESC)
}

func SetForeground(color int) {
	fg = color + 30
}

func SetBackground(color int) {
	bg = color + 40
}

func colorize(r rune) string {
	return fmt.Sprintf("%c[%d;%dm%c", ESC, fg, bg, r)
}
