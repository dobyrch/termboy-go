package inputoutput

import (
	"github.com/dobyrch/termboy-go/ansi"
	"github.com/dobyrch/termboy-go/types"
)

type Display struct {
        Name                 string
        ScreenSizeMultiplier int
	offX                 int
	offY                 int
}

func (s *Display) init(title string, screenSizeMultiplier int) error {
        //TODO: use ScreenSizeMultiplier as an indicator of whether to use
        //TODO: left half block or top half block
        //TODO: Perhaps use escape code to set title of terminal?
	ansi.HideCursor()
	ansi.ClearScreen()
        ansi.DefineColor(0x0, 0x000000)
        ansi.DefineColor(0x4, 0x555555)
        ansi.DefineColor(0x6, 0xAAAAAA)
        ansi.DefineColor(0x7, 0xFFFFFF)
	//TODO: how should bright colors be dealt with?
        ansi.DefineColor(0x8, 0x000000)
        ansi.DefineColor(0xC, 0x555555)
        ansi.DefineColor(0xE, 0xAAAAAA)
        ansi.DefineColor(0xF, 0xFFFFFF)

        return nil
}

func (s *Display) drawFrame(screenData *types.Screen) {
        for y := 0; y < SCREEN_HEIGHT; y ++ {
                for x := 0; x < SCREEN_WIDTH; x += 2 {
                        c1 := screenData[y][x]
                        c2 := screenData[y][x+1]

			var fg, bg int

			switch c1.Red {
			case 0:
				fg = 0x0
			case 96:
				fg = 0x4
			case 196:
				fg = 0x6
			case 235:
				fg = 0x7
			}

			switch c2.Red {
			case 0:
				bg = 0x0
			case 96:
				bg = 0x4
			case 196:
				bg = 0x6
			case 235:
				bg = 0x7
			}

			ansi.SetForeground(fg)
			ansi.SetBackground(bg)
			ansi.PutRune('▌', x/2, y)
                }
        }
}

/*
func (s *Display) initOffset() {
	var x, y int

	// Move cursor to bottom right
	fmt.Printf("%c[1000B", ESC)
	fmt.Printf("%c[1000C", ESC)
	fmt.Printf("%c[6n", ESC)

	// Get current position
	cup := fmt.Sprintf("%c[%%d;%%dR", ESC)
	n, scanError := fmt.Scanf(cup, &y, &x)
	file, _ := os.Create("error.out")
	fmt.Fprintf(file, "%d: %s\n", n, scanError)
	file.Close()

	switch {
	case x > 160/2:
		s.offX = (x+1)/2 - 160/4
	case y > 144:
		s.offY = (y+1)/2 - 144/2
	default:
		//TODO: are struct members initialized to 0?
		s.offX = 0
		s.offY = 0
	}
}
*/

func (s *Display) CleanUp() {
	ansi.ClearScreen()
	ansi.ShowCursor()
}
